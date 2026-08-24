package service

import (
	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/maturity"
)

// deriveJudgment computes the derived judgment for an accepted reading against
// the locked thresholds. Moisture and pH are threshold-gated per bag position;
// the remaining environmental metrics only require metric-range validity.
func deriveJudgment(snap catalog.Snapshot, r *maturity.PhysChemReading) maturity.DerivedJudgment {
	if !r.Accepted() {
		return maturity.DerivedNone
	}
	switch r.Metric {
	case domain.MetricMoisture:
		if domain.WithinRange(r.Value, snap.Thresholds.MoistureMin, snap.Thresholds.MoistureMax) {
			return maturity.DerivedPass
		}
		return maturity.DerivedFail
	case domain.MetricPH:
		if domain.WithinRange(r.Value, snap.Thresholds.PHMin, snap.Thresholds.PHMax) {
			return maturity.DerivedPass
		}
		return maturity.DerivedFail
	default:
		return maturity.DerivedPass
	}
}
