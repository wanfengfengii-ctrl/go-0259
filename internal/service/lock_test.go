package service

import (
	"errors"
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

func TestLockFreezesFullSnapshot(t *testing.T) {
	s := newTestService(t)
	res, err := s.Lock(defaultLockRequest())
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	if res.TaskID == "" {
		t.Fatal("empty task id")
	}
	if res.Generation != 1 {
		t.Fatalf("generation = %d, want 1", res.Generation)
	}
	if res.State != inspection.StatePendingSampling {
		t.Fatalf("state = %s, want pending_sampling", res.State)
	}
	snap := res.Summary
	if snap.StrainRevision != "strain-po-2024.03" ||
		snap.SubstrateRevision != "sub-hw-01" ||
		snap.SterilizerSummary != "run-A-2026-08-21" ||
		snap.InoculationLine != "line-1" ||
		snap.BagBatch != "B-001" {
		t.Fatalf("snapshot did not freeze catalog fields: %+v", snap)
	}
	if len(snap.BagPositions) != 2 || len(snap.Schedule.DayAges) != 2 || len(snap.Reviewers) != 4 {
		t.Fatalf("snapshot did not freeze sets: %+v", snap)
	}
}

func TestLockSubstrateSummaryMismatchRejected(t *testing.T) {
	s := newTestService(t)
	req := defaultLockRequest()
	req.SubstrateSummary = "wrong summary"
	_, err := s.Lock(req)
	assertErrorCode(t, err, domain.CodeSubstrateMismatch)
}

func TestLockStaleSterilizerRejected(t *testing.T) {
	s := newTestService(t)
	req := defaultLockRequest()
	req.SterilizerSummary = "run-stale-2026-01-01"
	_, err := s.Lock(req)
	assertErrorCode(t, err, domain.CodeStaleSterilizer)
}

func TestLockDuplicateBagPositionSortedError(t *testing.T) {
	s := newTestService(t)
	req := defaultLockRequest()
	req.BagPositions = []domain.BagPosition{"P2", "P1", "P2"}
	_, err := s.Lock(req)
	var de *domain.Error
	if !errors.As(err, &de) {
		t.Fatalf("expected domain error, got %v", err)
	}
	if de.Code != domain.CodeDuplicateBagPosition {
		t.Fatalf("code = %q, want duplicate_bag_position", de.Code)
	}
	if len(de.Reasons) != 1 || de.Reasons[0] != "P2" {
		t.Fatalf("reasons = %v, want [P2]", de.Reasons)
	}
}
