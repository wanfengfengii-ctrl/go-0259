package store

import (
	"context"
	"database/sql"
	"errors"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/occupancy"
)

func insertLease(tx *sql.Tx, l *occupancy.Lease) error {
	_, err := tx.Exec(`
		INSERT INTO occupancy_leases
		(resource_type, resource_id, position, window_start, window_end,
		 task_id, generation, state, release_reason)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ResourceType, l.ResourceID, l.Position, l.WindowStart, l.WindowEnd,
		l.TaskID, l.Generation, l.State, l.ReleaseReason)
	return err
}

// AcquireLease atomically acquires a lease. Exclusive resources (bag batch,
// bag position, rack) are adjudicated by a unique index; probe windows are
// adjudicated by an overlap check within the same transaction.
func (s *SQLite) AcquireLease(ctx context.Context, l *occupancy.Lease) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		return acquireLeaseTx(tx, l)
	})
}

// ReleaseLease marks a lease released.
func (s *SQLite) ReleaseLease(ctx context.Context, id domain.TaskID, rt occupancy.ResourceType, resourceID string) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			UPDATE occupancy_leases SET state = 'released'
			WHERE task_id = ? AND resource_type = ? AND resource_id = ? AND state = 'active'`,
			id, rt, resourceID)
		return err
	})
}

func scanLease(sc interface{ Scan(...any) error }) (*occupancy.Lease, error) {
	var l occupancy.Lease
	var id int64
	if err := sc.Scan(&id, &l.ResourceType, &l.ResourceID, &l.Position,
		&l.WindowStart, &l.WindowEnd, &l.TaskID, &l.Generation, &l.State, &l.ReleaseReason); err != nil {
		return nil, err
	}
	return &l, nil
}

// LoadTaskLeases returns all leases (active and released) for a task.
func (s *SQLite) LoadTaskLeases(ctx context.Context, id domain.TaskID) ([]occupancy.Lease, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, resource_type, resource_id, position, window_start, window_end,
		       task_id, generation, state, release_reason
		FROM occupancy_leases WHERE task_id = ? AND state = 'active' ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []occupancy.Lease
	for rows.Next() {
		l, err := scanLease(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

// ActiveLease returns the active lease for an exclusive resource, if any.
func (s *SQLite) ActiveLease(ctx context.Context, rt occupancy.ResourceType, resourceID string) (*occupancy.Lease, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, resource_type, resource_id, position, window_start, window_end,
		       task_id, generation, state, release_reason
		FROM occupancy_leases
		WHERE resource_type = ? AND resource_id = ? AND state = 'active'
		ORDER BY id DESC LIMIT 1`, rt, resourceID)
	l, err := scanLease(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return l, nil
}

// ProbeWindowLeases returns active leases for a probe.
func (s *SQLite) ProbeWindowLeases(ctx context.Context, probe domain.ProbeID) ([]occupancy.Lease, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, resource_type, resource_id, position, window_start, window_end,
		       task_id, generation, state, release_reason
		FROM occupancy_leases
		WHERE resource_type = 'probe_window' AND resource_id = ? AND state = 'active'
		ORDER BY window_start`, probe)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []occupancy.Lease
	for rows.Next() {
		l, err := scanLease(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}
