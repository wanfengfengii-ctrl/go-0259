package domain

import (
	"math"
	"testing"
)

func TestFixedAddOverflow(t *testing.T) {
	f := MustFixed(math.MaxInt64, 0)
	o := MustFixed(1, 0)
	if _, err := f.Add(o); err != ErrOverflow {
		t.Fatalf("want ErrOverflow, got %v", err)
	}
}

func TestFixedMulOverflow(t *testing.T) {
	f := MustFixed(math.MaxInt64/2+1, 0)
	o := MustFixed(2, 0)
	if _, err := f.Mul(o); err != ErrOverflow {
		t.Fatalf("want ErrOverflow, got %v", err)
	}
}

func TestFixedDivByZero(t *testing.T) {
	f := MustFixed(10, 0)
	o := MustFixed(0, 0)
	if _, err := f.Div(o); err != ErrDivisionByZero {
		t.Fatalf("want ErrDivisionByZero, got %v", err)
	}
}

func TestFixedDivMinByNegativeOneOverflow(t *testing.T) {
	f := MustFixed(math.MinInt64, 0)
	o := MustFixed(-1, 0)
	if _, err := f.Div(o); err != ErrOverflow {
		t.Fatalf("want ErrOverflow, got %v", err)
	}
}

func TestFixedRescale(t *testing.T) {
	f := MustFixed(12345, 2)
	down, err := f.Rescale(0)
	if err != nil {
		t.Fatal(err)
	}
	if down.Value != 123 || down.Scale != 0 {
		t.Fatalf("down = %+v", down)
	}
	up, err := f.Rescale(4)
	if err != nil {
		t.Fatal(err)
	}
	if up.Value != 1234500 || up.Scale != 4 {
		t.Fatalf("up = %+v", up)
	}
}

func TestFormatFixed(t *testing.T) {
	cases := []struct {
		value int64
		scale int
		want  string
	}{
		{12345, 2, "123.45"},
		{-5, 2, "-0.05"},
		{0, 0, "0"},
		{7, 0, "7"},
		{-7, 0, "-7"},
	}
	for _, c := range cases {
		if got := FormatFixed(c.value, c.scale); got != c.want {
			t.Fatalf("FormatFixed(%d, %d) = %q, want %q", c.value, c.scale, got, c.want)
		}
	}
}

func TestNewFixedRejectsNegativeScale(t *testing.T) {
	if _, err := NewFixed(1, -1); err == nil {
		t.Fatal("want error for negative scale")
	}
}
