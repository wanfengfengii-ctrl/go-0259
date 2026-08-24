package arbiter

import (
	"fmt"

	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/contamination"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/maturity"
)

// EvaluateRequest carries every input needed to judge a finalization.
type EvaluateRequest struct {
	Snapshot catalog.Snapshot
	Cells    []maturity.ObservationCell
	Readings []maturity.PhysChemReading
	Evidence []contamination.Evidence
	Reviews  []Review
	Conclusion
}

// Signal is a contamination recheck trigger slot.
type Signal struct {
	Position domain.BagPosition
	DayAge   domain.DayAge
	Reason   string
}

// DetectSignals returns the recheck-trigger slots derived from observation
// cells (bag damage and over-threshold contamination counts). Positive
// molecular evidence is handled separately as an outright positive signal.
func DetectSignals(snap catalog.Snapshot, cells []maturity.ObservationCell) []Signal {
	var out []Signal
	for _, c := range cells {
		if c.BagDamage {
			out = append(out, Signal{Position: c.Position, DayAge: c.DayAge, Reason: "bag_damage"})
		}
		if int64(c.ContaminationCount) > snap.Thresholds.Contamination.Value {
			out = append(out, Signal{Position: c.Position, DayAge: c.DayAge, Reason: "contamination_count_over_threshold"})
		}
	}
	return out
}

// Evaluate judges whether the requested conclusion is reachable and returns the
// corresponding terminal type and a credential, or a stable domain error.
func Evaluate(req EvaluateRequest) (FinalType, string, *domain.Error) {
	snap := req.Snapshot

	// 1. Coverage matrix must be complete.
	if missing := maturity.MissingCells(snap.Schedule.DayAges, snap.BagPositions, req.Cells); len(missing) > 0 {
		reasons := make([]string, 0, len(missing))
		for _, m := range missing {
			reasons = append(reasons, fmt.Sprintf("%s@%d", m.Position, m.DayAge))
		}
		return "", "", domain.NewError(domain.CodeRecheckInsufficient,
			"observation matrix incomplete", reasons...)
	}

	// 2. Maturity must be within the locked maturity interval for every cell.
	for _, c := range req.Cells {
		if !domain.WithinRange(c.MyceliumCoverage, snap.Thresholds.MaturityMin, snap.Thresholds.MaturityMax) {
			return "", "", domain.NewError(domain.CodeReadingOutOfRange,
				"mycelium coverage outside maturity threshold",
				fmt.Sprintf("%s@%d", c.Position, c.DayAge))
		}
	}

	// 3. Contamination signals must be closed (covered by negative evidence).
	signals := DetectSignals(snap, req.Cells)
	positiveEvidence := contamination.AnyPositive(req.Evidence)

	if req.Conclusion == ConclusionCancel {
		return FinalCancelled, "", nil
	}

	if req.Conclusion == ConclusionIsolate {
		if !positiveEvidence && len(signals) == 0 {
			return "", "", domain.NewError(domain.CodeRecheckInsufficient,
				"no contamination signal to isolate")
		}
		return FinalContaminationIsolated, "", nil
	}

	// transfer
	if positiveEvidence {
		return "", "", domain.NewError(domain.CodeRecheckInsufficient,
			"positive contamination evidence present")
	}
	if len(signals) > 0 {
		affected := make([]contamination.EvidenceKey, 0, len(signals))
		for _, s := range signals {
			affected = append(affected, contamination.EvidenceKey{Position: s.Position, DayAge: s.DayAge})
		}
		if missing := contamination.Covered(req.Evidence, affected); len(missing) > 0 {
			reasons := make([]string, 0, len(missing))
			for _, m := range missing {
				reasons = append(reasons, fmt.Sprintf("%s@%d", m.Position, m.DayAge))
			}
			return "", "", domain.NewError(domain.CodeRecheckInsufficient,
				"contamination recheck coverage incomplete", reasons...)
		}
	}

	// 4. Physico-chemical moisture and pH per bag position within thresholds.
	if err := checkPhysChem(snap, req.Readings); err != nil {
		return "", "", err
	}

	// 5. Two distinct qualified independent reviews.
	if err := checkReviews(snap, req.Reviews); err != nil {
		return "", "", err
	}

	return FinalTransferable, "", nil
}

func checkPhysChem(snap catalog.Snapshot, readings []maturity.PhysChemReading) *domain.Error {
	moistOK := false
	phOK := false
	for _, r := range readings {
		if !r.Accepted() {
			continue
		}
		switch r.Metric {
		case domain.MetricMoisture:
			if domain.WithinRange(r.Value, snap.Thresholds.MoistureMin, snap.Thresholds.MoistureMax) {
				moistOK = true
			}
		case domain.MetricPH:
			if domain.WithinRange(r.Value, snap.Thresholds.PHMin, snap.Thresholds.PHMax) {
				phOK = true
			}
		}
	}
	for _, pos := range snap.BagPositions {
		if !moistOK || !phOK {
			return domain.NewError(domain.CodeReadingOutOfRange,
				"physico-chemical thresholds not met", string(pos))
		}
	}
	return nil
}

func checkReviews(snap catalog.Snapshot, reviews []Review) *domain.Error {
	eligible := make(map[domain.PersonID]bool, len(snap.Reviewers))
	for _, p := range snap.Reviewers {
		eligible[p] = true
	}
	seen := make(map[domain.PersonID]bool)
	approved := 0
	for _, r := range reviews {
		if !eligible[r.Person] {
			return domain.NewError(domain.CodeRoleOverlap, "reviewer not eligible", string(r.Person))
		}
		if r.Conclusion != "approve" {
			continue
		}
		if seen[r.Person] {
			continue
		}
		seen[r.Person] = true
		approved++
	}
	if approved < 2 {
		return domain.NewError(domain.CodeRoleOverlap, "need two distinct approved reviewers")
	}
	return nil
}

// Credential builds a deterministic transfer credential from the task id and a
// monotonic version.
func Credential(task domain.TaskID, version int) string {
	return fmt.Sprintf("MYC-%s-%04d", task, version)
}
