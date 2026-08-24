package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

func TestMatrixCompleteAdvancesToContaminationVerification(t *testing.T) {
	s := newTestService(t)
	id := lockTask(t, s)
	advanceToObserving(t, s, id)
	completeObservations(t, s, id)

	view, err := s.GetTask(id)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if view.State != inspection.StateVerifyingContamination {
		t.Fatalf("state = %s, want verifying_contamination", view.State)
	}
}

func TestMissingDayAgeDoesNotAdvance(t *testing.T) {
	s := newTestService(t)
	id := lockTask(t, s)
	advanceToObserving(t, s, id)
	// Observe only three of four cells, leaving P2@2 missing.
	for i, c := range []struct {
		day domain.DayAge
		pos domain.BagPosition
	}{{1, "P1"}, {1, "P2"}, {2, "P1"}} {
		if _, err := s.Observation(id, ObservationRequest{
			Operation: domain.OperationID("o" + itoa(i)), Generation: 1, DayAge: c.day, Position: c.pos,
			MyceliumCoverage: "85.0", Observer: "carol",
		}); err != nil {
			t.Fatalf("observe %v: %v", c, err)
		}
	}
	view, _ := s.GetTask(id)
	if view.State != inspection.StateObserving {
		t.Fatalf("state = %s, want observing (matrix incomplete)", view.State)
	}
}

func TestRepeatObservationDoesNotOverwrite(t *testing.T) {
	s := newTestService(t)
	id := lockTask(t, s)
	advanceToObserving(t, s, id)
	req := ObservationRequest{
		Operation: "o1", Generation: 1, DayAge: 1, Position: "P1",
		MyceliumCoverage: "85.0", Observer: "carol",
	}
	if _, err := s.Observation(id, req); err != nil {
		t.Fatalf("first observe: %v", err)
	}
	req.Operation = "o2"
	req.MyceliumCoverage = "99.0"
	_, err := s.Observation(id, req)
	assertErrorCode(t, err, domain.CodeIdempotencyConflict)
}

func TestCoverageBoundary(t *testing.T) {
	s := newTestService(t)
	id := lockTask(t, s)
	advanceToObserving(t, s, id)
	// Max coverage 100.0 is accepted.
	if _, err := s.Observation(id, ObservationRequest{
		Operation: "o1", Generation: 1, DayAge: 1, Position: "P1",
		MyceliumCoverage: "100.0", Observer: "carol",
	}); err != nil {
		t.Fatalf("100.0 should be accepted: %v", err)
	}
	// Over 100.0 is rejected.
	_, err := s.Observation(id, ObservationRequest{
		Operation: "o2", Generation: 1, DayAge: 1, Position: "P2",
		MyceliumCoverage: "100.1", Observer: "carol",
	})
	assertErrorCode(t, err, domain.CodeReadingOutOfRange)
}

func TestContaminationCountBoundary(t *testing.T) {
	s := newTestService(t)
	id := lockTask(t, s)
	advanceToObserving(t, s, id)
	_, err := s.Observation(id, ObservationRequest{
		Operation: "o1", Generation: 1, DayAge: 1, Position: "P1",
		MyceliumCoverage: "85.0", ContaminationCount: -1, Observer: "carol",
	})
	assertErrorCode(t, err, domain.CodeReadingOutOfRange)
}
