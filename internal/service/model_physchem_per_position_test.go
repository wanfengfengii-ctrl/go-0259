package service_test

import (
	"errors"
	"testing"

	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/service"
	"mycocycle-growbag-transfer-gate/internal/store"
)

func TestModel_FinalizeTransferRequiresPhysChemPerBagPosition(t *testing.T) {
	type reading struct {
		operation  domain.OperationID
		deviceType domain.DeviceType
		deviceID   domain.DeviceID
		metric     domain.Metric
		value      string
		position   domain.BagPosition
	}

	allReadings := []reading{
		{"m-p1", domain.DeviceMoisture, "moist-1", domain.MetricMoisture, "65.0", "P1"},
		{"ph-p1", domain.DevicePHMeter, "ph-1", domain.MetricPH, "6.00", "P1"},
		{"m-p2", domain.DeviceMoisture, "moist-1", domain.MetricMoisture, "65.0", "P2"},
		{"ph-p2", domain.DevicePHMeter, "ph-1", domain.MetricPH, "6.00", "P2"},
	}
	onlyP1Readings := allReadings[:2]

	lockRequest := func() catalog.LockRequest {
		return catalog.LockRequest{
			StrainRevision:    "strain-po-2024.03",
			SubstrateRevision: "sub-hw-01",
			SubstrateSummary:  "hardwood sawdust + wheat bran 78:20",
			SterilizerSummary: "run-A-2026-08-21",
			InoculationLine:   "line-1",
			BagBatch:          "B-001",
			BagPositions:      []domain.BagPosition{"P1", "P2"},
			RackID:            "R1",
			ProbeWindow:       catalog.ProbeWindow{ProbeID: "probe-1", Start: 1, End: 100},
			Schedule:          catalog.ScheduleTemplate{DayAges: []domain.DayAge{1, 2}},
			Thresholds: catalog.Thresholds{
				Contamination: domain.MustFixed(0, 0),
				MaturityMin:   domain.MustFixed(700, 1),
				MaturityMax:   domain.MustFixed(1000, 1),
				MoistureMin:   domain.MustFixed(600, 1),
				MoistureMax:   domain.MustFixed(700, 1),
				PHMin:         domain.MustFixed(550, 2),
				PHMax:         domain.MustFixed(650, 2),
			},
			Reviewers: []domain.PersonID{"alice", "bob", "carol", "dave"},
		}
	}

	newTaskAtVerification := func(t *testing.T) (*service.Service, domain.TaskID) {
		t.Helper()

		st, err := store.OpenMemory()
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		t.Cleanup(func() {
			if err := st.Close(); err != nil {
				t.Fatalf("close store: %v", err)
			}
		})
		cat, err := st.LoadCatalog()
		if err != nil {
			t.Fatalf("load catalog: %v", err)
		}
		svc := service.New(st, cat, domain.NewClock())

		locked, err := svc.Lock(lockRequest())
		if err != nil {
			t.Fatalf("lock: %v", err)
		}
		id := locked.TaskID

		for _, confirm := range []struct {
			operation domain.OperationID
			person    domain.PersonID
		}{
			{"sample-alice", "alice"},
			{"sample-bob", "bob"},
		} {
			if _, err := svc.SamplingConfirm(id, service.SamplingConfirmRequest{
				Operation:  confirm.operation,
				Generation: 1,
				Person:     confirm.person,
				Batch:      "B-001",
				Positions:  []domain.BagPosition{"P1", "P2"},
			}); err != nil {
				t.Fatalf("sampling confirm %s: %v", confirm.person, err)
			}
		}
		if _, err := svc.SampleSeal(id, service.SampleSealRequest{
			Operation:  "seal",
			Generation: 1,
			Person:     "alice",
			Positions:  []domain.BagPosition{"P1", "P2"},
		}); err != nil {
			t.Fatalf("seal: %v", err)
		}
		if _, err := svc.OccupancyStart(id, service.OccupancyStartRequest{
			Operation:  "occupy",
			Generation: 1,
			RackID:     "R1",
			ProbeID:    "probe-1",
			ProbeStart: 1,
			ProbeEnd:   100,
		}); err != nil {
			t.Fatalf("occupy: %v", err)
		}
		for _, obs := range []struct {
			operation domain.OperationID
			day       domain.DayAge
			position  domain.BagPosition
		}{
			{"observe-p1-d1", 1, "P1"},
			{"observe-p2-d1", 1, "P2"},
			{"observe-p1-d2", 2, "P1"},
			{"observe-p2-d2", 2, "P2"},
		} {
			if _, err := svc.Observation(id, service.ObservationRequest{
				Operation:          obs.operation,
				Generation:         1,
				DayAge:             obs.day,
				Position:           obs.position,
				MyceliumCoverage:   "85.0",
				ContaminationCount: 0,
				Observer:           "carol",
			}); err != nil {
				t.Fatalf("observe %s: %v", obs.operation, err)
			}
		}
		return svc, id
	}

	submitReadings := func(t *testing.T, svc *service.Service, id domain.TaskID, readings []reading) {
		t.Helper()
		for _, r := range readings {
			if _, err := svc.DeviceReading(id, service.DeviceReadingRequest{
				Operation:  r.operation,
				Generation: 1,
				DeviceType: r.deviceType,
				DeviceID:   r.deviceID,
				Metric:     r.metric,
				Value:      r.value,
				Position:   r.position,
			}); err != nil {
				t.Fatalf("device reading %s: %v", r.operation, err)
			}
		}
	}

	submitApprovals := func(t *testing.T, svc *service.Service, id domain.TaskID) {
		t.Helper()
		for _, review := range []struct {
			operation domain.OperationID
			person    domain.PersonID
		}{
			{"review-carol", "carol"},
			{"review-dave", "dave"},
		} {
			if _, err := svc.Review(id, service.ReviewRequest{
				Operation:  review.operation,
				Generation: 1,
				Person:     review.person,
				Conclusion: "approve",
			}); err != nil {
				t.Fatalf("review %s: %v", review.person, err)
			}
		}
	}

	cases := []struct {
		name             string
		readings         []reading
		reviews          bool
		positiveEvidence bool
		conclusion       string
		wantFinalType    arbiter.FinalType
		wantErrCode      domain.ErrorCode
	}{
		{
			name:        "transfer rejects when P2 has no accepted readings",
			readings:    onlyP1Readings,
			reviews:     true,
			conclusion:  "transfer",
			wantErrCode: domain.CodeReadingOutOfRange,
		},
		{
			name:          "transfer succeeds when every position has moisture and pH",
			readings:      allReadings,
			reviews:       true,
			conclusion:    "transfer",
			wantFinalType: arbiter.FinalTransferable,
		},
		{
			name:          "cancel remains independent of physchem completion",
			readings:      onlyP1Readings,
			conclusion:    "cancel",
			wantFinalType: arbiter.FinalCancelled,
		},
		{
			name:             "isolate remains independent of physchem completion",
			readings:         onlyP1Readings,
			positiveEvidence: true,
			conclusion:       "isolate",
			wantFinalType:    arbiter.FinalContaminationIsolated,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, id := newTaskAtVerification(t)
			submitReadings(t, svc, id, tc.readings)
			if tc.positiveEvidence {
				if _, err := svc.ContaminationRecheck(id, service.ContaminationRecheckRequest{
					Operation:         "positive-evidence",
					Generation:        1,
					RecheckGeneration: 1,
					Position:          "P1",
					DayAge:            1,
					Well:              "A1",
					Source:            "molecular",
					Positive:          true,
				}); err != nil {
					t.Fatalf("positive evidence: %v", err)
				}
			}
			if tc.reviews {
				submitApprovals(t, svc, id)
			}

			got, err := svc.Finalize(id, service.FinalizeRequest{
				Operation:  "finalize",
				Generation: 1,
				Conclusion: tc.conclusion,
			})
			if tc.wantErrCode != "" {
				if err == nil {
					t.Fatalf("finalize succeeded with credential %q, want error %s", got.Credential, tc.wantErrCode)
				}
				var domainErr *domain.Error
				if !errors.As(err, &domainErr) {
					t.Fatalf("finalize error = %T %v, want domain error %s", err, err, tc.wantErrCode)
				}
				if domainErr.Code != tc.wantErrCode {
					t.Fatalf("finalize error code = %s, want %s", domainErr.Code, tc.wantErrCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("finalize: %v", err)
			}
			if got.FinalType != string(tc.wantFinalType) {
				t.Fatalf("final type = %s, want %s", got.FinalType, tc.wantFinalType)
			}
			if got.Credential == "" {
				t.Fatal("finalize returned empty credential")
			}
		})
	}
}
