package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
)

func toVerifying(t *testing.T, s *Service) domain.TaskID {
	t.Helper()
	id := lockTask(t, s)
	advanceToObserving(t, s, id)
	completeObservations(t, s, id)
	return id
}

func TestPositiveContaminationCreatesSingleEvidence(t *testing.T) {
	s := newTestService(t)
	id := toVerifying(t, s)

	res, err := s.ContaminationRecheck(id, ContaminationRecheckRequest{
		Operation: "c1", Generation: 1, RecheckGeneration: 1,
		Position: "P1", DayAge: 1, Well: "A1", Source: "molecular", Positive: true,
	})
	if err != nil {
		t.Fatalf("recheck: %v", err)
	}
	if res.Version != 1 {
		t.Fatalf("version = %d, want 1", res.Version)
	}
	evidence, err := s.store.LoadEvidence(ctx(), id)
	if err != nil {
		t.Fatalf("load evidence: %v", err)
	}
	if len(evidence) != 1 {
		t.Fatalf("evidence = %d, want 1", len(evidence))
	}
	if !evidence[0].Positive {
		t.Fatal("evidence should be positive")
	}
}

func TestLateRecheckGenerationAuditedOnly(t *testing.T) {
	s := newTestService(t)
	id := toVerifying(t, s)

	if _, err := s.ContaminationRecheck(id, ContaminationRecheckRequest{
		Operation: "c1", Generation: 1, RecheckGeneration: 0,
		Position: "P1", DayAge: 1, Well: "A1", Source: "molecular", Positive: true,
	}); err != nil {
		t.Fatalf("late recheck should not error: %v", err)
	}
	evidence, _ := s.store.LoadEvidence(ctx(), id)
	if len(evidence) != 0 {
		t.Fatalf("late recheck wrote %d evidence records, want 0", len(evidence))
	}
	audit, _ := s.store.LoadAudit(ctx(), id)
	found := false
	for _, a := range audit {
		if a.Action == domain.AuditLateReading {
			found = true
		}
	}
	if !found {
		t.Fatal("late recheck should be audited")
	}
}

func TestRecheckGenerationConflictRejected(t *testing.T) {
	s := newTestService(t)
	id := toVerifying(t, s)

	_, err := s.ContaminationRecheck(id, ContaminationRecheckRequest{
		Operation: "c1", Generation: 1, RecheckGeneration: 2,
		Position: "P1", DayAge: 1, Well: "A1", Source: "molecular", Positive: true,
	})
	assertErrorCode(t, err, domain.CodeGenerationConflict)
}

func TestEvidenceVersionChain(t *testing.T) {
	s := newTestService(t)
	id := toVerifying(t, s)

	r1, err := s.ContaminationRecheck(id, ContaminationRecheckRequest{
		Operation: "c1", Generation: 1, RecheckGeneration: 1,
		Position: "P1", DayAge: 1, Well: "A1", Source: "molecular", Positive: false,
	})
	if err != nil {
		t.Fatalf("first recheck: %v", err)
	}
	r2, err := s.ContaminationRecheck(id, ContaminationRecheckRequest{
		Operation: "c2", Generation: 1, RecheckGeneration: 1,
		Position: "P1", DayAge: 1, Well: "A2", Source: "molecular", Positive: false,
	})
	if err != nil {
		t.Fatalf("second recheck: %v", err)
	}
	if r1.Version != 1 || r2.Version != 2 {
		t.Fatalf("versions = %d, %d; want 1, 2", r1.Version, r2.Version)
	}
}
