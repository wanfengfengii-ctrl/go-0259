package maturity

import "mycocycle-growbag-transfer-gate/internal/domain"

// DayCoverage is a per-day-age coverage report: how many of the required bag
// positions have a valid observation for that day age.
type DayCoverage struct {
	DayAge   domain.DayAge `json:"day_age"`
	Observed int           `json:"observed"`
	Required int           `json:"required"`
	Missing  int           `json:"missing"`
	Complete bool          `json:"complete"`
}

// ReportDayCoverage computes per-day-age coverage completeness from the locked
// schedule, bag positions and the existing observation cells.
func ReportDayCoverage(schedule []domain.DayAge, positions []domain.BagPosition, cells []ObservationCell) []DayCoverage {
	idx := IndexCells(cells)
	out := make([]DayCoverage, 0, len(schedule))
	for _, d := range schedule {
		dc := DayCoverage{DayAge: d, Required: len(positions)}
		for _, p := range positions {
			c, ok := idx[CellKey{DayAge: d, Position: p}]
			if ok && !c.Missing {
				dc.Observed++
			} else {
				dc.Missing++
			}
		}
		dc.Complete = dc.Missing == 0
		out = append(out, dc)
	}
	return out
}
