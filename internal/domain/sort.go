package domain

import "sort"

// SortBagPositions returns a deterministically sorted copy of a bag position
// set.
func SortBagPositions(in []BagPosition) []BagPosition {
	out := append([]BagPosition(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// EqualBagPositions reports whether two bag position sets contain the same
// elements regardless of order.
func EqualBagPositions(a, b []BagPosition) bool {
	if len(a) != len(b) {
		return false
	}
	aa := SortBagPositions(a)
	bb := SortBagPositions(b)
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}
