package store

import (
	"context"
	"database/sql"
	"errors"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

// SaveOperation inserts an idempotency record (unique per operation id).
func (s *SQLite) SaveOperation(ctx context.Context, op *inspection.OperationRecord) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO operations
			(task_id, generation, operation, digest, result_json, at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			op.TaskID, op.Generation, op.Operation, op.Digest, op.ResultJSON, op.At)
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	})
}

// LoadOperation returns the idempotency record for an operation id.
func (s *SQLite) LoadOperation(ctx context.Context, id domain.TaskID, gen domain.Generation, op domain.OperationID) (*inspection.OperationRecord, error) {
	var rec inspection.OperationRecord
	err := s.db.QueryRowContext(ctx, `
		SELECT task_id, generation, operation, digest, result_json, at
		FROM operations WHERE task_id = ? AND generation = ? AND operation = ?`,
		id, gen, op).
		Scan(&rec.TaskID, &rec.Generation, &rec.Operation, &rec.Digest, &rec.ResultJSON, &rec.At)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}
