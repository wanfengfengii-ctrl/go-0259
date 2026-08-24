package service

import (
	"path/filepath"
	"sync"
	"testing"

	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
	"mycocycle-growbag-transfer-gate/internal/store"
)

// submitPhysChem submits accepted moisture and pH readings for both bag
// positions within the locked thresholds.
func submitPhysChem(t *testing.T, s *Service, id domain.TaskID) {
	t.Helper()
	for _, pos := range []domain.BagPosition{"P1", "P2"} {
		if _, err := s.DeviceReading(id, DeviceReadingRequest{
			Operation: domain.OperationID("m-" + string(pos)), Generation: 1, DeviceType: domain.DeviceMoisture,
			DeviceID: "moist-1", Metric: domain.MetricMoisture, Value: "65.0", Position: pos,
		}); err != nil {
			t.Fatalf("moisture %s: %v", pos, err)
		}
		if _, err := s.DeviceReading(id, DeviceReadingRequest{
			Operation: domain.OperationID("p-" + string(pos)), Generation: 1, DeviceType: domain.DevicePHMeter,
			DeviceID: "ph-1", Metric: domain.MetricPH, Value: "6.00", Position: pos,
		}); err != nil {
			t.Fatalf("ph %s: %v", pos, err)
		}
	}
}

func reviewApprove(t *testing.T, s *Service, id domain.TaskID, op string, person domain.PersonID) {
	t.Helper()
	if _, err := s.Review(id, ReviewRequest{
		Operation: domain.OperationID(op), Generation: 1, Person: person, Conclusion: "approve",
	}); err != nil {
		t.Fatalf("review %s: %v", person, err)
	}
}

func toReviewReady(t *testing.T, s *Service) domain.TaskID {
	t.Helper()
	id := toVerifying(t, s)
	submitPhysChem(t, s, id)
	reviewApprove(t, s, id, "r1", "carol")
	reviewApprove(t, s, id, "r2", "dave")
	return id
}

func TestFinalizeTransferGeneratesSingleCredential(t *testing.T) {
	s := newTestService(t)
	id := toReviewReady(t, s)

	res, err := s.Finalize(id, FinalizeRequest{Operation: "f1", Generation: 1, Conclusion: "transfer"})
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if res.FinalType != string(arbiter.FinalTransferable) {
		t.Fatalf("final type = %s, want transferable", res.FinalType)
	}
	if res.Credential == "" {
		t.Fatal("empty credential")
	}
	if res.State != inspection.StateTransferable {
		t.Fatalf("state = %s, want transferable", res.State)
	}
}

func TestConcurrentFinalizeOnlyOneWins(t *testing.T) {
	s := newTestService(t)
	id := toReviewReady(t, s)

	var wg sync.WaitGroup
	var transferErr, cancelErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, transferErr = s.Finalize(id, FinalizeRequest{Operation: "f1", Generation: 1, Conclusion: "transfer"})
	}()
	go func() {
		defer wg.Done()
		_, cancelErr = s.Finalize(id, FinalizeRequest{Operation: "f2", Generation: 1, Conclusion: "cancel"})
	}()
	wg.Wait()

	if transferErr == nil && cancelErr == nil {
		t.Fatal("both finalize calls succeeded")
	}
	if transferErr != nil && cancelErr != nil {
		t.Fatalf("both failed: transfer=%v cancel=%v", transferErr, cancelErr)
	}
	d, err := s.store.LoadDecision(ctx(), id)
	if err != nil {
		t.Fatalf("load decision: %v", err)
	}
	if d.FinalType != arbiter.FinalTransferable && d.FinalType != arbiter.FinalCancelled {
		t.Fatalf("unexpected final type %s", d.FinalType)
	}
}

func TestFinalStateRejectsFurtherOperations(t *testing.T) {
	s := newTestService(t)
	id := toReviewReady(t, s)
	if _, err := s.Finalize(id, FinalizeRequest{Operation: "f1", Generation: 1, Conclusion: "transfer"}); err != nil {
		t.Fatalf("finalize: %v", err)
	}

	_, err := s.Observation(id, ObservationRequest{
		Operation: "o", Generation: 1, DayAge: 1, Position: "P1",
		MyceliumCoverage: "85.0", Observer: "carol",
	})
	assertErrorCode(t, err, domain.CodeFinalStateRejected)

	_, err = s.ContaminationRecheck(id, ContaminationRecheckRequest{
		Operation: "c", Generation: 1, RecheckGeneration: 1,
		Position: "P1", DayAge: 1, Well: "A1", Positive: false,
	})
	assertErrorCode(t, err, domain.CodeFinalStateRejected)

	_, err = s.Review(id, ReviewRequest{Operation: "r", Generation: 1, Person: "carol", Conclusion: "approve"})
	assertErrorCode(t, err, domain.CodeFinalStateRejected)
}

func TestRestartPreservesFinalBarrier(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "myco.db")

	st, err := store.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	cat, err := st.LoadCatalog()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	s := New(st, cat, domain.NewClock())
	id := toReviewReady(t, s)
	if _, err := s.Finalize(id, FinalizeRequest{Operation: "f1", Generation: 1, Conclusion: "transfer"}); err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Restart from the same database file.
	st2, err := store.Open(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer st2.Close()
	cat2, err := st2.LoadCatalog()
	if err != nil {
		t.Fatalf("reload catalog: %v", err)
	}
	s2 := New(st2, cat2, domain.NewClock())

	d, err := s2.store.LoadDecision(ctx(), id)
	if err != nil {
		t.Fatalf("decision lost after restart: %v", err)
	}
	if d.FinalType != arbiter.FinalTransferable {
		t.Fatalf("final type after restart = %s", d.FinalType)
	}
	_, err = s2.Finalize(id, FinalizeRequest{Operation: "f2", Generation: 1, Conclusion: "cancel"})
	assertErrorCode(t, err, domain.CodeFinalStateRejected)
}
