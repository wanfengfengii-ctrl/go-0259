package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/occupancy"
)

func TestDeviceFailureRecordsRetryAttempt(t *testing.T) {
	s := newTestService(t)
	id := lockTask(t, s)
	advanceToObserving(t, s, id)

	s.scripts.Set(domain.DeviceProbe, "probe-1", 1, domain.AttemptDisconnected)
	s.scripts.Set(domain.DeviceMolecular, "mol-1", 1, domain.AttemptTimeout)
	s.scripts.Set(domain.DeviceMoisture, "moist-1", 1, domain.AttemptFormatError)

	cases := []struct {
		dt   domain.DeviceType
		dev  domain.DeviceID
		want domain.AttemptResult
	}{
		{domain.DeviceProbe, "probe-1", domain.AttemptDisconnected},
		{domain.DeviceMolecular, "mol-1", domain.AttemptTimeout},
		{domain.DeviceMoisture, "moist-1", domain.AttemptFormatError},
	}
	for _, c := range cases {
		res, err := s.DeviceReading(id, DeviceReadingRequest{
			Operation: "d", Generation: 1, DeviceType: c.dt, DeviceID: c.dev,
			Metric: domain.MetricMoisture, Value: "65.0", Position: "P1", DayAge: 1, ScriptSeq: 1,
		})
		if err != nil {
			t.Fatalf("device reading %s: %v", c.dt, err)
		}
		if res.Result != c.want {
			t.Fatalf("%s result = %s, want %s", c.dt, res.Result, c.want)
		}
		if res.RetryCount != 1 {
			t.Fatalf("%s retry = %d, want 1", c.dt, res.RetryCount)
		}
	}

	attempts, err := s.store.LoadAttempts(ctx(), id)
	if err != nil {
		t.Fatalf("load attempts: %v", err)
	}
	if len(attempts) != 3 {
		t.Fatalf("attempts = %d, want 3", len(attempts))
	}
	readings, _ := s.store.LoadReadings(ctx(), id)
	if len(readings) != 0 {
		t.Fatalf("failed devices produced %d readings, want 0", len(readings))
	}
}

func TestDeviceRetryCountDeterministic(t *testing.T) {
	s := newTestService(t)
	id := lockTask(t, s)
	advanceToObserving(t, s, id)

	s.scripts.Set(domain.DeviceProbe, "probe-1", 1, domain.AttemptDisconnected)
	s.scripts.Set(domain.DeviceProbe, "probe-1", 2, domain.AttemptDisconnected)

	r1, _ := s.DeviceReading(id, DeviceReadingRequest{
		Operation: "d1", Generation: 1, DeviceType: domain.DeviceProbe, DeviceID: "probe-1",
		Metric: domain.MetricTemperature, Value: "25.0", ScriptSeq: 1,
	})
	r2, _ := s.DeviceReading(id, DeviceReadingRequest{
		Operation: "d2", Generation: 1, DeviceType: domain.DeviceProbe, DeviceID: "probe-1",
		Metric: domain.MetricTemperature, Value: "25.0", ScriptSeq: 2,
	})
	if r1.RetryCount != 1 || r2.RetryCount != 2 {
		t.Fatalf("retry counts = %d, %d; want 1, 2", r1.RetryCount, r2.RetryCount)
	}
}

func TestLeaseHeldDuringDeviceFailure(t *testing.T) {
	s := newTestService(t)
	id := lockTask(t, s)
	advanceToObserving(t, s, id)

	s.scripts.Set(domain.DeviceProbe, "probe-1", 1, domain.AttemptDisconnected)
	if _, err := s.DeviceReading(id, DeviceReadingRequest{
		Operation: "d1", Generation: 1, DeviceType: domain.DeviceProbe, DeviceID: "probe-1",
		Metric: domain.MetricTemperature, Value: "25.0", ScriptSeq: 1,
	}); err != nil {
		t.Fatalf("device reading: %v", err)
	}
	if _, err := s.store.ActiveLease(ctx(), occupancy.ResourceRack, "R1"); err != nil {
		t.Fatalf("rack lease should still be active: %v", err)
	}
}
