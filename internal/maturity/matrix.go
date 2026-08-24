package maturity

import (
	"mycocycle-growbag-transfer-gate/internal/domain"
)

// CellKey identifies a single day-age x bag-position coverage cell.
type CellKey struct {
	DayAge   domain.DayAge
	Position domain.BagPosition
}

// MatrixKeys enumerates the full schedule x position matrix for a task.
func MatrixKeys(schedule []domain.DayAge, positions []domain.BagPosition) []CellKey {
	out := make([]CellKey, 0, len(schedule)*len(positions))
	for _, d := range schedule {
		for _, p := range positions {
			out = append(out, CellKey{DayAge: d, Position: p})
		}
	}
	return out
}

// IndexCells maps existing observation cells by their (dayAge, position) key.
func IndexCells(cells []ObservationCell) map[CellKey]ObservationCell {
	m := make(map[CellKey]ObservationCell, len(cells))
	for _, c := range cells {
		m[CellKey{DayAge: c.DayAge, Position: c.Position}] = c
	}
	return m
}

// MissingCells returns the coverage keys that have no valid (non-missing)
// observation yet. A cell is "valid" when it is present and not marked missing.
func MissingCells(schedule []domain.DayAge, positions []domain.BagPosition, cells []ObservationCell) []CellKey {
	idx := IndexCells(cells)
	var out []CellKey
	for _, k := range MatrixKeys(schedule, positions) {
		c, ok := idx[k]
		if !ok || c.Missing {
			out = append(out, k)
		}
	}
	return out
}

// MatrixComplete reports whether every schedule x position cell has a valid
// observation.
func MatrixComplete(schedule []domain.DayAge, positions []domain.BagPosition, cells []ObservationCell) bool {
	return len(MissingCells(schedule, positions, cells)) == 0
}
