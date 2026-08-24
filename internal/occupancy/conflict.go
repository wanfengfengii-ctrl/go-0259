package occupancy

import "sort"

// ConflictReasons enumerates the deterministically sorted rejection reasons for
// a candidate lease set against the existing active leases. Reasons are shaped
// as "resource:identifier" so callers can assert an exact, sorted list.
func ConflictReasons(existing []Lease, candidates []Lease) []string {
	reasons := make([]string, 0)
	for _, c := range candidates {
		if c.ResourceType == ResourceProbeWindow {
			for _, e := range existing {
				if e.ResourceType != ResourceProbeWindow {
					continue
				}
				if e.ResourceID != c.ResourceID {
					continue
				}
				if WindowsOverlap(c.WindowStart, c.WindowEnd, e.WindowStart, e.WindowEnd) {
					reasons = append(reasons, "probe_window:"+c.ResourceID)
					break
				}
			}
			continue
		}
		for _, e := range existing {
			if e.ResourceType == c.ResourceType && e.ResourceID == c.ResourceID {
				reasons = append(reasons, string(c.ResourceType)+":"+c.ResourceID)
				break
			}
		}
	}
	sort.Strings(reasons)
	return dedupe(reasons)
}

func dedupe(in []string) []string {
	out := make([]string, 0, len(in))
	for i, s := range in {
		if i > 0 && in[i-1] == s {
			continue
		}
		out = append(out, s)
	}
	return out
}
