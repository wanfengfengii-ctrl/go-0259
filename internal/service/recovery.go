package service

import (
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/store"
)

// Recover returns the persisted-state recovery report from the store. It is
// called at startup so operators can observe what was rebuilt after a restart.
func (s *Service) Recover() (store.RecoveryReport, error) {
	return s.store.Recover(ctx())
}

// RetryPendingDevices returns the unclosed device invocations that recovery
// turned back into retryable work. Each entry carries the retry object, the
// deterministic retry count and the logical time of the last attempt.
func (s *Service) RetryPendingDevices() ([]domain.DeviceAttempt, error) {
	return s.store.PendingDeviceRetries(ctx())
}
