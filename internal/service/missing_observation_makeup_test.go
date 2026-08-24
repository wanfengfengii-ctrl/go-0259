package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

func TestModel_MissingObservationMakeupClosesProgress(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "same generation missing cell accepts one non missing makeup",
			run: func(t *testing.T) {
				s := newTestService(t)
				id := lockTask(t, s)
				advanceToObserving(t, s, id)

				if res, err := s.Observation(id, ObservationRequest{
					Operation: "missing-p1-day1", Generation: 1,
					DayAge: 1, Position: "P1", Missing: true,
					Summary: "observer missed bag", Observer: "carol",
				}); err != nil {
					t.Fatalf("missing observation: %v", err)
				} else if res.State != inspection.StateObserving || res.Missing != 4 {
					t.Fatalf("missing result = state %s missing %d, want observing and 4 missing cells", res.State, res.Missing)
				}

				progress, err := s.Progress(id)
				if err != nil {
					t.Fatalf("progress after missing: %v", err)
				}
				if len(progress.MissingCells) != 4 || progress.MissingCells[0] != "P1@1" {
					t.Fatalf("missing cells after missing observation = %#v, want P1@1 still open in full matrix", progress.MissingCells)
				}

				for _, c := range []struct {
					op  domain.OperationID
					day domain.DayAge
					pos domain.BagPosition
				}{
					{op: "valid-p2-day1", day: 1, pos: "P2"},
					{op: "valid-p1-day2", day: 2, pos: "P1"},
					{op: "valid-p2-day2", day: 2, pos: "P2"},
				} {
					if _, err := s.Observation(id, ObservationRequest{
						Operation: c.op, Generation: 1,
						DayAge: c.day, Position: c.pos,
						MyceliumCoverage: "85.0", Observer: "carol",
					}); err != nil {
						t.Fatalf("observe %s: %v", c.op, err)
					}
				}

				res, err := s.Observation(id, ObservationRequest{
					Operation: "makeup-p1-day1", Generation: 1,
					DayAge: 1, Position: "P1",
					MyceliumCoverage: "85.0", ContaminationCount: 0,
					Summary: "makeup measurement", Observer: "dave",
				})
				if err != nil {
					t.Fatalf("makeup observation should close missing cell: %v", err)
				}
				if res.State != inspection.StateVerifyingContamination || res.Missing != 0 {
					t.Fatalf("makeup result = state %s missing %d, want verifying_contamination and 0 missing cells", res.State, res.Missing)
				}

				progress, err = s.Progress(id)
				if err != nil {
					t.Fatalf("progress after makeup: %v", err)
				}
				if !progress.MatrixComplete || len(progress.MissingCells) != 0 {
					t.Fatalf("progress after makeup = complete %v missing %#v, want complete matrix", progress.MatrixComplete, progress.MissingCells)
				}

				detail, err := s.GetTaskDetail(id)
				if err != nil {
					t.Fatalf("detail after makeup: %v", err)
				}
				for _, cell := range detail.Cells {
					if cell.DayAge == 1 && cell.Position == "P1" {
						if cell.Missing {
							t.Fatalf("P1@1 cell remains missing after makeup")
						}
						if cell.MyceliumCoverage.Value != 850 || cell.MyceliumCoverage.Scale != domain.CoverageScale {
							t.Fatalf("P1@1 coverage = %d scale %d, want 850 scale %d", cell.MyceliumCoverage.Value, cell.MyceliumCoverage.Scale, domain.CoverageScale)
						}
						return
					}
				}
				t.Fatalf("P1@1 cell not found after makeup")
			},
		},
		{
			name: "already valid cell still rejects later write",
			run: func(t *testing.T) {
				s := newTestService(t)
				id := lockTask(t, s)
				advanceToObserving(t, s, id)

				if _, err := s.Observation(id, ObservationRequest{
					Operation: "valid-p1-day1", Generation: 1,
					DayAge: 1, Position: "P1",
					MyceliumCoverage: "85.0", Observer: "carol",
				}); err != nil {
					t.Fatalf("valid observation: %v", err)
				}

				_, err := s.Observation(id, ObservationRequest{
					Operation: "overwrite-p1-day1", Generation: 1,
					DayAge: 1, Position: "P1",
					MyceliumCoverage: "86.0", Observer: "dave",
				})
				assertErrorCode(t, err, domain.CodeIdempotencyConflict)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
