package store

import (
	"context"
	"database/sql"

	"mycocycle-growbag-transfer-gate/internal/contamination"
	"mycocycle-growbag-transfer-gate/internal/domain"
)

// SaveEvidence appends a contamination evidence record.
func (s *SQLite) SaveEvidence(ctx context.Context, e *contamination.Evidence) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO contamination_evidence
			(task_id, recheck_generation, version, position, day_age, well,
			 source, positive, summary)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			e.TaskID, e.RecheckGeneration, e.Version, e.Position, e.DayAge,
			e.Well, e.Source, boolInt(e.Positive), e.Summary)
		return err
	})
}

// LoadEvidence returns all evidence for a task ordered by recheck generation
// and version.
func (s *SQLite) LoadEvidence(ctx context.Context, id domain.TaskID) ([]contamination.Evidence, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT task_id, recheck_generation, version, position, day_age, well,
		       source, positive, summary
		FROM contamination_evidence WHERE task_id = ?
		ORDER BY recheck_generation, version, position, day_age`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contamination.Evidence
	for rows.Next() {
		var e contamination.Evidence
		var positive int
		if err := rows.Scan(&e.TaskID, &e.RecheckGeneration, &e.Version,
			&e.Position, &e.DayAge, &e.Well, &e.Source, &positive, &e.Summary); err != nil {
			return nil, err
		}
		e.Positive = intBool(positive)
		out = append(out, e)
	}
	return out, rows.Err()
}
