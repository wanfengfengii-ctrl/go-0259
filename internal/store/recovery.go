package store

import (
	"context"

	"mycocycle-growbag-transfer-gate/internal/domain"
)

// RecoveryReport summarizes the persisted state recovered from the database at
// startup. It is the deterministic answer to "what is the current state of the
// world" after a restart.
type RecoveryReport struct {
	Tasks                int `json:"tasks"`
	ActiveLeases         int `json:"active_leases"`
	LockedBagPositions   int `json:"locked_bag_positions"`
	Cells                int `json:"cells"`
	Readings             int `json:"readings"`
	DeviceAttempts       int `json:"device_attempts"`
	Evidence             int `json:"evidence"`
	Reviews              int `json:"reviews"`
	Decisions            int `json:"decisions"`
	PendingDeviceRetries int `json:"pending_device_retries"`
}

// Recover rebuilds a recovery report from the database. It reads every table
// that participates in restart recovery and counts the live records.
func (s *SQLite) Recover(ctx context.Context) (RecoveryReport, error) {
	var r RecoveryReport
	counts := []struct {
		query string
		dst   *int
	}{
		{`SELECT COUNT(*) FROM inspection_tasks`, &r.Tasks},
		{`SELECT COUNT(*) FROM occupancy_leases WHERE state = 'active'`, &r.ActiveLeases},
		{`SELECT COUNT(*) FROM locked_bag_positions`, &r.LockedBagPositions},
		{`SELECT COUNT(*) FROM maturity_cells`, &r.Cells},
		{`SELECT COUNT(*) FROM physchem_readings`, &r.Readings},
		{`SELECT COUNT(*) FROM device_attempts`, &r.DeviceAttempts},
		{`SELECT COUNT(*) FROM contamination_evidence`, &r.Evidence},
		{`SELECT COUNT(*) FROM reviews`, &r.Reviews},
		{`SELECT COUNT(*) FROM final_decisions`, &r.Decisions},
	}
	for _, c := range counts {
		if err := s.db.QueryRowContext(ctx, c.query).Scan(c.dst); err != nil {
			return RecoveryReport{}, err
		}
	}
	pending, err := s.PendingDeviceRetries(ctx)
	if err != nil {
		return RecoveryReport{}, err
	}
	r.PendingDeviceRetries = len(pending)
	return r, nil
}

// PendingDeviceRetries returns the failed device invocations that have no later
// accepted result for the same task and device. These are the "unclosed" device
// calls that recovery turns back into retryable work.
func (s *SQLite) PendingDeviceRetries(ctx context.Context) ([]domain.DeviceAttempt, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT device_type, device_id, object, task_id, generation, at,
		       script_seq, result, retry_count, error_code
		FROM device_attempts a
		WHERE a.result != 'accepted'
		  AND a.id = (SELECT MAX(b.id) FROM device_attempts b
		              WHERE b.task_id = a.task_id AND b.device_type = a.device_type
		                AND b.device_id = a.device_id)
		ORDER BY a.task_id, a.device_type, a.device_id`)
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
	if out == nil {
		out = []domain.DeviceAttempt{}
	}
	return out, rows.Err()
}
