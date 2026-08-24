package store

import (
	"context"
	"database/sql"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/maturity"
)

// SaveReading appends a physico-chemical reading.
func (s *SQLite) SaveReading(ctx context.Context, r *maturity.PhysChemReading) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO physchem_readings
			(task_id, generation, position, day_age, metric, value, scale,
			 source_device, status, derived)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.TaskID, r.Generation, r.Position, r.DayAge, r.Metric,
			r.Value.Value, r.Value.Scale, r.SourceDevice, r.Status, r.Derived)
		return err
	})
}

// LoadReadings returns all physico-chemical readings for a task.
func (s *SQLite) LoadReadings(ctx context.Context, id domain.TaskID) ([]maturity.PhysChemReading, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT task_id, generation, position, day_age, metric, value, scale,
		       source_device, status, derived
		FROM physchem_readings WHERE task_id = ? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []maturity.PhysChemReading
	for rows.Next() {
		var r maturity.PhysChemReading
		if err := rows.Scan(&r.TaskID, &r.Generation, &r.Position, &r.DayAge,
			&r.Metric, &r.Value.Value, &r.Value.Scale, &r.SourceDevice,
			&r.Status, &r.Derived); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SaveAttempt appends a device invocation attempt.
func (s *SQLite) SaveAttempt(ctx context.Context, a *domain.DeviceAttempt) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO device_attempts
			(device_type, device_id, object, task_id, generation, at,
			 script_seq, result, retry_count, error_code)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			a.DeviceType, a.DeviceID, a.Object, a.TaskID, a.Generation, a.At,
			a.ScriptSeq, a.Result, a.RetryCount, a.ErrorCode)
		return err
	})
}

// LoadAttempts returns all device attempts for a task.
func (s *SQLite) LoadAttempts(ctx context.Context, id domain.TaskID) ([]domain.DeviceAttempt, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT device_type, device_id, object, task_id, generation, at,
		       script_seq, result, retry_count, error_code
		FROM device_attempts ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.DeviceAttempt
	for rows.Next() {
		var a domain.DeviceAttempt
		if err := rows.Scan(&a.DeviceType, &a.DeviceID, &a.Object, &a.TaskID,
			&a.Generation, &a.At, &a.ScriptSeq, &a.Result, &a.RetryCount,
			&a.ErrorCode); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
