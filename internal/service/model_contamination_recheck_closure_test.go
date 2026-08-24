package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

func TestModel_ContaminationRecheckClosure(t *testing.T) {
	type signal struct {
		position domain.BagPosition
		day      domain.DayAge
	}

	observeMatrix := func(t *testing.T, s *Service, id domain.TaskID, signals []signal) {
		t.Helper()
		cells := []struct {
			day      domain.DayAge
			position domain.BagPosition
		}{
			{1, "P1"}, {1, "P2"}, {2, "P1"}, {2, "P2"},
		}
		for i, cell := range cells {
			contaminationCount := 0
			for _, sig := range signals {
				if sig.position == cell.position && sig.day == cell.day {
					contaminationCount = 1
				}
			}
			res, err := s.Observation(id, ObservationRequest{
				Operation:          domain.OperationID("obs-" + itoa(i)),
				Generation:         1,
				DayAge:             cell.day,
				Position:           cell.position,
				MyceliumCoverage:   "85.0",
				ContaminationCount: contaminationCount,
				Observer:           "carol",
			})
			if err != nil {
				t.Fatalf("observe %s@%d: %v", cell.position, cell.day, err)
			}
			if i == len(cells)-1 && res.State != inspection.StateVerifyingContamination {
				t.Fatalf("matrix completion state = %s, want %s", res.State, inspection.StateVerifyingContamination)
			}
		}
	}

	detailOf := func(t *testing.T, s *Service, id domain.TaskID) TaskDetail {
		t.Helper()
		detail, err := s.GetTaskDetail(id)
		if err != nil {
			t.Fatalf("task detail: %v", err)
		}
		return detail
	}

	hasLateAudit := func(detail TaskDetail) bool {
		for _, event := range detail.Audit {
			if event.Action == domain.AuditLateReading {
				return true
			}
		}
		return false
	}

	cases := []struct {
		name    string
		signals []signal
		run     func(t *testing.T, s *Service, id domain.TaskID)
	}{
		{
			name:    "last negative recheck closes signal before physchem review gate",
			signals: []signal{{position: "P1", day: 1}},
			run: func(t *testing.T, s *Service, id domain.TaskID) {
				res, err := s.ContaminationRecheck(id, ContaminationRecheckRequest{
					Operation: "rc-1", Generation: 1, RecheckGeneration: 1,
					Position: "P1", DayAge: 1, Well: "A1", Source: "molecular", Positive: false,
					Summary: "negative molecular recheck",
				})
				if err != nil {
					t.Fatalf("recheck: %v", err)
				}
				if res.Version != 1 {
					t.Fatalf("version = %d, want 1", res.Version)
				}

				detail := detailOf(t, s, id)
				if detail.State != string(inspection.StateVerifyingPhysChem) {
					t.Fatalf("state after closing recheck = %s, want %s", detail.State, inspection.StateVerifyingPhysChem)
				}
				if len(detail.Evidence) != 1 {
					t.Fatalf("evidence count = %d, want 1", len(detail.Evidence))
				}
				if got := detail.Evidence[0]; got.Position != "P1" || got.DayAge != 1 || got.Positive {
					t.Fatalf("evidence = %+v, want negative P1@1", got)
				}

				progress, err := s.Progress(id)
				if err != nil {
					t.Fatalf("progress: %v", err)
				}
				if !progress.RecheckCovered {
					t.Fatal("recheck coverage should be closed after the final legal evidence")
				}

				submitPhysChem(t, s, id)
				detail = detailOf(t, s, id)
				if detail.State != string(inspection.StatePendingReview) {
					t.Fatalf("state after physchem = %s, want %s", detail.State, inspection.StatePendingReview)
				}
			},
		},
		{
			name:    "partial recheck coverage stays in contamination verification",
			signals: []signal{{position: "P1", day: 1}, {position: "P2", day: 2}},
			run: func(t *testing.T, s *Service, id domain.TaskID) {
				res, err := s.ContaminationRecheck(id, ContaminationRecheckRequest{
					Operation: "rc-1", Generation: 1, RecheckGeneration: 1,
					Position: "P1", DayAge: 1, Well: "A1", Source: "molecular", Positive: false,
				})
				if err != nil {
					t.Fatalf("recheck: %v", err)
				}
				if res.Version != 1 {
					t.Fatalf("version = %d, want 1", res.Version)
				}

				detail := detailOf(t, s, id)
				if detail.State != string(inspection.StateVerifyingContamination) {
					t.Fatalf("state after partial recheck = %s, want %s", detail.State, inspection.StateVerifyingContamination)
				}
				if len(detail.Evidence) != 1 {
					t.Fatalf("evidence count = %d, want 1", len(detail.Evidence))
				}

				progress, err := s.Progress(id)
				if err != nil {
					t.Fatalf("progress: %v", err)
				}
				if progress.RecheckCovered {
					t.Fatal("partial recheck coverage should not be closed")
				}
				if len(progress.ContaminationSignals) != 2 {
					t.Fatalf("contamination signals = %d, want 2", len(progress.ContaminationSignals))
				}

				submitPhysChem(t, s, id)
				detail = detailOf(t, s, id)
				if detail.State != string(inspection.StateVerifyingContamination) {
					t.Fatalf("state after physchem with partial recheck = %s, want %s", detail.State, inspection.StateVerifyingContamination)
				}
			},
		},
		{
			name:    "late recheck audits only and legal versions still increment",
			signals: []signal{{position: "P1", day: 1}},
			run: func(t *testing.T, s *Service, id domain.TaskID) {
				if _, err := s.ContaminationRecheck(id, ContaminationRecheckRequest{
					Operation: "late-rc", Generation: 1, RecheckGeneration: 0,
					Position: "P1", DayAge: 1, Well: "A1", Source: "molecular", Positive: false,
				}); err != nil {
					t.Fatalf("late recheck: %v", err)
				}
				detail := detailOf(t, s, id)
				if len(detail.Evidence) != 0 {
					t.Fatalf("late recheck evidence count = %d, want 0", len(detail.Evidence))
				}
				if !hasLateAudit(detail) {
					t.Fatal("late recheck should be audited")
				}

				first, err := s.ContaminationRecheck(id, ContaminationRecheckRequest{
					Operation: "rc-1", Generation: 1, RecheckGeneration: 1,
					Position: "P1", DayAge: 1, Well: "A1", Source: "molecular", Positive: false,
				})
				if err != nil {
					t.Fatalf("first legal recheck: %v", err)
				}
				if first.Version != 1 {
					t.Fatalf("first legal version = %d, want 1", first.Version)
				}
				detail = detailOf(t, s, id)
				if detail.State != string(inspection.StateVerifyingPhysChem) {
					t.Fatalf("state after first legal recheck = %s, want %s", detail.State, inspection.StateVerifyingPhysChem)
				}

				second, err := s.ContaminationRecheck(id, ContaminationRecheckRequest{
					Operation: "rc-2", Generation: 1, RecheckGeneration: 1,
					Position: "P1", DayAge: 1, Well: "A2", Source: "molecular", Positive: false,
				})
				if err != nil {
					t.Fatalf("second legal recheck: %v", err)
				}
				if second.Version != 2 {
					t.Fatalf("second legal version = %d, want 2", second.Version)
				}
				detail = detailOf(t, s, id)
				if len(detail.Evidence) != 2 {
					t.Fatalf("evidence count = %d, want 2", len(detail.Evidence))
				}
				if detail.Evidence[0].Version != 1 || detail.Evidence[1].Version != 2 {
					t.Fatalf("evidence versions = %d,%d; want 1,2", detail.Evidence[0].Version, detail.Evidence[1].Version)
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			s := newTestService(t)
			id := lockTask(t, s)
			advanceToObserving(t, s, id)
			observeMatrix(t, s, id, tc.signals)
			tc.run(t, s, id)
		})
	}
}
