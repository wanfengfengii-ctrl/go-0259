package store

import (
	"context"
	"database/sql"

	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/domain"
)

// SaveReview inserts an independent review (unique per person).
func (s *SQLite) SaveReview(ctx context.Context, r *arbiter.Review) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT OR REPLACE INTO reviews
			(task_id, generation, person, qualification, conclusion, digest, operation, at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			r.TaskID, r.Generation, r.Person, r.Qualification, r.Conclusion,
			r.Digest, r.Operation, r.At)
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	})
}

// LoadReviews returns all reviews for a task.
func (s *SQLite) LoadReviews(ctx context.Context, id domain.TaskID) ([]arbiter.Review, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT task_id, generation, person, qualification, conclusion, digest, operation, at
		FROM reviews WHERE task_id = ? ORDER BY at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []arbiter.Review
	for rows.Next() {
		var r arbiter.Review
		if err := rows.Scan(&r.TaskID, &r.Generation, &r.Person, &r.Qualification,
			&r.Conclusion, &r.Digest, &r.Operation, &r.At); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
