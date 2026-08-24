package occupancy

import "mycocycle-growbag-transfer-gate/internal/domain"

// WindowsOverlap reports whether two half-open windows [aStart,aEnd) and
// [bStart,bEnd) intersect. Boundary adjacency (aEnd == bStart or bEnd == aStart)
// does not overlap, so adjacent probe windows may both be granted.
func WindowsOverlap(aStart, aEnd, bStart, bEnd domain.LogicalTime) bool {
	return aStart < bEnd && bStart < aEnd
}

// ProbeOverlap reports whether a candidate probe window overlaps any existing
// lease for the same probe.
func ProbeOverlap(probe domain.ProbeID, start, end domain.LogicalTime, existing []Lease) []Lease {
	var out []Lease
	for _, l := range existing {
		if l.ResourceType != ResourceProbeWindow {
			continue
		}
		if string(domain.ProbeID(l.ResourceID)) != string(probe) {
			continue
		}
		if WindowsOverlap(start, end, l.WindowStart, l.WindowEnd) {
			out = append(out, l)
		}
	}
	return out
}
