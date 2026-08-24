package service

import (
	"errors"
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/store"
)

func TestModel_FinalizePreconditionErrorsDoNotCloseTask(t *testing.T) {
	cases := []struct {
		name           string
		arrange        func(*testing.T, *Service, domain.TaskID)
		finalize       FinalizeRequest
		wantErr        domain.ErrorCode
		assertWritable func(*testing.T, *Service, domain.TaskID)
	}{
		{
			name: "transfer rejected before physchem readings leaves task writable",
			finalize: FinalizeRequest{
				Operation:  "finalize-missing-physchem",
				Generation: 1,
				Conclusion: "transfer",
			},
			wantErr: domain.CodeReadingOutOfRange,
			assertWritable: func(t *testing.T, s *Service, id domain.TaskID) {
				t.Helper()
				res, err := s.DeviceReading(id, DeviceReadingRequest{
					Operation:  "reading-after-failed-finalize",
					Generation: 1,
					DeviceType: domain.DeviceMoisture,
					DeviceID:   "moisture-after-finalize",
					Metric:     domain.MetricMoisture,
					Value:      "65.0",
					Position:   "P1",
				})
				if err != nil {
					t.Fatalf("device reading after failed finalize: %v", err)
				}
				if res.Result != domain.AttemptAccepted {
					t.Fatalf("device reading result = %s, want accepted", res.Result)
				}
			},
		},
		{
			name: "transfer rejected before independent reviews leaves task writable",
			arrange: func(t *testing.T, s *Service, id domain.TaskID) {
				t.Helper()
				submitPhysChem(t, s, id)
			},
			finalize: FinalizeRequest{
				Operation:  "finalize-missing-reviews",
				Generation: 1,
				Conclusion: "transfer",
			},
			wantErr: domain.CodeRoleOverlap,
			assertWritable: func(t *testing.T, s *Service, id domain.TaskID) {
				t.Helper()
				if _, err := s.Review(id, ReviewRequest{
					Operation:  "review-after-failed-finalize",
					Generation: 1,
					Person:     "carol",
					Conclusion: "approve",
				}); err != nil {
					t.Fatalf("review after failed finalize: %v", err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestService(t)
			id := toVerifying(t, s)
			if tc.arrange != nil {
				tc.arrange(t, s, id)
			}

			beforeTask, err := s.store.LoadTask(ctx(), id)
			if err != nil {
				t.Fatalf("load task before finalize: %v", err)
			}

			_, err = s.Finalize(id, tc.finalize)
			assertErrorCode(t, err, tc.wantErr)

			view, err := s.GetTask(id)
			if err != nil {
				t.Fatalf("get task after failed finalize: %v", err)
			}
			if view.State != beforeTask.State {
				t.Fatalf("GET task state after failed finalize = %s, want %s", view.State, beforeTask.State)
			}
			if view.FinalType != "" || view.Credential != "" {
				t.Fatalf("GET task exposed final decision after failed finalize: final_type=%q credential=%q", view.FinalType, view.Credential)
			}

			afterTask, err := s.store.LoadTask(ctx(), id)
			if err != nil {
				t.Fatalf("load task after failed finalize: %v", err)
			}
			if afterTask.FinalVersion != beforeTask.FinalVersion {
				t.Fatalf("final_version after failed finalize = %d, want %d", afterTask.FinalVersion, beforeTask.FinalVersion)
			}
			if afterTask.State != beforeTask.State {
				t.Fatalf("stored state after failed finalize = %s, want %s", afterTask.State, beforeTask.State)
			}
			if _, err := s.store.LoadDecision(ctx(), id); !errors.Is(err, store.ErrNotFound) {
				t.Fatalf("final decision after failed finalize error = %v, want not found", err)
			}

			tc.assertWritable(t, s, id)
		})
	}
}
