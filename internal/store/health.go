package store

import "context"

// Ping verifies that the underlying database is reachable.
func (s *SQLite) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
