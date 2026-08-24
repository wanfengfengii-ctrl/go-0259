package service

import (
	"sort"
	"strings"
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
	"mycocycle-growbag-transfer-gate/internal/occupancy"
)

func TestModel_MoveRackMigrationLeaseBoundary(t *testing.T) {
	activeRackProbe := func(t *testing.T, s *Service, id domain.TaskID) string {
		t.Helper()
		detail, err := s.GetTaskDetail(id)
		if err != nil {
			t.Fatalf("detail: %v", err)
		}
		var active []string
		for _, lease := range detail.Leases {
			if lease.ResourceType != occupancy.ResourceRack && lease.ResourceType != occupancy.ResourceProbeWindow {
				continue
			}
			if lease.State == occupancy.LeaseActive {
				active = append(active, string(lease.ResourceType)+":"+lease.ResourceID)
			}
		}
		sort.Strings(active)
		return strings.Join(active, ",")
	}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "successful move releases source rack and probe window",
			run: func(t *testing.T) {
				s := newTestService(t)
				movedID := lockConfirmSeal(t, s, lockReqFor("B-001", []domain.BagPosition{"P1", "P2"}, "R1", "probe-1"))
				reuserID := lockConfirmSeal(t, s, lockReqFor("B-002", []domain.BagPosition{"P3", "P4"}, "R3", "probe-3"))

				if _, err := s.OccupancyStart(movedID, OccupancyStartRequest{
					Operation: "start-moved", Generation: 1,
					RackID: "R1", ProbeID: "probe-1", ProbeStart: 10, ProbeEnd: 80,
				}); err != nil {
					t.Fatalf("start moved task: %v", err)
				}
				res, err := s.OccupancyMoveRack(movedID, OccupancyMoveRequest{
					Operation: "move-rack", Generation: 1,
					RackID: "R2", ProbeID: "probe-2", ProbeStart: 10, ProbeEnd: 80,
				})
				if err != nil {
					t.Fatalf("move rack: %v", err)
				}
				if res.State != inspection.StateObserving {
					t.Fatalf("move state = %s, want %s", res.State, inspection.StateObserving)
				}
				if got, want := activeRackProbe(t, s, movedID), "probe_window:probe-2,rack:R2"; got != want {
					t.Fatalf("active rack/probe leases after move = %s, want %s", got, want)
				}
				if _, err := s.OccupancyStart(reuserID, OccupancyStartRequest{
					Operation: "reuse-source", Generation: 1,
					RackID: "R1", ProbeID: "probe-1", ProbeStart: 10, ProbeEnd: 80,
				}); err != nil {
					t.Fatalf("source rack/probe should be reusable after move: %v", err)
				}
			},
		},
		{
			name: "target conflict rejects move and keeps source occupancy",
			run: func(t *testing.T) {
				s := newTestService(t)
				movedID := lockConfirmSeal(t, s, lockReqFor("B-001", []domain.BagPosition{"P1", "P2"}, "R1", "probe-1"))
				occupantID := lockConfirmSeal(t, s, lockReqFor("B-002", []domain.BagPosition{"P3", "P4"}, "R2", "probe-2"))

				if _, err := s.OccupancyStart(movedID, OccupancyStartRequest{
					Operation: "start-moved", Generation: 1,
					RackID: "R1", ProbeID: "probe-1", ProbeStart: 10, ProbeEnd: 80,
				}); err != nil {
					t.Fatalf("start moved task: %v", err)
				}
				if _, err := s.OccupancyStart(occupantID, OccupancyStartRequest{
					Operation: "start-occupant", Generation: 1,
					RackID: "R2", ProbeID: "probe-2", ProbeStart: 10, ProbeEnd: 80,
				}); err != nil {
					t.Fatalf("start occupant task: %v", err)
				}
				_, err := s.OccupancyMoveRack(movedID, OccupancyMoveRequest{
					Operation: "move-conflict", Generation: 1,
					RackID: "R2", ProbeID: "probe-2", ProbeStart: 10, ProbeEnd: 80,
				})
				assertErrorCode(t, err, domain.CodeResourceWindowConflict)
				if got, want := activeRackProbe(t, s, movedID), "probe_window:probe-1,rack:R1"; got != want {
					t.Fatalf("active rack/probe leases after rejected move = %s, want %s", got, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
