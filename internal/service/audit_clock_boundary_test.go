package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/domain"
)

func TestModel_AuditDoesNotConsumeSterilizerFreshnessBoundary(t *testing.T) {
	const runAUpperBound domain.LogicalTime = 10000

	nonConflicting := defaultLockRequest()
	nonConflicting.BagBatch = "B-002"
	nonConflicting.BagPositions = []domain.BagPosition{"P3", "P4"}
	nonConflicting.RackID = "R2"
	nonConflicting.ProbeWindow = catalog.ProbeWindow{ProbeID: "probe-2", Start: 1, End: 100}

	expired := defaultLockRequest()

	tests := []struct {
		name                string
		startAt             domain.LogicalTime
		preLock             bool
		wantClockAfterLock  domain.LogicalTime
		req                 catalog.LockRequest
		wantSecondErrorCode domain.ErrorCode
	}{
		{
			name:               "nonconflicting second lock is accepted at freshness upper bound",
			startAt:            runAUpperBound - 1,
			preLock:            true,
			wantClockAfterLock: runAUpperBound,
			req:                nonConflicting,
		},
		{
			name:                "explicitly expired sterilizer run is rejected",
			startAt:             runAUpperBound + 1,
			req:                 expired,
			wantSecondErrorCode: domain.CodeStaleSterilizer,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestService(t)
			s.Clock().Set(tc.startAt)

			if tc.preLock {
				if _, err := s.Lock(defaultLockRequest()); err != nil {
					t.Fatalf("first lock: %v", err)
				}
				if got := s.Clock().Now(); got != tc.wantClockAfterLock {
					t.Errorf("clock after audited lock = %d, want %d", got, tc.wantClockAfterLock)
				}
			}

			_, err := s.Lock(tc.req)
			if tc.wantSecondErrorCode != "" {
				assertErrorCode(t, err, tc.wantSecondErrorCode)
				return
			}
			if err != nil {
				t.Fatalf("second lock: %v", err)
			}
		})
	}
}
