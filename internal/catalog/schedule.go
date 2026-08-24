package catalog

import (
	"mycocycle-growbag-transfer-gate/internal/domain"
)

// ValidateSchedule checks that a culture schedule template is non-empty, that
// every day age is positive and that the ages are strictly increasing. A
// schedule is the fixed matrix spine against which every bag position must be
// observed.
func ValidateSchedule(s ScheduleTemplate) error {
	if len(s.DayAges) == 0 {
		return newCatalogError(domain.CodeReadingOutOfRange, "empty culture schedule")
	}
	prev := domain.DayAge(0)
	for _, d := range s.DayAges {
		if d <= 0 {
			return newCatalogError(domain.CodeReadingOutOfRange, "non-positive day age")
		}
		if d <= prev {
			return newCatalogError(domain.CodeDuplicateBagPosition, "day ages must be strictly increasing")
		}
		prev = d
	}
	return nil
}

// ContainsDayAge reports whether the schedule covers the given day age.
func (s ScheduleTemplate) ContainsDayAge(d domain.DayAge) bool {
	for _, a := range s.DayAges {
		if a == d {
			return true
		}
	}
	return false
}
