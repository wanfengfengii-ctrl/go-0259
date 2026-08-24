package store

import (
	"context"
	"database/sql"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/occupancy"
)

// AcquireLeases atomically acquires a set of leases; any conflict aborts all.
func (s *SQLite) AcquireLeases(ctx context.Context, leases []occupancy.Lease) error {
	for i := range leases {
		if err := s.AcquireLease(ctx, &leases[i]); err != nil {
			return err
		}
	}
	return nil
}

// TransitionOccupancy atomically releases the given resource kinds for a task
// and acquires replacement leases. It is used by rack moves so that a failed
// move never leaves a task with no rack or a half-updated probe window.
func (s *SQLite) TransitionOccupancy(ctx context.Context, id domain.TaskID, release []occupancy.ResourceType, acquire []occupancy.Lease) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		for _, rt := range release {
			if _, err := tx.Exec(`
				UPDATE occupancy_leases SET state = 'released'
				WHERE task_id = ? AND resource_type = ? AND state = 'active'`, id, rt); err != nil {
				return err
			}
		}
		for i := range acquire {
			if err := acquireLeaseTx(tx, &acquire[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// acquireLeaseTx performs a single lease acquisition inside a transaction,
// mapping unique/overlap conflicts to ErrConflict.
func acquireLeaseTx(tx *sql.Tx, l *occupancy.Lease) error {
	if l.ResourceType == occupancy.ResourceProbeWindow {
		rows, err := tx.Query(`
			SELECT window_start, window_end FROM occupancy_leases
			WHERE resource_type = 'probe_window' AND resource_id = ? AND state = 'active'`,
			l.ResourceID)
		if err != nil {
			return err
		}
		var overlaps []domain.LogicalTime
		for rows.Next() {
			var start, end domain.LogicalTime
			if err := rows.Scan(&start, &end); err != nil {
				rows.Close()
				return err
			}
			if occupancy.WindowsOverlap(l.WindowStart, l.WindowEnd, start, end) {
				overlaps = append(overlaps, start, end)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(overlaps) > 0 {
			return ErrConflict
		}
	}
	if err := insertLease(tx, l); err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}
