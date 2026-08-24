package domain

// Metric scale and range metadata. Every physico-chemical and biological
// measurement uses fixed-point integer arithmetic at a canonical scale so that
// parsing and derived computation can deterministically check range, sign,
// length, division-by-zero and overflow before a value is accepted.
import (
	"fmt"
)

// MetricScales maps each metric to its canonical decimal scale.
var MetricScales = map[Metric]int{
	MetricMoisture:    1, // 0.0 .. 100.0
	MetricPH:          2, // 0.00 .. 14.00
	MetricTemperature: 1, // degrees Celsius, one decimal
	MetricHumidity:    1, // 0.0 .. 100.0 percent
	MetricCO2:         0, // parts per million, integer
}

// CoverageScale is the canonical scale for mycelium coverage percentages.
const CoverageScale = 1

// MetricScale returns the canonical scale for a metric.
func MetricScale(m Metric) (int, bool) {
	s, ok := MetricScales[m]
	return s, ok
}

// MetricAllowNegative reports whether a metric may carry a negative sign.
// Moisture, pH, humidity and CO2 are physically non-negative; temperature may
// be negative in principle but the cultivation domain only accepts non-negative
// values for the probe windows it governs.
func MetricAllowNegative(m Metric) bool {
	return false
}

// MetricRange describes the inclusive fixed-point bounds for a metric.
type MetricRange struct {
	Min int64
	Max int64
}

// MetricRanges returns the inclusive fixed-point bounds for a metric at its
// canonical scale.
func MetricRanges(m Metric) (MetricRange, bool) {
	switch m {
	case MetricMoisture:
		return MetricRange{Min: 0, Max: 1000}, true // 0.0..100.0
	case MetricPH:
		return MetricRange{Min: 0, Max: 1400}, true // 0.00..14.00
	case MetricTemperature:
		return MetricRange{Min: 0, Max: 600}, true // 0.0..60.0 C
	case MetricHumidity:
		return MetricRange{Min: 0, Max: 1000}, true // 0.0..100.0
	case MetricCO2:
		return MetricRange{Min: 0, Max: 100000}, true // 0..100000 ppm
	default:
		return MetricRange{}, false
	}
}

// ValidateMetricValue checks that a fixed value is representable at the
// metric's canonical scale and inside its inclusive range, returning a stable
// domain error otherwise.
func ValidateMetricValue(m Metric, f Fixed) error {
	scale, ok := MetricScale(m)
	if !ok {
		return NewError(CodeReadingOutOfRange, fmt.Sprintf("unknown metric %q", m))
	}
	if f.Scale != scale {
		return NewError(CodeReadingOutOfRange, "scale mismatch",
			fmt.Sprintf("metric %s expects scale %d", m, scale))
	}
	if !MetricAllowNegative(m) && f.Value < 0 {
		return NewError(CodeReadingOutOfRange, "negative value not allowed",
			fmt.Sprintf("metric %s", m))
	}
	r, ok := MetricRanges(m)
	if !ok {
		return nil
	}
	if f.Value < r.Min || f.Value > r.Max {
		return NewError(CodeReadingOutOfRange, "value out of range",
			fmt.Sprintf("metric %s range [%d,%d]", m, r.Min, r.Max))
	}
	return nil
}

// WithinRange reports whether fixed value v lies inside [lo, hi] after
// aligning scales. It is used for threshold comparisons and returns false (not
// an error) for out-of-range values.
func WithinRange(v, lo, hi Fixed) bool {
	if c, err := v.Cmp(lo); err != nil || c < 0 {
		return false
	}
	if c, err := v.Cmp(hi); err != nil || c > 0 {
		return false
	}
	return true
}
