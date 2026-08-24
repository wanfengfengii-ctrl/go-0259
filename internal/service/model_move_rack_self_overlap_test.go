package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/occupancy"
)

func TestModel_MoveRackReplacesOwnRackProbeBeforeAcquire(t *testing.T) {
	type occupancyPlan struct {
		batch     domain.BagBatch
		positions []domain.BagPosition
		rack      domain.RackID
		probe     domain.ProbeID
		start     domain.LogicalTime
		end       domain.LogicalTime
	}

	primary := occupancyPlan{
		batch:     "B-001",
		positions: []domain.BagPosition{"P1", "P2"},
		rack:      "R1",
		probe:     "probe-1",
		start:     1,
		end:       100,
	}

	cases := []struct {
		name      string
		blockers  []occupancyPlan
		move      OccupancyMoveRequest
		wantErr   domain.ErrorCode
		wantRack  domain.RackID
		wantProbe domain.ProbeID
		wantStart domain.LogicalTime
		wantEnd   domain.LogicalTime
	}{
		{
			name: "self-overlapping probe window is atomically replaced",
			move: OccupancyMoveRequest{
				Operation: "move-self-overlap", Generation: 1,
				RackID: "R2", ProbeID: "probe-1", ProbeStart: 50, ProbeEnd: 150,
			},
			wantRack:  "R2",
			wantProbe: "probe-1",
			wantStart: 50,
			wantEnd:   150,
		},
		{
			name: "other task probe overlap is rejected without half update",
			blockers: []occupancyPlan{
				{
					batch:     "B-002",
					positions: []domain.BagPosition{"P3", "P4"},
					rack:      "R3",
					probe:     "probe-1",
					start:     125,
					end:       175,
				},
			},
			move: OccupancyMoveRequest{
				Operation: "move-into-other-probe", Generation: 1,
				RackID: "R2", ProbeID: "probe-1", ProbeStart: 100, ProbeEnd: 150,
			},
			wantErr:   domain.CodeResourceWindowConflict,
			wantRack:  "R1",
			wantProbe: "probe-1",
			wantStart: 1,
			wantEnd:   100,
		},
		{
			name: "ordinary adjacent move remains available",
			move: OccupancyMoveRequest{
				Operation: "move-adjacent", Generation: 1,
				RackID: "R2", ProbeID: "probe-1", ProbeStart: 100, ProbeEnd: 150,
			},
			wantRack:  "R2",
			wantProbe: "probe-1",
			wantStart: 100,
			wantEnd:   150,
		},
	}

	startObserving := func(t *testing.T, s *Service, p occupancyPlan, op domain.OperationID) domain.TaskID {
		t.Helper()
		id := lockConfirmSeal(t, s, lockReqFor(p.batch, p.positions, p.rack, p.probe))
		if _, err := s.OccupancyStart(id, OccupancyStartRequest{
			Operation: op, Generation: 1,
			RackID: p.rack, ProbeID: p.probe, ProbeStart: p.start, ProbeEnd: p.end,
		}); err != nil {
			t.Fatalf("start observing %s: %v", p.batch, err)
		}
		return id
	}

	assertActiveOccupancy := func(t *testing.T, s *Service, id domain.TaskID, rack domain.RackID, probe domain.ProbeID, start, end domain.LogicalTime) {
		t.Helper()
		detail, err := s.GetTaskDetail(id)
		if err != nil {
			t.Fatalf("detail: %v", err)
		}
		if detail.State != "observing" {
			t.Fatalf("state = %q, want observing", detail.State)
		}

		var activeRacks, activeProbes int
		var gotRack, gotProbe string
		var gotStart, gotEnd domain.LogicalTime
		for _, lease := range detail.Leases {
			if lease.State != occupancy.LeaseActive {
				continue
			}
			switch lease.ResourceType {
			case occupancy.ResourceRack:
				activeRacks++
				gotRack = lease.ResourceID
			case occupancy.ResourceProbeWindow:
				activeProbes++
				gotProbe = lease.ResourceID
				gotStart = lease.WindowStart
				gotEnd = lease.WindowEnd
			}
		}

		if activeRacks != 1 || activeProbes != 1 {
			t.Fatalf("active rack/probe leases = %d/%d, want 1/1", activeRacks, activeProbes)
		}
		if gotRack != string(rack) {
			t.Fatalf("active rack = %q, want %q", gotRack, rack)
		}
		if gotProbe != string(probe) || gotStart != start || gotEnd != end {
			t.Fatalf("active probe window = %s [%d,%d), want %s [%d,%d)",
				gotProbe, gotStart, gotEnd, probe, start, end)
		}
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestService(t)
			id := startObserving(t, s, primary, "start-primary")
			for i, blocker := range tc.blockers {
				startObserving(t, s, blocker, domain.OperationID("start-blocker-"+itoa(i)))
			}

			_, err := s.OccupancyMoveRack(id, tc.move)
			if tc.wantErr != "" {
				assertErrorCode(t, err, tc.wantErr)
			} else if err != nil {
				t.Fatalf("move rack: %v", err)
			}
			assertActiveOccupancy(t, s, id, tc.wantRack, tc.wantProbe, tc.wantStart, tc.wantEnd)
		})
	}
}
