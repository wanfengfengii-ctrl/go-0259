package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
)

func TestModel_DeviceScriptRegistryDeterminism(t *testing.T) {
	observingTask := func(t *testing.T) (*Service, domain.TaskID) {
		t.Helper()
		s := newTestService(t)
		id := lockTask(t, s)
		advanceToObserving(t, s, id)
		return s, id
	}
	submitReading := func(t *testing.T, s *Service, id domain.TaskID, req DeviceReadingRequest) DeviceReadingResult {
		t.Helper()
		res, err := s.DeviceReading(id, req)
		if err != nil {
			t.Fatalf("device reading %s/%s seq %d: %v", req.DeviceType, req.DeviceID, req.ScriptSeq, err)
		}
		return res
	}
	moistureReq := func(op domain.OperationID, dev domain.DeviceID, seq int) DeviceReadingRequest {
		return DeviceReadingRequest{
			Operation: op, Generation: 1, DeviceType: domain.DeviceMoisture, DeviceID: dev,
			Metric: domain.MetricMoisture, Value: "65.0", Position: "P1", DayAge: 1, ScriptSeq: seq,
		}
	}
	assertReadings := func(t *testing.T, s *Service, id domain.TaskID, want int) {
		t.Helper()
		detail, err := s.GetTaskDetail(id)
		if err != nil {
			t.Fatalf("task detail: %v", err)
		}
		if len(detail.Readings) != want {
			t.Fatalf("readings = %d, want %d: %+v", len(detail.Readings), want, detail.Readings)
		}
	}

	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "same script sequence remains configured across different operations",
			run: func(t *testing.T) {
				s, id := observingTask(t)
				s.Scripts().Set(domain.DeviceMoisture, "moist-1", 7, domain.AttemptTimeout)

				first := submitReading(t, s, id, moistureReq("moist-timeout-1", "moist-1", 7))
				second := submitReading(t, s, id, moistureReq("moist-timeout-2", "moist-1", 7))

				if first.Result != domain.AttemptTimeout || second.Result != domain.AttemptTimeout {
					t.Fatalf("results = %s, %s; want timeout, timeout", first.Result, second.Result)
				}
				if first.RetryCount != 1 || second.RetryCount != 2 {
					t.Fatalf("retry counts = %d, %d; want 1, 2", first.RetryCount, second.RetryCount)
				}
				assertReadings(t, s, id, 0)
				report, err := s.Report()
				if err != nil {
					t.Fatalf("report: %v", err)
				}
				if report.PendingRetries != 1 {
					t.Fatalf("pending retries = %d, want 1", report.PendingRetries)
				}
			},
		},
		{
			name: "unconfigured script sequence defaults to accepted",
			run: func(t *testing.T) {
				s, id := observingTask(t)

				res := submitReading(t, s, id, moistureReq("moist-unscripted", "moist-unscripted", 42))

				if res.Result != domain.AttemptAccepted {
					t.Fatalf("result = %s, want accepted", res.Result)
				}
				if res.RetryCount != 1 {
					t.Fatalf("retry count = %d, want 1", res.RetryCount)
				}
				assertReadings(t, s, id, 1)
			},
		},
		{
			name: "clear removes scripts for the selected device",
			run: func(t *testing.T) {
				s, id := observingTask(t)
				s.Scripts().Set(domain.DeviceMoisture, "moist-clear", 7, domain.AttemptTimeout)
				s.Scripts().Clear(domain.DeviceMoisture, "moist-clear")

				res := submitReading(t, s, id, moistureReq("moist-cleared", "moist-clear", 7))

				if res.Result != domain.AttemptAccepted {
					t.Fatalf("result = %s, want accepted", res.Result)
				}
				assertReadings(t, s, id, 1)
			},
		},
		{
			name: "script outcomes are isolated by device type and id",
			run: func(t *testing.T) {
				s, id := observingTask(t)
				s.Scripts().Set(domain.DeviceMoisture, "shared-id", 7, domain.AttemptTimeout)

				otherType := submitReading(t, s, id, DeviceReadingRequest{
					Operation: "ph-shared-id", Generation: 1, DeviceType: domain.DevicePHMeter, DeviceID: "shared-id",
					Metric: domain.MetricPH, Value: "6.00", Position: "P1", DayAge: 1, ScriptSeq: 7,
				})
				otherID := submitReading(t, s, id, moistureReq("moist-other-id", "other-id", 7))
				configured := submitReading(t, s, id, moistureReq("moist-shared-id", "shared-id", 7))

				if otherType.Result != domain.AttemptAccepted {
					t.Fatalf("other type result = %s, want accepted", otherType.Result)
				}
				if otherID.Result != domain.AttemptAccepted {
					t.Fatalf("other id result = %s, want accepted", otherID.Result)
				}
				if configured.Result != domain.AttemptTimeout {
					t.Fatalf("configured device result = %s, want timeout", configured.Result)
				}
				assertReadings(t, s, id, 2)
			},
		},
		{
			name: "operation replay remains idempotent for modeled operations",
			run: func(t *testing.T) {
				s := newTestService(t)
				id := lockTask(t, s)
				req := SamplingConfirmRequest{
					Operation: "sample-replay", Generation: 1, Person: "alice", Batch: "B-001",
					Positions: []domain.BagPosition{"P1", "P2"},
				}

				first, err := s.SamplingConfirm(id, req)
				if err != nil {
					t.Fatalf("first confirm: %v", err)
				}
				second, err := s.SamplingConfirm(id, req)
				if err != nil {
					t.Fatalf("replay confirm: %v", err)
				}
				if first.State != second.State || first.Confirmations != second.Confirmations {
					t.Fatalf("replay changed result: first=%+v second=%+v", first, second)
				}
				req.Person = "bob"
				_, err = s.SamplingConfirm(id, req)
				assertErrorCode(t, err, domain.CodeIdempotencyConflict)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}
