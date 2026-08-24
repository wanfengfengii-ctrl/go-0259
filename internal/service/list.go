package service

import "mycocycle-growbag-transfer-gate/internal/domain"

// ListTaskSummaries returns a compact summary of every task, ordered by
// creation time.
func (s *Service) ListTaskSummaries() ([]TaskSummary, error) {
	tasks, err := s.store.ListTasks(ctx())
	if err != nil {
		return nil, err
	}
	out := make([]TaskSummary, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, TaskSummary{
			TaskID:     string(t.ID),
			BagBatch:   string(t.BagBatch),
			Generation: int(t.Generation),
			State:      t.State,
		})
	}
	return out, nil
}

// TasksByBatch returns the compact summaries of every task in a bag batch.
func (s *Service) TasksByBatch(batch string) ([]TaskSummary, error) {
	tasks, err := s.store.LoadTasksByBatch(ctx(), domain.BagBatch(batch))
	if err != nil {
		return nil, err
	}
	out := make([]TaskSummary, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, TaskSummary{
			TaskID:     string(t.ID),
			BagBatch:   string(t.BagBatch),
			Generation: int(t.Generation),
			State:      t.State,
		})
	}
	return out, nil
}
