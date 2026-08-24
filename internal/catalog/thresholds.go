package catalog

import (
	"mycocycle-growbag-transfer-gate/internal/domain"
)

// ValidateThresholds checks that every threshold interval is non-degenerate:
// maturity, moisture and pH lower bounds must not exceed their upper bounds.
func ValidateThresholds(t Thresholds) error {
	if t.MaturityMin.Value > t.MaturityMax.Value || t.MaturityMin.Scale != t.MaturityMax.Scale {
		return newCatalogError(domain.CodeReadingOutOfRange, "invalid maturity threshold interval")
	}
	if t.MoistureMin.Value > t.MoistureMax.Value {
		return newCatalogError(domain.CodeReadingOutOfRange, "invalid moisture threshold interval")
	}
	if t.PHMin.Value > t.PHMax.Value {
		return newCatalogError(domain.CodeReadingOutOfRange, "invalid pH threshold interval")
	}
	if t.Contamination.Value < 0 {
		return newCatalogError(domain.CodeReadingOutOfRange, "negative contamination threshold")
	}
	return nil
}
