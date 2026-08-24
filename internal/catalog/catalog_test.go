package catalog

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
)

func TestNewSnapshotCopiesSlices(t *testing.T) {
	req := LockRequest{
		BagBatch:     "B-001",
		BagPositions: []domain.BagPosition{"P1", "P2"},
		Schedule:     ScheduleTemplate{DayAges: []domain.DayAge{1, 2, 3}},
		Reviewers:    []domain.PersonID{"alice", "bob"},
	}
	snap := NewSnapshot(req)

	req.BagPositions[0] = "MUTATED"
	req.Schedule.DayAges[0] = 99
	req.Reviewers[0] = "MUTATED"

	if snap.BagPositions[0] != "P1" {
		t.Fatalf("bag positions not copied: %v", snap.BagPositions)
	}
	if snap.Schedule.DayAges[0] != 1 {
		t.Fatalf("schedule not copied: %v", snap.Schedule.DayAges)
	}
	if snap.Reviewers[0] != "alice" {
		t.Fatalf("reviewers not copied: %v", snap.Reviewers)
	}
}
