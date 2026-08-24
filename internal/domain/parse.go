package domain

import (
	"fmt"
	"strings"
)

// Parsing sentinel errors. These are distinct from arithmetic overflow and
// division-by-zero so callers can distinguish a malformed input from a value
// that is numerically well-formed but out of range.
var (
	ErrEmptyValue      = fmt.Errorf("domain: empty value")
	ErrValueTooLong    = fmt.Errorf("domain: value too long")
	ErrInvalidSyntax   = fmt.Errorf("domain: invalid decimal syntax")
	ErrScaleMismatch   = fmt.Errorf("domain: scale mismatch")
	ErrValueNegative   = fmt.Errorf("domain: negative value not allowed")
	ErrValueOutOfRange = fmt.Errorf("domain: value out of range")
)

// MaxValueLength bounds the number of significant characters accepted for a
// decimal value, providing the "length" check required by the fixed-integer
// rules.
const MaxValueLength = 32

// ParseFixed parses a decimal string into a Fixed at the given scale. It
// enforces length, syntax, sign, scale and overflow rules. A negative sign is
// accepted syntactically here; sign policy is applied by callers or by
// ParseFixedNonNegative.
func ParseFixed(s string, scale int) (Fixed, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Fixed{}, ErrEmptyValue
	}
	if len(s) > MaxValueLength {
		return Fixed{}, ErrValueTooLong
	}
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}
	if s == "" {
		return Fixed{}, ErrInvalidSyntax
	}
	intPart := s
	fracPart := ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart = s[:i]
		fracPart = s[i+1:]
		if strings.Contains(fracPart, ".") {
			return Fixed{}, ErrInvalidSyntax
		}
	}
	if intPart == "" {
		intPart = "0"
	}
	for _, r := range intPart {
		if r < '0' || r > '9' {
			return Fixed{}, ErrInvalidSyntax
		}
	}
	for _, r := range fracPart {
		if r < '0' || r > '9' {
			return Fixed{}, ErrInvalidSyntax
		}
	}
	if len(fracPart) > scale {
		return Fixed{}, ErrScaleMismatch
	}
	// Combine integer and fractional parts into a scaled integer, checking for
	// overflow at each step.
	digits := intPart + fracPart
	value := int64(0)
	for _, r := range digits {
		value = value*10 + int64(r-'0')
		if value < 0 {
			return Fixed{}, ErrOverflow
		}
	}
	// Pad fractional part out to the requested scale.
	for i := len(fracPart); i < scale; i++ {
		value, _ = mulChecked(value, 10)
		if value < 0 {
			return Fixed{}, ErrOverflow
		}
	}
	if neg {
		if value == 0 {
			// -0 collapses to 0.
			value = 0
		} else {
			value = -value
		}
	}
	return Fixed{Value: value, Scale: scale}, nil
}

// ParseFixedNonNegative parses a decimal string and rejects a negative sign.
func ParseFixedNonNegative(s string, scale int) (Fixed, error) {
	if strings.HasPrefix(strings.TrimSpace(s), "-") {
		return Fixed{}, ErrValueNegative
	}
	return ParseFixed(s, scale)
}
