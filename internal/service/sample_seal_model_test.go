package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

func TestModel_PartialSampleSealRequiresAllLockedPositions(t *testing.T) {
	type sealAttempt struct {
		operation            domain.OperationID
		person               domain.PersonID
		positions            []domain.BagPosition
		wantErr              domain.ErrorCode
		wantResultState      inspection.TaskState
		wantDetailState      inspection.TaskState
		wantSealed           map[domain.BagPosition]bool
		wantOccupancyBlocked bool
	}

	tests := []struct {
		name                    string
		attempts                []sealAttempt
		wantFinalOccupancyStart bool
	}{
		{
			name: "partial seal only updates matched bag and keeps occupancy closed",
			attempts: []sealAttempt{
				{
					operation:            "seal-p1",
					person:               "alice",
					positions:            []domain.BagPosition{"P1"},
					wantResultState:      inspection.StateSealingSamples,
					wantDetailState:      inspection.StateSealingSamples,
					wantSealed:           map[domain.BagPosition]bool{"P1": true, "P2": false},
					wantOccupancyBlocked: true,
				},
				{
					operation:       "seal-p2",
					person:          "alice",
					positions:       []domain.BagPosition{"P2"},
					wantResultState: inspection.StateOccupying,
					wantDetailState: inspection.StateOccupying,
					wantSealed:      map[domain.BagPosition]bool{"P1": true, "P2": true},
				},
			},
			wantFinalOccupancyStart: true,
		},
		{
			name: "complete seal advances immediately",
			attempts: []sealAttempt{
				{
					operation:       "seal-all",
					person:          "alice",
					positions:       []domain.BagPosition{"P1", "P2"},
					wantResultState: inspection.StateOccupying,
					wantDetailState: inspection.StateOccupying,
					wantSealed:      map[domain.BagPosition]bool{"P1": true, "P2": true},
				},
			},
			wantFinalOccupancyStart: true,
		},
		{
			name: "unmatched position is rejected without sealing",
			attempts: []sealAttempt{
				{
					operation:            "seal-missing",
					person:               "alice",
					positions:            []domain.BagPosition{"P9"},
					wantErr:              domain.CodeDuplicateBagPosition,
					wantDetailState:      inspection.StateSealingSamples,
					wantSealed:           map[domain.BagPosition]bool{"P1": false, "P2": false},
					wantOccupancyBlocked: true,
				},
			},
		},
		{
			name: "unqualified sealer is rejected without sealing",
			attempts: []sealAttempt{
				{
					operation:            "seal-unqualified",
					person:               "mallory",
					positions:            []domain.BagPosition{"P1"},
					wantErr:              domain.CodeRoleOverlap,
					wantDetailState:      inspection.StateSealingSamples,
					wantSealed:           map[domain.BagPosition]bool{"P1": false, "P2": false},
					wantOccupancyBlocked: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService(t)
			id := lockTask(t, s)
			mustSamplingConfirm(t, s, id, "confirm-a", "alice")
			mustSamplingConfirm(t, s, id, "confirm-b", "bob")

			checkDetail := func(step sealAttempt) {
				t.Helper()
				detail, err := s.GetTaskDetail(id)
				if err != nil {
					t.Fatalf("detail: %v", err)
				}
				if detail.State != string(step.wantDetailState) {
					t.Fatalf("detail state = %s, want %s", detail.State, step.wantDetailState)
				}
				if len(detail.BagPositions) != len(step.wantSealed) {
					t.Fatalf("bag positions = %d, want %d", len(detail.BagPositions), len(step.wantSealed))
				}
				for _, p := range detail.BagPositions {
					want, ok := step.wantSealed[p.Position]
					if !ok {
						t.Fatalf("unexpected bag position in detail: %s", p.Position)
					}
					if p.Sealed != want {
						t.Fatalf("%s sealed = %v, want %v", p.Position, p.Sealed, want)
					}
					if !p.Sealed && (p.SealedBy != "" || p.SealedAt != 0) {
						t.Fatalf("%s has seal metadata while unsealed: %+v", p.Position, p)
					}
				}
			}

			for _, attempt := range tt.attempts {
				res, err := s.SampleSeal(id, SampleSealRequest{
					Operation:  attempt.operation,
					Generation: 1,
					Person:     attempt.person,
					Positions:  attempt.positions,
				})
				if attempt.wantErr != "" {
					assertErrorCode(t, err, attempt.wantErr)
				} else if err != nil {
					t.Fatalf("sample seal %s: %v", attempt.operation, err)
				} else if res.State != attempt.wantResultState {
					t.Fatalf("sample seal %s state = %s, want %s", attempt.operation, res.State, attempt.wantResultState)
				}

				checkDetail(attempt)

				if attempt.wantOccupancyBlocked {
					_, err := s.OccupancyStart(id, OccupancyStartRequest{
						Operation:  domain.OperationID("start-blocked-" + string(attempt.operation)),
						Generation: 1,
						RackID:     "R1",
						ProbeID:    "probe-1",
						ProbeStart: 1,
						ProbeEnd:   100,
					})
					assertErrorCode(t, err, domain.CodeFinalStateRejected)
				}
			}

			if tt.wantFinalOccupancyStart {
				res, err := s.OccupancyStart(id, OccupancyStartRequest{
					Operation:  "start-final",
					Generation: 1,
					RackID:     "R1",
					ProbeID:    "probe-1",
					ProbeStart: 1,
					ProbeEnd:   100,
				})
				if err != nil {
					t.Fatalf("occupancy start after all sealed: %v", err)
				}
				if res.State != inspection.StateObserving {
					t.Fatalf("occupancy state = %s, want %s", res.State, inspection.StateObserving)
				}
			}
		})
	}
}
