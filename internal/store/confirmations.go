package store

import (
	"context"
	"database/sql"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

// SaveConfirmation inserts a sampling confirmation (unique per person).
func (s *SQLite) SaveConfirmation(ctx context.Context, c *inspection.SamplingConfirmation) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO sampling_confirmations
			(task_id, generation, person, batch, positions_digest, operation, at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			c.TaskID, c.Generation, c.Person, c.Batch, c.PositionsDigest, c.Operation, c.At)
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	})
}

// LoadConfirmations returns the confirmations for a task.
func (s *SQLite) LoadConfirmations(ctx context.Context, id domain.TaskID) ([]inspection.SamplingConfirmation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT task_id, generation, person, batch, positions_digest, operation, at
		FROM sampling_confirmations WHERE task_id = ? ORDER BY at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []inspection.SamplingConfirmation
	for rows.Next() {
		var c inspection.SamplingConfirmation
		if err := rows.Scan(&c.TaskID, &c.Generation, &c.Person, &c.Batch,
			&c.PositionsDigest, &c.Operation, &c.At); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
