package domain

import (
	"fmt"
	"math"
)

// Fixed is a fixed-point integer quantity with an explicit decimal scale.
// Every physical and biological measurement in MycoCycle is represented as a
// Fixed so that parsing and derived arithmetic can deterministically check
// range, sign, length, division-by-zero and overflow before a value is ever
// accepted into a valid coverage cell or derived evidence record.
type Fixed struct {
	Value int64 `json:"value"`
	Scale int   `json:"scale"`
}

// NewFixed constructs a Fixed, rejecting a negative scale.
func NewFixed(value int64, scale int) (Fixed, error) {
	if scale < 0 {
		return Fixed{}, fmt.Errorf("fixed: negative scale %d", scale)
	}
	return Fixed{Value: value, Scale: scale}, nil
}

// MustFixed panics on error; intended for tests and literals.
func MustFixed(value int64, scale int) Fixed {
	f, err := NewFixed(value, scale)
	if err != nil {
		panic(err)
	}
	return f
}

// String renders the fixed-point value in decimal notation.
func (f Fixed) String() string {
	return FormatFixed(f.Value, f.Scale)
}

// FormatFixed renders an integer value at the given scale as a decimal string.
func FormatFixed(value int64, scale int) string {
	neg := value < 0
	var abs uint64
	if neg {
		abs = uint64(-(value + 1)) + 1
	} else {
		abs = uint64(value)
	}
	if scale == 0 {
		if neg {
			return fmt.Sprintf("-%d", abs)
		}
		return fmt.Sprintf("%d", abs)
	}
	s := fmt.Sprintf("%0*d", scale+1, abs)
	ip := s[:len(s)-scale]
	fp := s[len(s)-scale:]
	if neg {
		ip = "-" + ip
	}
	return ip + "." + fp
}

// Rescale converts f to target scale, checking for overflow during
// up-scaling and truncating toward zero during down-scaling.
func (f Fixed) Rescale(target int) (Fixed, error) {
	if target < 0 {
		return Fixed{}, fmt.Errorf("fixed: negative target scale %d", target)
	}
	if f.Scale == target {
		return f, nil
	}
	if f.Scale < target {
		v := f.Value
		for i := f.Scale; i < target; i++ {
			var ok bool
			v, ok = mulChecked(v, 10)
			if !ok {
				return Fixed{}, ErrOverflow
			}
		}
		return Fixed{Value: v, Scale: target}, nil
	}
	v := f.Value
	for i := target; i < f.Scale; i++ {
		v /= 10
	}
	return Fixed{Value: v, Scale: target}, nil
}

// Add returns f+o at f's scale, checking for overflow.
func (f Fixed) Add(o Fixed) (Fixed, error) {
	o, err := o.Rescale(f.Scale)
	if err != nil {
		return Fixed{}, err
	}
	v, ok := addChecked(f.Value, o.Value)
	if !ok {
		return Fixed{}, ErrOverflow
	}
	return Fixed{Value: v, Scale: f.Scale}, nil
}

// Sub returns f-o at f's scale, checking for overflow.
func (f Fixed) Sub(o Fixed) (Fixed, error) {
	o, err := o.Rescale(f.Scale)
	if err != nil {
		return Fixed{}, err
	}
	v, ok := subChecked(f.Value, o.Value)
	if !ok {
		return Fixed{}, ErrOverflow
	}
	return Fixed{Value: v, Scale: f.Scale}, nil
}

// Mul returns f*o at scale f.Scale+o.Scale, checking for overflow.
func (f Fixed) Mul(o Fixed) (Fixed, error) {
	v, ok := mulChecked(f.Value, o.Value)
	if !ok {
		return Fixed{}, ErrOverflow
	}
	return Fixed{Value: v, Scale: f.Scale + o.Scale}, nil
}

// Div returns f/o at scale f.Scale-o.Scale, rejecting division by zero and
// overflow.
func (f Fixed) Div(o Fixed) (Fixed, error) {
	if o.Value == 0 {
		return Fixed{}, ErrDivisionByZero
	}
	v, ok := divChecked(f.Value, o.Value)
	if !ok {
		return Fixed{}, ErrOverflow
	}
	return Fixed{Value: v, Scale: f.Scale - o.Scale}, nil
}

func addChecked(a, b int64) (int64, bool) {
	c := a + b
	if (b > 0 && c < a) || (b < 0 && c > a) {
		return 0, false
	}
	return c, true
}

func subChecked(a, b int64) (int64, bool) {
	c := a - b
	if (b > 0 && c > a) || (b < 0 && c < a) {
		return 0, false
	}
	return c, true
}

func mulChecked(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	c := a * b
	if c/b != a {
		return 0, false
	}
	return c, true
}

func divChecked(a, b int64) (int64, bool) {
	if a == math.MinInt64 && b == -1 {
		return 0, false
	}
	return a / b, true
}
