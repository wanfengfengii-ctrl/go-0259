package service

import (
	"errors"
	"testing"

	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/store"
)

// newTestService builds a service backed by an in-memory SQLite store with the
// deterministic seeded catalog.
func newTestService(t *testing.T) *Service {
	t.Helper()
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	cat, err := st.LoadCatalog()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	return New(st, cat, domain.NewClock())
}

// defaultLockRequest returns a valid lock request against the seeded catalog.
func defaultLockRequest() catalog.LockRequest {
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

// lockTask locks a default task and returns its id.
func lockTask(t *testing.T, s *Service) domain.TaskID {
	t.Helper()
	res, err := s.Lock(defaultLockRequest())
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	return res.TaskID
}

// assertErrorCode asserts that err is a domain error with the given code.
func assertErrorCode(t *testing.T, err error, code domain.ErrorCode) {
	t.Helper()
	var de *domain.Error
	if !errors.As(err, &de) {
		t.Fatalf("expected domain error %q, got %v", code, err)
	}
	if de.Code != code {
		t.Fatalf("error code = %q, want %q (msg %s)", de.Code, code, de.Message)
	}
}

// advanceToObserving drives a freshly locked task through two confirmations,
// sealing and occupancy so it reaches the observing state.
func advanceToObserving(t *testing.T, s *Service, id domain.TaskID) {
	t.Helper()
	mustSamplingConfirm(t, s, id, "op1", "alice")
	mustSamplingConfirm(t, s, id, "op2", "bob")
	if _, err := s.SampleSeal(id, SampleSealRequest{
		Operation: "op3", Generation: 1, Person: "alice",
		Positions: []domain.BagPosition{"P1", "P2"},
	}); err != nil {
		t.Fatalf("seal: %v", err)
	}
	if _, err := s.OccupancyStart(id, OccupancyStartRequest{
		Operation: "op4", Generation: 1, RackID: "R1",
		ProbeID: "probe-1", ProbeStart: 1, ProbeEnd: 100,
	}); err != nil {
		t.Fatalf("occupy: %v", err)
	}
}

func mustSamplingConfirm(t *testing.T, s *Service, id domain.TaskID, op, person string) {
	t.Helper()
	if _, err := s.SamplingConfirm(id, SamplingConfirmRequest{
		Operation: domain.OperationID(op), Generation: 1, Person: domain.PersonID(person),
		Batch: "B-001", Positions: []domain.BagPosition{"P1", "P2"},
	}); err != nil {
		t.Fatalf("sampling confirm %s: %v", person, err)
	}
}

// completeObservations fills the full day-age x position matrix with valid
// observations and returns after the task advances to contamination verification.
func completeObservations(t *testing.T, s *Service, id domain.TaskID) {
	t.Helper()
	cells := []struct {
		day domain.DayAge
		pos domain.BagPosition
	}{
		{1, "P1"}, {1, "P2"}, {2, "P1"}, {2, "P2"},
	}
	for i, c := range cells {
		if _, err := s.Observation(id, ObservationRequest{
			Operation: domain.OperationID("o" + itoa(i)), Generation: 1,
			DayAge: c.day, Position: c.pos,
			MyceliumCoverage: "85.0", ContaminationCount: 0,
			Observer: "carol",
		}); err != nil {
			t.Fatalf("observe %v: %v", c, err)
		}
	}
}

func itoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	var b [12]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = digits[i%10]
		i /= 10
	}
	return string(b[p:])
}

// lockConfirmSeal locks a task with the given request, then runs two sampling
// confirmations and sealing so the task reaches the occupying state.
func lockConfirmSeal(t *testing.T, s *Service, req catalog.LockRequest) domain.TaskID {
	t.Helper()
	res, err := s.Lock(req)
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	id := res.TaskID
	pos := req.BagPositions
	mustSamplingConfirmPositions(t, s, id, "op1", "alice", req.BagBatch, pos)
	mustSamplingConfirmPositions(t, s, id, "op2", "bob", req.BagBatch, pos)
	if _, err := s.SampleSeal(id, SampleSealRequest{
		Operation: "op3", Generation: 1, Person: "alice", Positions: pos,
	}); err != nil {
		t.Fatalf("seal: %v", err)
	}
	return id
}

func mustSamplingConfirmPositions(t *testing.T, s *Service, id domain.TaskID, op, person string, batch domain.BagBatch, pos []domain.BagPosition) {
	t.Helper()
	if _, err := s.SamplingConfirm(id, SamplingConfirmRequest{
		Operation: domain.OperationID(op), Generation: 1, Person: domain.PersonID(person),
		Batch: batch, Positions: pos,
	}); err != nil {
		t.Fatalf("sampling confirm %s: %v", person, err)
	}
}
