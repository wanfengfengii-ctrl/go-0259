package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

func TestModel_HistoricalPositiveContaminationEvidenceBlocksTransfer(t *testing.T) {
	cases := []struct {
		name         string
		positives    []bool
		wantTransfer bool
	}{
		{
			name:         "historical positive followed by negative blocks transfer until isolate",
			positives:    []bool{true, false},
			wantTransfer: false,
		},
		{
			name:         "pure negative rechecks keep transfer path open",
			positives:    []bool{false, false},
			wantTransfer: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestService(t)
			id := lockTask(t, s)
			advanceToObserving(t, s, id)

			observations := []struct {
				day   domain.DayAge
				pos   domain.BagPosition
				count int
			}{
				{day: 1, pos: "P1", count: 1},
				{day: 1, pos: "P2"},
				{day: 2, pos: "P1"},
				{day: 2, pos: "P2"},
			}
			for i, obs := range observations {
				if _, err := s.Observation(id, ObservationRequest{
					Operation:          domain.OperationID("obs-" + itoa(i)),
					Generation:         1,
					DayAge:             obs.day,
					Position:           obs.pos,
					MyceliumCoverage:   "85.0",
					ContaminationCount: obs.count,
					Observer:           "carol",
				}); err != nil {
					t.Fatalf("observe %s@%d: %v", obs.pos, obs.day, err)
				}
			}

			for i, positive := range tc.positives {
				res, err := s.ContaminationRecheck(id, ContaminationRecheckRequest{
					Operation:         domain.OperationID("recheck-" + itoa(i)),
					Generation:        1,
					RecheckGeneration: 1,
					Position:          "P1",
					DayAge:            1,
					Well:              "A" + itoa(i+1),
					Source:            "molecular",
					Positive:          positive,
				})
				if err != nil {
					t.Fatalf("recheck %d: %v", i, err)
				}
				if res.Version != i+1 {
					t.Fatalf("recheck %d version = %d, want %d", i, res.Version, i+1)
				}
			}

			evidence, err := s.store.LoadEvidence(ctx(), id)
			if err != nil {
				t.Fatalf("load evidence: %v", err)
			}
			if len(evidence) != len(tc.positives) {
				t.Fatalf("evidence count = %d, want %d", len(evidence), len(tc.positives))
			}
			for i, ev := range evidence {
				if ev.Position != "P1" || ev.DayAge != 1 {
					t.Fatalf("evidence %d slot = %s@%d, want P1@1", i, ev.Position, ev.DayAge)
				}
				if ev.Version != i+1 {
					t.Fatalf("evidence %d version = %d, want %d", i, ev.Version, i+1)
				}
				if ev.Positive != tc.positives[i] {
					t.Fatalf("evidence %d positive = %v, want %v", i, ev.Positive, tc.positives[i])
				}
			}

			submitPhysChem(t, s, id)
			reviewApprove(t, s, id, "review-1", "carol")
			reviewApprove(t, s, id, "review-2", "dave")

			transfer, err := s.Finalize(id, FinalizeRequest{
				Operation:  "final-transfer",
				Generation: 1,
				Conclusion: "transfer",
			})
			if tc.wantTransfer {
				if err != nil {
					t.Fatalf("finalize transfer: %v", err)
				}
				if transfer.FinalType != string(arbiter.FinalTransferable) {
					t.Fatalf("final type = %s, want transferable", transfer.FinalType)
				}
				if transfer.Credential == "" {
					t.Fatal("transfer credential should be present")
				}
				if transfer.State != inspection.StateTransferable {
					t.Fatalf("state = %s, want transferable", transfer.State)
				}
				return
			}

			assertErrorCode(t, err, domain.CodeRecheckInsufficient)
			if transfer.Credential != "" {
				t.Fatalf("blocked transfer credential = %q, want empty", transfer.Credential)
			}

			isolated, err := s.Finalize(id, FinalizeRequest{
				Operation:  "final-isolate",
				Generation: 1,
				Conclusion: "isolate",
			})
			if err != nil {
				t.Fatalf("finalize isolate: %v", err)
			}
			if isolated.FinalType != string(arbiter.FinalContaminationIsolated) {
				t.Fatalf("final type = %s, want contamination_isolated", isolated.FinalType)
			}
			if isolated.State != inspection.StateContaminationIsolated {
				t.Fatalf("state = %s, want contamination_isolated", isolated.State)
			}
		})
	}
}
