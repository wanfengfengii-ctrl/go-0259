package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

func TestModel_FinalizeCancelBypassesTransferPrerequisites(t *testing.T) {
	cases := []struct {
		name          string
		conclusion    string
		wantFinalType string
		wantState     inspection.TaskState
		wantErr       domain.ErrorCode
	}{
		{
			name:          "cancel succeeds without contamination recheck physchem or review quorum",
			conclusion:    "cancel",
			wantFinalType: string(arbiter.FinalCancelled),
			wantState:     inspection.StateCancelled,
		},
		{
			name:       "transfer remains blocked by the same missing prerequisites",
			conclusion: "transfer",
			wantErr:    domain.CodeRecheckInsufficient,
		},
		{
			name:          "isolate remains driven by the contamination signal",
			conclusion:    "isolate",
			wantFinalType: string(arbiter.FinalContaminationIsolated),
			wantState:     inspection.StateContaminationIsolated,
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
					t.Fatalf("observation %d: %v", i, err)
				}
			}

			view, err := s.GetTask(id)
			if err != nil {
				t.Fatalf("get task: %v", err)
			}
			if view.State != inspection.StateVerifyingContamination {
				t.Fatalf("state = %s, want verifying_contamination", view.State)
			}

			res, err := s.Finalize(id, FinalizeRequest{
				Operation:  domain.OperationID("final-" + tc.conclusion),
				Generation: 1,
				Conclusion: tc.conclusion,
			})
			if tc.wantErr != "" {
				assertErrorCode(t, err, tc.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("finalize %s: %v", tc.conclusion, err)
			}
			if res.FinalType != tc.wantFinalType {
				t.Fatalf("final type = %s, want %s", res.FinalType, tc.wantFinalType)
			}
			if res.State != tc.wantState {
				t.Fatalf("result state = %s, want %s", res.State, tc.wantState)
			}

			view, err = s.GetTask(id)
			if err != nil {
				t.Fatalf("get finalized task: %v", err)
			}
			if view.State != tc.wantState {
				t.Fatalf("persisted state = %s, want %s", view.State, tc.wantState)
			}
		})
	}
}
