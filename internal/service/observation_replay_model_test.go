package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

func TestModel_ObservationIdempotentReplayAfterMatrixClosure(t *testing.T) {
	observe := func(t *testing.T, s *Service, id domain.TaskID, req ObservationRequest) ObservationResult {
		t.Helper()
		res, err := s.Observation(id, req)
		if err != nil {
			t.Fatalf("observe %+v: %v", req, err)
		}
		return res
	}
	observingTask := func(t *testing.T) (*Service, domain.TaskID) {
		t.Helper()
		s := newTestService(t)
		id := lockTask(t, s)
		advanceToObserving(t, s, id)
		return s, id
	}
	validObservation := func(op string, day domain.DayAge, pos domain.BagPosition) ObservationRequest {
		return ObservationRequest{
			Operation:          domain.OperationID(op),
			Generation:         1,
			DayAge:             day,
			Position:           pos,
			MyceliumCoverage:   "85.0",
			ContaminationCount: 0,
			Observer:           "carol",
		}
	}
	assertTaskState := func(t *testing.T, s *Service, id domain.TaskID, want inspection.TaskState) {
		t.Helper()
		view, err := s.GetTask(id)
		if err != nil {
			t.Fatalf("get task: %v", err)
		}
		if view.State != want {
			t.Fatalf("task state = %s, want %s", view.State, want)
		}
	}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "same operation replay of final observation is accepted after state advances",
			run: func(t *testing.T) {
				s, id := observingTask(t)
				for _, req := range []ObservationRequest{
					validObservation("obs-fill-1", 1, "P1"),
					validObservation("obs-fill-2", 1, "P2"),
					validObservation("obs-fill-3", 2, "P1"),
				} {
					res := observe(t, s, id, req)
					if res.State != inspection.StateObserving {
						t.Fatalf("partial observation state = %s, want %s", res.State, inspection.StateObserving)
					}
					assertTaskState(t, s, id, inspection.StateObserving)
				}

				finalReq := validObservation("obs-final-retry", 2, "P2")
				first := observe(t, s, id, finalReq)
				if first.State != inspection.StateVerifyingContamination || first.Missing != 0 {
					t.Fatalf("final observation result = %+v, want state %s with no missing cells",
						first, inspection.StateVerifyingContamination)
				}
				assertTaskState(t, s, id, inspection.StateVerifyingContamination)

				replayed, err := s.Observation(id, finalReq)
				if err != nil {
					t.Fatalf("replay final observation: %v", err)
				}
				if replayed != first {
					t.Fatalf("replay result = %+v, want stored result %+v", replayed, first)
				}
				assertTaskState(t, s, id, inspection.StateVerifyingContamination)
			},
		},
		{
			name: "different operation cannot rewrite an already valid observation cell",
			run: func(t *testing.T) {
				s, id := observingTask(t)
				observe(t, s, id, validObservation("obs-original-cell", 1, "P1"))

				rewrite := validObservation("obs-rewrite-cell", 1, "P1")
				rewrite.MyceliumCoverage = "90.0"
				_, err := s.Observation(id, rewrite)
				assertErrorCode(t, err, domain.CodeIdempotencyConflict)
				assertTaskState(t, s, id, inspection.StateObserving)
			},
		},
		{
			name: "new operations still reject old generation illegal day age and illegal bag position",
			run: func(t *testing.T) {
				s, id := observingTask(t)

				_, err := s.Observation(id, validObservation("obs-control-cell", 1, "P1"))
				if err != nil {
					t.Fatalf("control observation should be accepted before validation checks: %v", err)
				}

				oldGeneration := validObservation("obs-old-generation", 1, "P2")
				oldGeneration.Generation = 0
				_, err = s.Observation(id, oldGeneration)
				assertErrorCode(t, err, domain.CodeGenerationConflict)

				illegalDay := validObservation("obs-illegal-day", 99, "P2")
				_, err = s.Observation(id, illegalDay)
				assertErrorCode(t, err, domain.CodeReadingOutOfRange)

				illegalPosition := validObservation("obs-illegal-position", 2, "PX")
				_, err = s.Observation(id, illegalPosition)
				assertErrorCode(t, err, domain.CodeReadingOutOfRange)
				assertTaskState(t, s, id, inspection.StateObserving)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
