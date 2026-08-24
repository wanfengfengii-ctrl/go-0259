package store

import (
	"context"
	"database/sql"

	"mycocycle-growbag-transfer-gate/internal/domain"
)

// SaveAudit appends an audit event.
func (s *SQLite) SaveAudit(ctx context.Context, e *domain.AuditEvent) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO audit_events (task_id, generation, at, operation, action, detail)
			VALUES (?, ?, ?, ?, ?, ?)`,
			e.TaskID, e.Generation, e.At, e.Operation, e.Action, e.Detail)
		return err
	})
}

// LoadAudit returns all audit events for a task ordered by time.
func (s *SQLite) LoadAudit(ctx context.Context, id domain.TaskID) ([]domain.AuditEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT task_id, generation, at, operation, action, detail
		FROM audit_events WHERE task_id = ? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AuditEvent
	for rows.Next() {
		var e domain.AuditEvent
		if err := rows.Scan(&e.TaskID, &e.Generation, &e.At, &e.Operation, &e.Action, &e.Detail); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
