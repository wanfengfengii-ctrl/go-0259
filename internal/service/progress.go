package service

import (
	"fmt"

	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/maturity"
)

// Progress is the operational progress view of a task: how much of the
// observation matrix, physico-chemical collection, contamination closure and
// review quota have been completed, plus any unmet finalize preconditions.
type Progress struct {
	TaskID               string                   `json:"task_id"`
	State                string                   `json:"state"`
	MatrixComplete       bool                     `json:"matrix_complete"`
	MissingCells         []string                 `json:"missing_cells"`
	PhysChemCollected    bool                     `json:"physchem_collected"`
	ContaminationSignals []string                 `json:"contamination_signals"`
	Coverage             maturity.CoverageSummary `json:"coverage"`
	DayCoverage          []maturity.DayCoverage   `json:"day_coverage"`
	RecheckCovered       bool                     `json:"recheck_covered"`
	ApprovedReviews      int                      `json:"approved_reviews"`
	ReadyForTransfer     bool                     `json:"ready_for_transfer"`
	UnmetPreconditions   []string                 `json:"unmet_preconditions,omitempty"`
}

// Progress computes the task progress view from the persisted state.
func (s *Service) Progress(id domain.TaskID) (Progress, error) {
	t, err := s.loadTask(id)
	if err != nil {
		return Progress{}, err
	}
	p := Progress{TaskID: string(id), State: string(t.State)}

	cells, _ := s.store.LoadCells(ctx(), id, t.Generation)
	missing := maturity.MissingCells(t.Snapshot.Schedule.DayAges, t.Snapshot.BagPositions, cells)
	p.MatrixComplete = len(missing) == 0
	for _, m := range missing {
		p.MissingCells = append(p.MissingCells, fmt.Sprintf("%s@%d", m.Position, m.DayAge))
	}
	if summary, err := maturity.SummarizeCoverage(cells); err == nil {
		p.Coverage = summary
	}
	p.DayCoverage = maturity.ReportDayCoverage(t.Snapshot.Schedule.DayAges, t.Snapshot.BagPositions, cells)

	readings, _ := s.store.LoadReadings(ctx(), id)
	p.PhysChemCollected = physChemCollected(t.Snapshot.BagPositions, readings)

	evidence, _ := s.store.LoadEvidence(ctx(), id)
	signals := arbiter.DetectSignals(t.Snapshot, cells)
	for _, sig := range signals {
		p.ContaminationSignals = append(p.ContaminationSignals,
			fmt.Sprintf("%s@%d:%s", sig.Position, sig.DayAge, sig.Reason))
	}
	p.RecheckCovered = s.recheckClosed(id, t)

	reviews, _ := s.store.LoadReviews(ctx(), id)
	approved := make(map[domain.PersonID]bool)
	for _, r := range reviews {
		if r.Conclusion == "approve" {
			approved[r.Person] = true
		}
	}
	p.ApprovedReviews = len(approved)

	eval := arbiter.EvaluateRequest{
		Snapshot: t.Snapshot, Cells: cells, Readings: readings,
		Evidence: evidence, Reviews: reviews, Conclusion: arbiter.ConclusionTransfer,
	}
	items := arbiter.Checklist(eval)
	p.ReadyForTransfer = len(items) == 0
	for _, it := range items {
		p.UnmetPreconditions = append(p.UnmetPreconditions, it.Code+": "+it.Reason)
	}
	return p, nil
}
