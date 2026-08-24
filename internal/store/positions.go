package store

import (
	"context"
	"database/sql"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/occupancy"
)

// LoadBagPositions returns the locked bag positions for a task, sorted by
// position.
func (s *SQLite) LoadBagPositions(ctx context.Context, id domain.TaskID) ([]occupancy.BagPositionRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT task_id, batch, position, sealed, sealed_by, sealed_at
		FROM locked_bag_positions WHERE task_id = ? ORDER BY position`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []occupancy.BagPositionRecord
	for rows.Next() {
		var p occupancy.BagPositionRecord
		var sealed int
		if err := rows.Scan(&p.TaskID, &p.Batch, &p.Position, &sealed, &p.SealedBy, &p.SealedAt); err != nil {
			return nil, err
		}
		p.Sealed = intBool(sealed)
		out = append(out, p)
	}
	return out, rows.Err()
}

// SealBagPositions atomically updates the seal status for the given positions.
func (s *SQLite) SealBagPositions(ctx context.Context, id domain.TaskID, positions []occupancy.BagPositionRecord) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		for _, p := range positions {
			if _, err := tx.Exec(`
				UPDATE locked_bag_positions SET sealed = ?, sealed_by = ?, sealed_at = ?
				WHERE task_id = ? AND position = ?`,
				boolInt(p.Sealed), p.SealedBy, p.SealedAt, id, p.Position); err != nil {
				return err
			}
		}
		return nil
	})
}
