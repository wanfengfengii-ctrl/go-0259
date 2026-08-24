package store

import (
	"context"
	"encoding/json"

	"mycocycle-growbag-transfer-gate/internal/domain"

	"mycocycle-growbag-transfer-gate/internal/inspection"
)

// ListTasks returns every task ordered by creation time.
func (s *SQLite) ListTasks(ctx context.Context) ([]inspection.Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, bag_batch, generation, state, snapshot, created_at, final_version
		FROM inspection_tasks ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []inspection.Task
	for rows.Next() {
		var t inspection.Task
		var snap string
		if err := rows.Scan(&t.ID, &t.BagBatch, &t.Generation, &t.State, &snap,
			&t.CreatedAt, &t.FinalVersion); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(snap), &t.Snapshot); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if out == nil {
		out = []inspection.Task{}
	}
	return out, rows.Err()
}

// CountTasksByState returns the number of tasks in each lifecycle state.
func (s *SQLite) CountTasksByState(ctx context.Context) (map[inspection.TaskState]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT state, COUNT(*) FROM inspection_tasks GROUP BY state`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[inspection.TaskState]int)
	for rows.Next() {
		var state inspection.TaskState
		var n int
		if err := rows.Scan(&state, &n); err != nil {
			return nil, err
		}
		out[state] = n
	}
	return out, rows.Err()
}

// LoadTasksByBatch returns every task for a bag batch ordered by creation time.
func (s *SQLite) LoadTasksByBatch(ctx context.Context, batch domain.BagBatch) ([]inspection.Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, bag_batch, generation, state, snapshot, created_at, final_version
		FROM inspection_tasks WHERE bag_batch = ? ORDER BY created_at, id`, batch)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []inspection.Task
	for rows.Next() {
		var t inspection.Task
		var snap string
		if err := rows.Scan(&t.ID, &t.BagBatch, &t.Generation, &t.State, &snap,
			&t.CreatedAt, &t.FinalVersion); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(snap), &t.Snapshot); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if out == nil {
		out = []inspection.Task{}
	}
	return out, rows.Err()
}
