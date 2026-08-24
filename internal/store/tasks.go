package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
	"mycocycle-growbag-transfer-gate/internal/occupancy"
)

// isUniqueViolation reports whether err is a SQLite unique-constraint failure.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "UNIQUE constraint failed") ||
		strings.Contains(s, "constraint failed") ||
		strings.Contains(s, "constraint")
}

// CreateTask atomically inserts a task, its locked bag positions and its bag
// batch/position leases. Any lease conflict aborts the whole transaction.
func (s *SQLite) CreateTask(ctx context.Context, t *inspection.Task, positions []occupancy.BagPositionRecord, leases []occupancy.Lease) error {
	snap, err := json.Marshal(t.Snapshot)
	if err != nil {
		return err
	}
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`
			INSERT INTO inspection_tasks
			(id, bag_batch, generation, state, snapshot, created_at, final_version)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			t.ID, t.BagBatch, t.Generation, t.State, string(snap), t.CreatedAt, t.FinalVersion); err != nil {
			if isUniqueViolation(err) {
				return ErrConflict
			}
			return err
		}
		for _, p := range positions {
			if _, err := tx.Exec(`
				INSERT INTO locked_bag_positions
				(task_id, batch, position, sealed, sealed_by, sealed_at)
				VALUES (?, ?, ?, ?, ?, ?)`,
				p.TaskID, p.Batch, p.Position, boolInt(p.Sealed), p.SealedBy, p.SealedAt); err != nil {
				return err
			}
		}
		for _, l := range leases {
			if err := insertLease(tx, &l); err != nil {
				if isUniqueViolation(err) {
					return ErrConflict
				}
				return err
			}
		}
		return nil
	})
}

// LoadTask loads a task aggregate by id.
func (s *SQLite) LoadTask(ctx context.Context, id domain.TaskID) (*inspection.Task, error) {
	var t inspection.Task
	var snap string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, bag_batch, generation, state, snapshot, created_at, final_version
		FROM inspection_tasks WHERE id = ?`, id).
		Scan(&t.ID, &t.BagBatch, &t.Generation, &t.State, &snap, &t.CreatedAt, &t.FinalVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(snap), &t.Snapshot); err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateTaskState updates a task's state and final version within a
// transaction.
func (s *SQLite) UpdateTaskState(ctx context.Context, id domain.TaskID, state inspection.TaskState, finalVersion int) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec(`UPDATE inspection_tasks SET state = ?, final_version = ? WHERE id = ?`,
			state, finalVersion, id)
		return err
	})
}
