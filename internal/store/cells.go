package store

import (
	"context"
	"database/sql"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/maturity"
)

// SaveCell writes a maturity coverage cell. A cell that was previously recorded
// as missing may be closed by a same-generation non-missing makeup observation;
// an already-valid cell can never be overwritten.
func (s *SQLite) SaveCell(ctx context.Context, c *maturity.ObservationCell) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		var existingMissing int
		err := tx.QueryRow(`
			SELECT missing FROM maturity_cells
			WHERE task_id = ? AND generation = ? AND day_age = ? AND position = ?`,
			c.TaskID, c.Generation, c.DayAge, c.Position).Scan(&existingMissing)
		if err == sql.ErrNoRows {
			return insertCell(tx, c)
		}
		if err != nil {
			return err
		}
		if intBool(existingMissing) && !c.Missing {
			return insertCell(tx, c)
		}
		return ErrConflict
	})
}

func insertCell(tx *sql.Tx, c *maturity.ObservationCell) error {
	_, err := tx.Exec(`
		INSERT INTO maturity_cells
		(task_id, generation, day_age, position, mycelium_value, mycelium_scale,
		 contamination_count, bag_damage, missing, summary, observer)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.TaskID, c.Generation, c.DayAge, c.Position,
		c.MyceliumCoverage.Value, c.MyceliumCoverage.Scale,
		c.ContaminationCount, boolInt(c.BagDamage), boolInt(c.Missing),
		c.ObservationSummary, c.Observer)
	return err
}

// LoadCells returns all cells for a task generation.
func (s *SQLite) LoadCells(ctx context.Context, id domain.TaskID, gen domain.Generation) ([]maturity.ObservationCell, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT task_id, generation, day_age, position, mycelium_value, mycelium_scale,
		       contamination_count, bag_damage, missing, summary, observer
		FROM maturity_cells WHERE task_id = ? AND generation = ?
		ORDER BY day_age, position`, id, gen)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []maturity.ObservationCell
	for rows.Next() {
		var c maturity.ObservationCell
		var damage, missing int
		if err := rows.Scan(&c.TaskID, &c.Generation, &c.DayAge, &c.Position,
			&c.MyceliumCoverage.Value, &c.MyceliumCoverage.Scale,
			&c.ContaminationCount, &damage, &missing,
			&c.ObservationSummary, &c.Observer); err != nil {
			return nil, err
		}
		c.BagDamage = intBool(damage)
		c.Missing = intBool(missing)
		out = append(out, c)
	}
	return out, rows.Err()
}
