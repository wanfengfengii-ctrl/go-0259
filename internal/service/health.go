package service

// Healthy reports whether the backing store is reachable.
func (s *Service) Healthy() error {
	return s.store.Ping(ctx())
}
