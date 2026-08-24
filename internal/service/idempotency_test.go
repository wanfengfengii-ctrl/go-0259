package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
)

func TestSamplingConfirmIdempotentRetry(t *testing.T) {
	s := newTestService(t)
	id := lockTask(t, s)
	req := SamplingConfirmRequest{
		Operation: "op1", Generation: 1, Person: "alice", Batch: "B-001",
		Positions: []domain.BagPosition{"P1", "P2"},
	}
	first, err := s.SamplingConfirm(id, req)
	if err != nil {
		t.Fatalf("first confirm: %v", err)
	}
	second, err := s.SamplingConfirm(id, req)
	if err != nil {
		t.Fatalf("retry confirm: %v", err)
	}
	if first.Confirmations != second.Confirmations || first.State != second.State {
		t.Fatalf("retry result differs: first=%+v second=%+v", first, second)
	}
}

func TestSamplingConfirmContentConflict(t *testing.T) {
	s := newTestService(t)
	id := lockTask(t, s)
	if _, err := s.SamplingConfirm(id, SamplingConfirmRequest{
		Operation: "op1", Generation: 1, Person: "alice", Batch: "B-001",
		Positions: []domain.BagPosition{"P1", "P2"},
	}); err != nil {
		t.Fatalf("first confirm: %v", err)
	}
	_, err := s.SamplingConfirm(id, SamplingConfirmRequest{
		Operation: "op1", Generation: 1, Person: "bob", Batch: "B-001",
		Positions: []domain.BagPosition{"P1", "P2"},
	})
	assertErrorCode(t, err, domain.CodeIdempotencyConflict)
}

func TestOldGenerationRejectedForAllSteps(t *testing.T) {
	s := newTestService(t)
	id := lockTask(t, s)

	_, err := s.SamplingConfirm(id, SamplingConfirmRequest{
		Operation: "op1", Generation: 2, Person: "alice", Batch: "B-001",
		Positions: []domain.BagPosition{"P1", "P2"},
	})
	assertErrorCode(t, err, domain.CodeGenerationConflict)

	_, err = s.SampleSeal(id, SampleSealRequest{
		Operation: "op2", Generation: 2, Person: "alice",
		Positions: []domain.BagPosition{"P1", "P2"},
	})
	assertErrorCode(t, err, domain.CodeGenerationConflict)

	_, err = s.Observation(id, ObservationRequest{
		Operation: "op3", Generation: 2, DayAge: 1, Position: "P1",
		MyceliumCoverage: "85.0", Observer: "carol",
	})
	assertErrorCode(t, err, domain.CodeGenerationConflict)

	_, err = s.Review(id, ReviewRequest{
		Operation: "op4", Generation: 2, Person: "carol", Conclusion: "approve",
	})
	assertErrorCode(t, err, domain.CodeGenerationConflict)
}
