package contamination

import (
	"fmt"
)

// WellPlate describes the geometry of a molecular detection plate. Wells are
// addressed as a row letter followed by a column number, e.g. "A1".
type WellPlate struct {
	Rows int
	Cols int
}

// DefaultWellPlate is the standard 8x12 molecular detection plate.
var DefaultWellPlate = WellPlate{Rows: 8, Cols: 12}

// ValidateWell checks that a well identifier is syntactically valid and inside
// the default plate.
func ValidateWell(w string) error {
	if !DefaultWellPlate.Contains(w) {
		return fmt.Errorf("contamination: invalid well %q", w)
	}
	return nil
}

// Contains reports whether the well identifier lies inside the plate.
func (p WellPlate) Contains(w string) bool {
	if w == "" || len(w) < 2 {
		return false
	}
	row := w[0]
	if row < 'A' || row > 'A'+byte(p.Rows-1) {
		return false
	}
	col := 0
	for _, r := range w[1:] {
		if r < '0' || r > '9' {
			return false
		}
		col = col*10 + int(r-'0')
	}
	return col >= 1 && col <= p.Cols
}
