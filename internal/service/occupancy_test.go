package service

import (
	"sync"
	"testing"

	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/domain"
)

func lockReqFor(batch domain.BagBatch, positions []domain.BagPosition, rack domain.RackID, probe domain.ProbeID) catalog.LockRequest {
	req := defaultLockRequest()
	req.BagBatch = batch
	req.BagPositions = positions
	req.RackID = rack
	req.ProbeWindow = catalog.ProbeWindow{ProbeID: probe, Start: 1, End: 100}
	return req
}

func TestConcurrentBagBatchExclusive(t *testing.T) {
	s := newTestService(t)
	if _, err := s.Lock(lockReqFor("B-001", []domain.BagPosition{"P1", "P2"}, "R1", "probe-1")); err != nil {
		t.Fatalf("first lock: %v", err)
	}
	_, err := s.Lock(lockReqFor("B-001", []domain.BagPosition{"P3", "P4"}, "R2", "probe-2"))
	assertErrorCode(t, err, domain.CodeResourceWindowConflict)
}

func TestConcurrentBagPositionExclusive(t *testing.T) {
	s := newTestService(t)
	if _, err := s.Lock(lockReqFor("B-001", []domain.BagPosition{"P1", "P2"}, "R1", "probe-1")); err != nil {
		t.Fatalf("first lock: %v", err)
	}
	_, err := s.Lock(lockReqFor("B-002", []domain.BagPosition{"P1", "P3"}, "R2", "probe-2"))
	assertErrorCode(t, err, domain.CodeResourceWindowConflict)
}

func TestRackConflictRejected(t *testing.T) {
	s := newTestService(t)
	id1 := lockConfirmSeal(t, s, lockReqFor("B-001", []domain.BagPosition{"P1", "P2"}, "R1", "probe-1"))
	id2 := lockConfirmSeal(t, s, lockReqFor("B-002", []domain.BagPosition{"P3", "P4"}, "R1", "probe-2"))
	if _, err := s.OccupancyStart(id1, OccupancyStartRequest{
		Operation: "s1", Generation: 1, RackID: "R1", ProbeID: "probe-1", ProbeStart: 1, ProbeEnd: 100,
	}); err != nil {
		t.Fatalf("first start: %v", err)
	}
	_, err := s.OccupancyStart(id2, OccupancyStartRequest{
		Operation: "s2", Generation: 1, RackID: "R1", ProbeID: "probe-2", ProbeStart: 1, ProbeEnd: 100,
	})
	assertErrorCode(t, err, domain.CodeResourceWindowConflict)
}

func TestProbeWindowOverlapRejected(t *testing.T) {
	s := newTestService(t)
	id1 := lockConfirmSeal(t, s, lockReqFor("B-001", []domain.BagPosition{"P1", "P2"}, "R1", "probe-1"))
	id2 := lockConfirmSeal(t, s, lockReqFor("B-002", []domain.BagPosition{"P3", "P4"}, "R2", "probe-1"))
	if _, err := s.OccupancyStart(id1, OccupancyStartRequest{
		Operation: "s1", Generation: 1, RackID: "R1", ProbeID: "probe-1", ProbeStart: 10, ProbeEnd: 60,
	}); err != nil {
		t.Fatalf("first start: %v", err)
	}
	_, err := s.OccupancyStart(id2, OccupancyStartRequest{
		Operation: "s2", Generation: 1, RackID: "R2", ProbeID: "probe-1", ProbeStart: 50, ProbeEnd: 90,
	})
	assertErrorCode(t, err, domain.CodeResourceWindowConflict)
}

func TestProbeWindowAdjacentAllowed(t *testing.T) {
	s := newTestService(t)
	id1 := lockConfirmSeal(t, s, lockReqFor("B-001", []domain.BagPosition{"P1", "P2"}, "R1", "probe-1"))
	id2 := lockConfirmSeal(t, s, lockReqFor("B-002", []domain.BagPosition{"P3", "P4"}, "R2", "probe-1"))
	if _, err := s.OccupancyStart(id1, OccupancyStartRequest{
		Operation: "s1", Generation: 1, RackID: "R1", ProbeID: "probe-1", ProbeStart: 1, ProbeEnd: 50,
	}); err != nil {
		t.Fatalf("first start: %v", err)
	}
	if _, err := s.OccupancyStart(id2, OccupancyStartRequest{
		Operation: "s2", Generation: 1, RackID: "R2", ProbeID: "probe-1", ProbeStart: 50, ProbeEnd: 100,
	}); err != nil {
		t.Fatalf("adjacent start should succeed: %v", err)
	}
}

func TestMoveRackAndStartCompeteAtomically(t *testing.T) {
	s := newTestService(t)
	id1 := lockConfirmSeal(t, s, lockReqFor("B-001", []domain.BagPosition{"P1", "P2"}, "R1", "probe-1"))
	id2 := lockConfirmSeal(t, s, lockReqFor("B-002", []domain.BagPosition{"P3", "P4"}, "R2", "probe-2"))
	// id1 starts observing on R1.
	if _, err := s.OccupancyStart(id1, OccupancyStartRequest{
		Operation: "s1", Generation: 1, RackID: "R1", ProbeID: "probe-1", ProbeStart: 1, ProbeEnd: 100,
	}); err != nil {
		t.Fatalf("first start: %v", err)
	}

	var wg sync.WaitGroup
	var moveErr, startErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, moveErr = s.OccupancyMoveRack(id1, OccupancyMoveRequest{
			Operation: "m1", Generation: 1, RackID: "R2", ProbeID: "probe-2", ProbeStart: 1, ProbeEnd: 100,
		})
	}()
	go func() {
		defer wg.Done()
		_, startErr = s.OccupancyStart(id2, OccupancyStartRequest{
			Operation: "s2", Generation: 1, RackID: "R2", ProbeID: "probe-2", ProbeStart: 1, ProbeEnd: 100,
		})
	}()
	wg.Wait()

	if moveErr == nil && startErr == nil {
		t.Fatal("both move and start acquired R2")
	}
	if moveErr != nil && startErr != nil {
		t.Fatalf("both failed: move=%v start=%v", moveErr, startErr)
	}
}
