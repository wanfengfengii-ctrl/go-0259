package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

func TestModel_ReviewQuorumIsTaskScoped(t *testing.T) {
	completeTransferInputsExceptReviews := func(t *testing.T, s *Service, req catalog.LockRequest) domain.TaskID {
		t.Helper()

		id := lockConfirmSeal(t, s, req)
		if _, err := s.OccupancyStart(id, OccupancyStartRequest{
			Operation:  domain.OperationID("occupy-" + string(req.BagBatch)),
			Generation: 1,
			RackID:     req.RackID,
			ProbeID:    req.ProbeWindow.ProbeID,
			ProbeStart: req.ProbeWindow.Start,
			ProbeEnd:   req.ProbeWindow.End,
		}); err != nil {
			t.Fatalf("occupancy start: %v", err)
		}

		op := 0
		for _, day := range req.Schedule.DayAges {
			for _, pos := range req.BagPositions {
				op++
				if _, err := s.Observation(id, ObservationRequest{
					Operation:        domain.OperationID("obs-" + string(req.BagBatch) + "-" + itoa(op)),
					Generation:       1,
					DayAge:           day,
					Position:         pos,
					MyceliumCoverage: "85.0",
					Observer:         "carol",
				}); err != nil {
					t.Fatalf("observe %s@%d: %v", pos, day, err)
				}
			}
		}

		for _, pos := range req.BagPositions {
			if _, err := s.DeviceReading(id, DeviceReadingRequest{
				Operation:  domain.OperationID("moist-" + string(req.BagBatch) + "-" + string(pos)),
				Generation: 1,
				DeviceType: domain.DeviceMoisture,
				DeviceID:   "moist-1",
				Metric:     domain.MetricMoisture,
				Value:      "65.0",
				Position:   pos,
			}); err != nil {
				t.Fatalf("moisture %s: %v", pos, err)
			}
			if _, err := s.DeviceReading(id, DeviceReadingRequest{
				Operation:  domain.OperationID("ph-" + string(req.BagBatch) + "-" + string(pos)),
				Generation: 1,
				DeviceType: domain.DevicePHMeter,
				DeviceID:   "ph-1",
				Metric:     domain.MetricPH,
				Value:      "6.00",
				Position:   pos,
			}); err != nil {
				t.Fatalf("ph %s: %v", pos, err)
			}
		}

		return id
	}

	seedOtherTaskApprovals := func(t *testing.T, s *Service) domain.TaskID {
		t.Helper()

		id := completeTransferInputsExceptReviews(t, s, lockReqFor("B-001", []domain.BagPosition{"P1", "P2"}, "R1", "probe-1"))
		reviewApprove(t, s, id, "task-a-carol", "carol")
		reviewApprove(t, s, id, "task-a-dave", "dave")
		return id
	}

	taskBReq := func() catalog.LockRequest {
		return lockReqFor("B-002", []domain.BagPosition{"P3", "P4"}, "R2", "probe-2")
	}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "detail excludes reviews from another task",
			run: func(t *testing.T) {
				s := newTestService(t)
				seedOtherTaskApprovals(t, s)
				id := completeTransferInputsExceptReviews(t, s, taskBReq())

				detail, err := s.GetTaskDetail(id)
				if err != nil {
					t.Fatalf("detail: %v", err)
				}
				if len(detail.Reviews) != 0 {
					t.Fatalf("detail reviews = %d, want 0 for current task %s: %+v", len(detail.Reviews), id, detail.Reviews)
				}
			},
		},
		{
			name: "another task quorum does not authorize transfer",
			run: func(t *testing.T) {
				s := newTestService(t)
				seedOtherTaskApprovals(t, s)
				id := completeTransferInputsExceptReviews(t, s, taskBReq())

				res, err := s.Finalize(id, FinalizeRequest{
					Operation:  "task-b-finalize",
					Generation: 1,
					Conclusion: "transfer",
				})
				if err == nil {
					t.Fatalf("finalize succeeded with type %q credential %q, want missing current-task review quorum error", res.FinalType, res.Credential)
				}
				assertErrorCode(t, err, domain.CodeRoleOverlap)
			},
		},
		{
			name: "current task quorum still authorizes transfer",
			run: func(t *testing.T) {
				s := newTestService(t)
				id := completeTransferInputsExceptReviews(t, s, lockReqFor("B-003", []domain.BagPosition{"P5", "P6"}, "R3", "probe-3"))
				reviewApprove(t, s, id, "task-c-carol", "carol")
				reviewApprove(t, s, id, "task-c-dave", "dave")

				detail, err := s.GetTaskDetail(id)
				if err != nil {
					t.Fatalf("detail: %v", err)
				}
				if len(detail.Reviews) != 2 {
					t.Fatalf("detail reviews = %d, want 2", len(detail.Reviews))
				}
				for _, review := range detail.Reviews {
					if review.TaskID != id {
						t.Fatalf("detail review task_id = %s, want %s", review.TaskID, id)
					}
				}

				res, err := s.Finalize(id, FinalizeRequest{
					Operation:  "task-c-finalize",
					Generation: 1,
					Conclusion: "transfer",
				})
				if err != nil {
					t.Fatalf("finalize: %v", err)
				}
				if res.FinalType != string(arbiter.FinalTransferable) {
					t.Fatalf("final type = %s, want %s", res.FinalType, arbiter.FinalTransferable)
				}
				if res.Credential == "" {
					t.Fatal("empty credential")
				}
				if res.State != inspection.StateTransferable {
					t.Fatalf("state = %s, want %s", res.State, inspection.StateTransferable)
				}
			},
		},
		{
			name: "duplicate reviewer still conflicts within current task",
			run: func(t *testing.T) {
				s := newTestService(t)
				id := completeTransferInputsExceptReviews(t, s, lockReqFor("B-004", []domain.BagPosition{"P7", "P8"}, "R4", "probe-4"))
				reviewApprove(t, s, id, "task-d-carol-1", "carol")

				_, err := s.Review(id, ReviewRequest{
					Operation:  "task-d-carol-2",
					Generation: 1,
					Person:     "carol",
					Conclusion: "approve",
				})
				assertErrorCode(t, err, domain.CodeRoleOverlap)
			},
		},
		{
			name: "qualified reviewer outside task snapshot remains ineligible",
			run: func(t *testing.T) {
				s := newTestService(t)
				req := lockReqFor("B-005", []domain.BagPosition{"P9", "P10"}, "R5", "probe-5")
				req.Reviewers = []domain.PersonID{"carol", "dave"}
				id := completeTransferInputsExceptReviews(t, s, req)

				_, err := s.Review(id, ReviewRequest{
					Operation:  "task-e-frank",
					Generation: 1,
					Person:     "frank",
					Conclusion: "approve",
				})
				assertErrorCode(t, err, domain.CodeRoleOverlap)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
