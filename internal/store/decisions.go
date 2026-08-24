package store

import (
	"context"
	"database/sql"
	"errors"

	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/domain"
)

// FinalizeTask writes a terminal decision under the single-writer barrier. The
// task_id primary key ensures only one decision can ever be written.
func (s *SQLite) FinalizeTask(ctx context.Context, d *arbiter.Decision) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO final_decisions
			(task_id, final_type, credential, winning_operation, written_at, summary)
			VALUES (?, ?, ?, ?, ?, ?)`,
			d.TaskID, d.FinalType, d.Credential, d.WinningOperation, d.WrittenAt, d.Summary)
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	})
}

// LoadDecision returns the terminal decision for a task, if any.
func (s *SQLite) LoadDecision(ctx context.Context, id domain.TaskID) (*arbiter.Decision, error) {
	var d arbiter.Decision
	err := s.db.QueryRowContext(ctx, `
		SELECT task_id, final_type, credential, winning_operation, written_at, summary
		FROM final_decisions WHERE task_id = ?`, id).
		Scan(&d.TaskID, &d.FinalType, &d.Credential, &d.WinningOperation, &d.WrittenAt, &d.Summary)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}
