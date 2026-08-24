package contamination

import "mycocycle-growbag-transfer-gate/internal/domain"

// NextVersion returns the next evidence version for a given task, recheck
// generation, bag position and day age. Evidence is append-only: once a version
// is recorded it can never be overwritten.
func NextVersion(existing []Evidence, task domain.TaskID, gen domain.Generation, pos domain.BagPosition, day domain.DayAge) int {
	max := 0
	for _, e := range existing {
		if e.TaskID != task || e.RecheckGeneration != gen {
			continue
		}
		if e.Position != pos || e.DayAge != day {
			continue
		}
		if e.Version > max {
			max = e.Version
		}
	}
	return max + 1
}

// EvidenceKey identifies a recheck evidence slot.
type EvidenceKey struct {
	Position domain.BagPosition
	DayAge   domain.DayAge
}

// Covered reports whether every affected slot has at least one evidence record
// in the given list.
func Covered(existing []Evidence, affected []EvidenceKey) (missing []EvidenceKey) {
	for _, k := range affected {
		found := false
		for _, e := range existing {
			if e.Position == k.Position && e.DayAge == k.DayAge {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, k)
		}
	}
	return missing
}

// AnyPositive reports whether any evidence record is a positive contamination
// result.
func AnyPositive(existing []Evidence) bool {
	latest := make(map[EvidenceKey]Evidence)
	for _, e := range existing {
		key := EvidenceKey{Position: e.Position, DayAge: e.DayAge}
		prev, ok := latest[key]
		if !ok || e.Version > prev.Version {
			latest[key] = e
		}
	}
	for _, e := range latest {
		if e.Positive {
			return true
		}
	}
	return false
}
