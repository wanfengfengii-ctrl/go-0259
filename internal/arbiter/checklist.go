package arbiter

import (
	"fmt"

	"mycocycle-growbag-transfer-gate/internal/contamination"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/maturity"
)

// ChecklistItem is a single unmet finalization precondition.
type ChecklistItem struct {
	Code   string `json:"code"`
	Reason string `json:"reason"`
}

// Checklist returns the ordered list of unmet transfer preconditions for a
// task. An empty list means the task is ready to be transferred.
func Checklist(req EvaluateRequest) []ChecklistItem {
	snap := req.Snapshot
	var out []ChecklistItem

	if missing := maturity.MissingCells(snap.Schedule.DayAges, snap.BagPositions, req.Cells); len(missing) > 0 {
		for _, m := range missing {
			out = append(out, ChecklistItem{
				Code:   "matrix_incomplete",
				Reason: fmt.Sprintf("missing observation %s@%d", m.Position, m.DayAge),
			})
		}
	}
	for _, c := range req.Cells {
		if !domain.WithinRange(c.MyceliumCoverage, snap.Thresholds.MaturityMin, snap.Thresholds.MaturityMax) {
			out = append(out, ChecklistItem{
				Code:   "maturity_out_of_range",
				Reason: fmt.Sprintf("coverage %s@%d outside maturity range", c.Position, c.DayAge),
			})
		}
	}
	if contamination.AnyPositive(req.Evidence) {
		out = append(out, ChecklistItem{Code: "contamination_positive", Reason: "positive contamination evidence present"})
	}
	signals := DetectSignals(snap, req.Cells)
	if len(signals) > 0 {
		affected := make([]contamination.EvidenceKey, 0, len(signals))
		for _, s := range signals {
			affected = append(affected, contamination.EvidenceKey{Position: s.Position, DayAge: s.DayAge})
		}
		if missing := contamination.Covered(req.Evidence, affected); len(missing) > 0 {
			for _, m := range missing {
				out = append(out, ChecklistItem{
					Code:   "recheck_coverage_incomplete",
					Reason: fmt.Sprintf("recheck missing coverage %s@%d", m.Position, m.DayAge),
				})
			}
		}
	}
	if err := checkPhysChem(snap, req.Readings); err != nil {
		out = append(out, ChecklistItem{Code: "physchem_unmet", Reason: err.Error()})
	}
	if err := checkReviews(snap, req.Reviews); err != nil {
		out = append(out, ChecklistItem{Code: "reviews_insufficient", Reason: err.Error()})
	}
	return out
}
