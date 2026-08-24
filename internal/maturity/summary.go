package maturity

import "mycocycle-growbag-transfer-gate/internal/domain"

// CoverageSummary is a fixed-point derived summary of mycelium coverage across
// the valid (non-missing) observation cells of a task.
type CoverageSummary struct {
	Count   int          `json:"count"`
	Minimum domain.Fixed `json:"minimum"`
	Maximum domain.Fixed `json:"maximum"`
	Average domain.Fixed `json:"average"`
}

// SummarizeCoverage computes the minimum, maximum and arithmetic mean of the
// mycelium coverage across the valid cells, using fixed-point arithmetic that
// checks for overflow at every step. Missing cells are skipped. It returns an
// empty summary (Count 0) when there are no valid cells.
func SummarizeCoverage(cells []ObservationCell) (CoverageSummary, error) {
	var minV, maxV, sum domain.Fixed
	count := 0
	have := false
	for _, c := range cells {
		if c.Missing {
			continue
		}
		cov := c.MyceliumCoverage
		if !have {
			minV, maxV, sum = cov, cov, cov
			have = true
		} else {
			if cmp, err := cov.Cmp(minV); err != nil {
				return CoverageSummary{}, err
			} else if cmp < 0 {
				minV = cov
			}
			if cmp, err := cov.Cmp(maxV); err != nil {
				return CoverageSummary{}, err
			} else if cmp > 0 {
				maxV = cov
			}
			next, err := sum.Add(cov)
			if err != nil {
				return CoverageSummary{}, err
			}
			sum = next
		}
		count++
	}
	if !have {
		return CoverageSummary{}, nil
	}
	avg, err := sum.Div(domain.Fixed{Value: int64(count), Scale: 0})
	if err != nil {
		return CoverageSummary{}, err
	}
	return CoverageSummary{Count: count, Minimum: minV, Maximum: maxV, Average: avg}, nil
}
