package service

import (
	"errors"
	"slices"
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
	"mycocycle-growbag-transfer-gate/internal/occupancy"
)

func TestModel_OccupancyStartRollsBackPartialLeaseSet(t *testing.T) {
	cases := []struct {
		name                string
		activeStart         OccupancyStartRequest
		conflictingStart    OccupancyStartRequest
		retryStart          OccupancyStartRequest
		wantConflictReasons []string
	}{
		{
			name: "probe conflict does not retain candidate rack",
			activeStart: OccupancyStartRequest{
				Operation: "active-start", Generation: 1, RackID: "R1",
				ProbeID: "probe-1", ProbeStart: 1, ProbeEnd: 100,
			},
			conflictingStart: OccupancyStartRequest{
				Operation: "conflicting-start", Generation: 1, RackID: "R2",
				ProbeID: "probe-1", ProbeStart: 40, ProbeEnd: 80,
			},
			retryStart: OccupancyStartRequest{
				Operation: "retry-start", Generation: 1, RackID: "R2",
				ProbeID: "probe-2", ProbeStart: 40, ProbeEnd: 80,
			},
			wantConflictReasons: []string{"probe_window:probe-1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestService(t)
			activeTask := lockConfirmSeal(t, s, lockReqFor("B-001", []domain.BagPosition{"P1", "P2"}, "R1", "probe-1"))
			candidateTask := lockConfirmSeal(t, s, lockReqFor("B-002", []domain.BagPosition{"P3", "P4"}, "R2", "probe-1"))

			activeOccupancyLeases := func(id domain.TaskID) []occupancy.Lease {
				t.Helper()
				detail, err := s.GetTaskDetail(id)
				if err != nil {
					t.Fatalf("detail %s: %v", id, err)
				}
				var out []occupancy.Lease
				for _, l := range detail.Leases {
					if l.State != occupancy.LeaseActive {
						continue
					}
					if l.ResourceType == occupancy.ResourceRack || l.ResourceType == occupancy.ResourceProbeWindow {
						out = append(out, l)
					}
				}
				return out
			}

			if got, err := s.OccupancyStart(activeTask, tc.activeStart); err != nil {
				t.Fatalf("active start: %v", err)
			} else if got.State != inspection.StateObserving {
				t.Fatalf("active start state = %s, want %s", got.State, inspection.StateObserving)
			}

			_, err := s.OccupancyStart(candidateTask, tc.conflictingStart)
			var de *domain.Error
			if !errors.As(err, &de) {
				t.Fatalf("conflicting start error = %v, want domain error", err)
			}
			if de.Code != domain.CodeResourceWindowConflict {
				t.Fatalf("conflicting start code = %q, want %q", de.Code, domain.CodeResourceWindowConflict)
			}
			if !slices.Equal(de.Reasons, tc.wantConflictReasons) {
				t.Fatalf("conflicting start reasons = %v, want %v", de.Reasons, tc.wantConflictReasons)
			}

			if got := activeOccupancyLeases(candidateTask); len(got) != 0 {
				t.Fatalf("failed start left active rack/probe leases: %+v", got)
			}
			if got := activeOccupancyLeases(activeTask); len(got) != 2 {
				t.Fatalf("existing task active rack/probe leases = %+v, want rack and probe window", got)
			}

			got, err := s.OccupancyStart(candidateTask, tc.retryStart)
			if err != nil {
				t.Fatalf("retry with non-conflicting probe: %v", err)
			}
			if got.State != inspection.StateObserving {
				t.Fatalf("retry state = %s, want %s", got.State, inspection.StateObserving)
			}

			leases := activeOccupancyLeases(candidateTask)
			want := map[occupancy.ResourceType]string{
				occupancy.ResourceRack:        string(tc.retryStart.RackID),
				occupancy.ResourceProbeWindow: string(tc.retryStart.ProbeID),
			}
			if len(leases) != len(want) {
				t.Fatalf("retry active rack/probe leases = %+v, want %v", leases, want)
			}
			for _, l := range leases {
				if want[l.ResourceType] != l.ResourceID {
					t.Fatalf("retry lease = %+v, want resource id %q", l, want[l.ResourceType])
				}
			}
		})
	}
}
