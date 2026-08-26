package service

import (
	"math"
	"strings"
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/maturity"
)

func TestParseAllMetrics(t *testing.T) {
	cases := []struct {
		metric domain.Metric
		value  string
		scale  int
		want   int64
	}{
		{domain.MetricMoisture, "65.0", 1, 650},
		{domain.MetricPH, "6.00", 2, 600},
		{domain.MetricTemperature, "25.5", 1, 255},
		{domain.MetricHumidity, "60.0", 1, 600},
		{domain.MetricCO2, "1200", 0, 1200},
	}
	for _, c := range cases {
		f, err := domain.ParseFixed(c.value, c.scale)
		if err != nil {
			t.Fatalf("parse %s %q: %v", c.metric, c.value, err)
		}
		if f.Value != c.want || f.Scale != c.scale {
			t.Fatalf("parse %s %q = %+v, want %d@%d", c.metric, c.value, f, c.want, c.scale)
		}
	}
}

func TestParseTooLongRejected(t *testing.T) {
	_, err := domain.ParseFixed(strings.Repeat("1", 40), 1)
	if err != domain.ErrValueTooLong {
		t.Fatalf("want ErrValueTooLong, got %v", err)
	}
}

func TestParseNegativeRejected(t *testing.T) {
	if _, err := domain.ParseFixedNonNegative("-5.0", 1); err != domain.ErrValueNegative {
		t.Fatalf("want ErrValueNegative, got %v", err)
	}
}

func TestParseScaleMismatchRejected(t *testing.T) {
	if _, err := domain.ParseFixed("6.123", 2); err != domain.ErrScaleMismatch {
		t.Fatalf("want ErrScaleMismatch, got %v", err)
	}
}

func TestDivisionByZeroNoDerived(t *testing.T) {
	f := domain.MustFixed(10, 0)
	o := domain.MustFixed(0, 0)
	if _, err := f.Div(o); err != domain.ErrDivisionByZero {
		t.Fatalf("want ErrDivisionByZero, got %v", err)
	}
}

func TestOverflowNoDerived(t *testing.T) {
	f := domain.MustFixed(math.MaxInt64, 0)
	o := domain.MustFixed(1, 0)
	if _, err := f.Add(o); err != domain.ErrOverflow {
		t.Fatalf("want ErrOverflow, got %v", err)
	}
}

func TestInvalidReadingNoDerivedEvidence(t *testing.T) {
	s := newTestService(t)
	id := lockTask(t, s)
	advanceToObserving(t, s, id)

	if _, err := s.DeviceReading(id, DeviceReadingRequest{
		Operation: "d1", Generation: 1, DeviceType: domain.DeviceProbe,
		DeviceID: "probe-1", Metric: domain.MetricMoisture,
		Value: strings.Repeat("9", 40), Position: "P1", DayAge: 1,
	}); err != nil {
		t.Fatalf("device reading: %v", err)
	}
	readings, err := s.store.LoadReadings(ctx(), id)
	if err != nil {
		t.Fatalf("load readings: %v", err)
	}
	if len(readings) != 1 {
		t.Fatalf("readings = %d, want 1", len(readings))
	}
	if readings[0].Accepted() {
		t.Fatal("too-long value should not be accepted")
	}
	if readings[0].Derived != maturity.DerivedNone {
		t.Fatalf("derived = %s, want none", readings[0].Derived)
	}
}
