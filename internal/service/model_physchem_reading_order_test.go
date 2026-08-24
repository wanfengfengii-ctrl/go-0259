package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

func TestModel_DeviceReadingsAdvanceAfterPersistingFinalPhysChem(t *testing.T) {
	acceptedMoisture := func(op string, pos domain.BagPosition, value string) DeviceReadingRequest {
		return DeviceReadingRequest{
			Operation: domain.OperationID(op), Generation: 1,
			DeviceType: domain.DeviceMoisture, DeviceID: domain.DeviceID("moist-" + string(pos)),
			Metric: domain.MetricMoisture, Value: value, Position: pos, DayAge: 1,
		}
	}
	acceptedPH := func(op string, pos domain.BagPosition, value string) DeviceReadingRequest {
		return DeviceReadingRequest{
			Operation: domain.OperationID(op), Generation: 1,
			DeviceType: domain.DevicePHMeter, DeviceID: domain.DeviceID("ph-" + string(pos)),
			Metric: domain.MetricPH, Value: value, Position: pos, DayAge: 1,
		}
	}

	cases := []struct {
		name                     string
		before                   []DeviceReadingRequest
		final                    DeviceReadingRequest
		finalScript              domain.AttemptResult
		wantResult               domain.AttemptResult
		wantRetryCount           int
		wantResponseState        inspection.TaskState
		wantTaskState            inspection.TaskState
		wantReadingCount         int
		wantAcceptedReadingCount int
		checkPhysChemCollected   bool
		wantPhysChemCollected    bool
		wantPendingRetries       int
		wantFinalizeError        domain.ErrorCode
	}{
		{
			name: "final accepted pH is persisted before deciding collection is complete",
			before: []DeviceReadingRequest{
				acceptedMoisture("m-p1", "P1", "65.0"),
				acceptedPH("ph-p1", "P1", "6.00"),
				acceptedMoisture("m-p2", "P2", "65.0"),
			},
			final:                    acceptedPH("ph-p2", "P2", "6.00"),
			wantResult:               domain.AttemptAccepted,
			wantRetryCount:           1,
			wantResponseState:        inspection.StatePendingReview,
			wantTaskState:            inspection.StatePendingReview,
			wantReadingCount:         4,
			wantAcceptedReadingCount: 4,
			checkPhysChemCollected:   true,
			wantPhysChemCollected:    true,
		},
		{
			name: "ordinary incomplete bag position set stays in physchem verification",
			before: []DeviceReadingRequest{
				acceptedMoisture("m-p1", "P1", "65.0"),
				acceptedPH("ph-p1", "P1", "6.00"),
			},
			final:                    acceptedMoisture("m-p2", "P2", "65.0"),
			wantResult:               domain.AttemptAccepted,
			wantRetryCount:           1,
			wantResponseState:        inspection.StateVerifyingPhysChem,
			wantTaskState:            inspection.StateVerifyingPhysChem,
			wantReadingCount:         3,
			wantAcceptedReadingCount: 3,
			checkPhysChemCollected:   true,
			wantPhysChemCollected:    false,
		},
		{
			name:                   "device failure records retry work without a reading or state advance",
			final:                  acceptedMoisture("m-p1-timeout", "P1", "65.0"),
			finalScript:            domain.AttemptTimeout,
			wantResult:             domain.AttemptTimeout,
			wantRetryCount:         1,
			wantResponseState:      inspection.StateVerifyingPhysChem,
			wantTaskState:          inspection.StateVerifyingPhysChem,
			checkPhysChemCollected: true,
			wantPendingRetries:     1,
		},
		{
			name: "threshold failing accepted readings do not satisfy transfer finalization",
			before: []DeviceReadingRequest{
				acceptedMoisture("m-p1", "P1", "65.0"),
				acceptedPH("ph-p1", "P1", "6.00"),
				acceptedMoisture("m-p2", "P2", "75.0"),
			},
			final:                    acceptedPH("ph-p2", "P2", "6.00"),
			wantResult:               domain.AttemptAccepted,
			wantRetryCount:           1,
			wantReadingCount:         4,
			wantAcceptedReadingCount: 4,
			wantFinalizeError:        domain.CodeReadingOutOfRange,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestService(t)
			id := lockTask(t, s)
			advanceToObserving(t, s, id)

			for i, cell := range []struct {
				day           domain.DayAge
				pos           domain.BagPosition
				contamination int
			}{
				{1, "P1", 1},
				{1, "P2", 0},
				{2, "P1", 0},
				{2, "P2", 0},
			} {
				if _, err := s.Observation(id, ObservationRequest{
					Operation: domain.OperationID("obs-" + itoa(i)), Generation: 1,
					DayAge: cell.day, Position: cell.pos, MyceliumCoverage: "85.0",
					ContaminationCount: cell.contamination, Observer: "carol",
				}); err != nil {
					t.Fatalf("observation %v: %v", cell, err)
				}
			}
			if _, err := s.ContaminationRecheck(id, ContaminationRecheckRequest{
				Operation: "close-contamination", Generation: 1, RecheckGeneration: 1,
				Position: "P1", DayAge: 1, Well: "A1", Source: "molecular", Positive: false,
			}); err != nil {
				t.Fatalf("close contamination recheck: %v", err)
			}
			view, err := s.GetTask(id)
			if err != nil {
				t.Fatalf("get verifying task: %v", err)
			}
			if view.State != inspection.StateVerifyingPhysChem {
				t.Fatalf("state after contamination closure = %s, want verifying_physchem", view.State)
			}

			for _, req := range tc.before {
				if _, err := s.DeviceReading(id, req); err != nil {
					t.Fatalf("seed device reading %s/%s: %v", req.Position, req.Metric, err)
				}
			}
			if tc.finalScript != "" {
				s.scripts.Set(tc.final.DeviceType, tc.final.DeviceID, tc.final.ScriptSeq, tc.finalScript)
			}

			res, err := s.DeviceReading(id, tc.final)
			if err != nil {
				t.Fatalf("final device reading: %v", err)
			}
			if res.Result != tc.wantResult {
				t.Fatalf("device result = %s, want %s", res.Result, tc.wantResult)
			}
			if res.RetryCount != tc.wantRetryCount {
				t.Fatalf("retry count = %d, want %d", res.RetryCount, tc.wantRetryCount)
			}
			if tc.wantResponseState != "" && res.State != tc.wantResponseState {
				t.Fatalf("device-reading response state = %s, want %s", res.State, tc.wantResponseState)
			}

			detail, err := s.GetTaskDetail(id)
			if err != nil {
				t.Fatalf("get detail: %v", err)
			}
			if len(detail.Readings) != tc.wantReadingCount {
				t.Fatalf("detail readings = %d, want %d", len(detail.Readings), tc.wantReadingCount)
			}
			accepted := 0
			for _, r := range detail.Readings {
				if r.Accepted() {
					accepted++
				}
			}
			if accepted != tc.wantAcceptedReadingCount {
				t.Fatalf("accepted readings in detail = %d, want %d", accepted, tc.wantAcceptedReadingCount)
			}

			progress, err := s.Progress(id)
			if err != nil {
				t.Fatalf("progress: %v", err)
			}
			if tc.checkPhysChemCollected && progress.PhysChemCollected != tc.wantPhysChemCollected {
				t.Fatalf("physchem_collected = %t, want %t", progress.PhysChemCollected, tc.wantPhysChemCollected)
			}

			report, err := s.Report()
			if err != nil {
				t.Fatalf("report: %v", err)
			}
			if report.PendingRetries != tc.wantPendingRetries {
				t.Fatalf("pending retries = %d, want %d", report.PendingRetries, tc.wantPendingRetries)
			}

			if tc.wantTaskState != "" {
				view, err = s.GetTask(id)
				if err != nil {
					t.Fatalf("get task: %v", err)
				}
				if view.State != tc.wantTaskState {
					t.Fatalf("task state = %s, want %s", view.State, tc.wantTaskState)
				}
				list, err := s.ListTaskSummaries()
				if err != nil {
					t.Fatalf("list tasks: %v", err)
				}
				if len(list) != 1 {
					t.Fatalf("listed tasks = %d, want 1", len(list))
				}
				if list[0].State != tc.wantTaskState {
					t.Fatalf("task list state = %s, want %s", list[0].State, tc.wantTaskState)
				}
				if detail.State != string(tc.wantTaskState) {
					t.Fatalf("detail state = %s, want %s", detail.State, tc.wantTaskState)
				}
			}

			if tc.wantFinalizeError != "" {
				reviewApprove(t, s, id, "r1", "carol")
				reviewApprove(t, s, id, "r2", "dave")
				_, err := s.Finalize(id, FinalizeRequest{
					Operation: "final-transfer", Generation: 1, Conclusion: "transfer",
				})
				assertErrorCode(t, err, tc.wantFinalizeError)
			}
		})
	}
}
