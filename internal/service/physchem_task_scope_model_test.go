package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
	"mycocycle-growbag-transfer-gate/internal/maturity"
	"mycocycle-growbag-transfer-gate/internal/occupancy"
)

func TestModel_PhysChemReadingsStayWithinTaskBoundary(t *testing.T) {
	type scenario struct {
		service *Service
		taskA   domain.TaskID
		taskB   domain.TaskID
	}

	build := func(t *testing.T, taskBReadings string) scenario {
		t.Helper()
		s := newTestService(t)

		taskA := toVerifying(t, s)
		submitPhysChem(t, s, taskA)
		for _, pos := range []domain.BagPosition{"P1", "P2"} {
			if err := s.Store().ReleaseLease(ctx(), taskA, occupancy.ResourceBagPosition, string(pos)); err != nil {
				t.Fatalf("release task A position %s: %v", pos, err)
			}
		}

		req := defaultLockRequest()
		req.BagBatch = "B-002"
		req.RackID = "R2"
		req.ProbeWindow.ProbeID = "probe-2"
		req.ProbeWindow.Start = 101
		req.ProbeWindow.End = 200

		res, err := s.Lock(req)
		if err != nil {
			t.Fatalf("lock task B: %v", err)
		}
		taskB := res.TaskID
		mustSamplingConfirmPositions(t, s, taskB, "b-op1", "alice", req.BagBatch, req.BagPositions)
		mustSamplingConfirmPositions(t, s, taskB, "b-op2", "bob", req.BagBatch, req.BagPositions)
		if _, err := s.SampleSeal(taskB, SampleSealRequest{
			Operation: "b-op3", Generation: 1, Person: "alice", Positions: req.BagPositions,
		}); err != nil {
			t.Fatalf("seal task B: %v", err)
		}
		if _, err := s.OccupancyStart(taskB, OccupancyStartRequest{
			Operation: "b-op4", Generation: 1, RackID: req.RackID,
			ProbeID: req.ProbeWindow.ProbeID, ProbeStart: req.ProbeWindow.Start, ProbeEnd: req.ProbeWindow.End,
		}); err != nil {
			t.Fatalf("occupy task B: %v", err)
		}
		completeObservations(t, s, taskB)

		switch taskBReadings {
		case "moisture-only":
			for _, pos := range []domain.BagPosition{"P1", "P2"} {
				if _, err := s.DeviceReading(taskB, DeviceReadingRequest{
					Operation: domain.OperationID("b-m-" + string(pos)), Generation: 1,
					DeviceType: domain.DeviceMoisture, DeviceID: "moist-2",
					Metric: domain.MetricMoisture, Value: "65.0", Position: pos,
				}); err != nil {
					t.Fatalf("task B moisture %s: %v", pos, err)
				}
			}
		case "complete":
			submitPhysChem(t, s, taskB)
		case "none":
		default:
			t.Fatalf("unknown task B reading setup %q", taskBReadings)
		}

		reviewApprove(t, s, taskB, "b-r1", "carol")
		reviewApprove(t, s, taskB, "b-r2", "dave")
		return scenario{service: s, taskA: taskA, taskB: taskB}
	}

	cases := []struct {
		name          string
		taskBReadings string
		assert        func(*testing.T, scenario)
	}{
		{
			name:          "detail excludes readings from another task",
			taskBReadings: "none",
			assert: func(t *testing.T, sc scenario) {
				t.Helper()
				taskADetail, err := sc.service.GetTaskDetail(sc.taskA)
				if err != nil {
					t.Fatalf("detail task A: %v", err)
				}
				if len(taskADetail.Readings) != 4 {
					t.Fatalf("task A readings = %d, want 4", len(taskADetail.Readings))
				}
				for _, r := range taskADetail.Readings {
					if r.TaskID != sc.taskA {
						t.Fatalf("task A detail included reading for %s", r.TaskID)
					}
				}

				taskBDetail, err := sc.service.GetTaskDetail(sc.taskB)
				if err != nil {
					t.Fatalf("detail task B: %v", err)
				}
				if len(taskBDetail.Readings) != 0 {
					t.Fatalf("task B readings = %d, want 0", len(taskBDetail.Readings))
				}
			},
		},
		{
			name:          "progress and finalize require current task pH readings",
			taskBReadings: "moisture-only",
			assert: func(t *testing.T, sc scenario) {
				t.Helper()
				progress, err := sc.service.Progress(sc.taskB)
				if err != nil {
					t.Fatalf("progress task B: %v", err)
				}
				if progress.PhysChemCollected {
					t.Fatal("task B progress reported physchem collected without current task pH readings")
				}
				if progress.ReadyForTransfer {
					t.Fatal("task B progress reported ready for transfer without current task pH readings")
				}
				_, err = sc.service.Finalize(sc.taskB, FinalizeRequest{
					Operation: "b-finalize", Generation: 1, Conclusion: "transfer",
				})
				assertErrorCode(t, err, domain.CodeReadingOutOfRange)

				detail, err := sc.service.GetTaskDetail(sc.taskB)
				if err != nil {
					t.Fatalf("detail task B after rejected finalize: %v", err)
				}
				if detail.Decision != nil {
					t.Fatal("task B received a transfer decision without current task pH readings")
				}
				if detail.State == string(inspection.StateTransferable) {
					t.Fatal("task B became transferable without current task pH readings")
				}
			},
		},
		{
			name:          "complete current task readings still transfer",
			taskBReadings: "complete",
			assert: func(t *testing.T, sc scenario) {
				t.Helper()
				detail, err := sc.service.GetTaskDetail(sc.taskB)
				if err != nil {
					t.Fatalf("detail task B: %v", err)
				}
				if len(detail.Readings) != 4 {
					t.Fatalf("task B readings = %d, want 4", len(detail.Readings))
				}
				for _, r := range detail.Readings {
					if r.TaskID != sc.taskB {
						t.Fatalf("task B detail included reading for %s", r.TaskID)
					}
					if !r.Accepted() {
						t.Fatalf("task B reading %s/%s was not accepted", r.Position, r.Metric)
					}
					if r.Derived != maturity.DerivedPass {
						t.Fatalf("task B reading %s/%s derived = %s, want pass", r.Position, r.Metric, r.Derived)
					}
				}

				progress, err := sc.service.Progress(sc.taskB)
				if err != nil {
					t.Fatalf("progress task B: %v", err)
				}
				if !progress.PhysChemCollected {
					t.Fatal("task B progress did not report collected physchem readings")
				}
				if !progress.ReadyForTransfer {
					t.Fatalf("task B not ready for transfer: %v", progress.UnmetPreconditions)
				}

				res, err := sc.service.Finalize(sc.taskB, FinalizeRequest{
					Operation: "b-finalize", Generation: 1, Conclusion: "transfer",
				})
				if err != nil {
					t.Fatalf("finalize task B: %v", err)
				}
				if res.FinalType != string(arbiter.FinalTransferable) {
					t.Fatalf("final type = %s, want transferable", res.FinalType)
				}
				if res.Credential == "" {
					t.Fatal("empty transfer credential")
				}
				if res.State != inspection.StateTransferable {
					t.Fatalf("state = %s, want transferable", res.State)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, build(t, tc.taskBReadings))
		})
	}
}
