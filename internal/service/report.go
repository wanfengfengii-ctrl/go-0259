package service

import (
	"mycocycle-growbag-transfer-gate/internal/inspection"
	"mycocycle-growbag-transfer-gate/internal/store"
)

// Report is the operational summary produced for diagnostics: the recovery
// report, per-state task counts, the task listing and pending device retries.
type Report struct {
	Recovery       store.RecoveryReport         `json:"recovery"`
	TasksByState   map[inspection.TaskState]int `json:"tasks_by_state"`
	Tasks          []TaskSummary                `json:"tasks"`
	PendingRetries int                          `json:"pending_device_retries"`
}

// TaskSummary is a compact per-task summary used in the operational report.
type TaskSummary struct {
	TaskID     string               `json:"task_id"`
	BagBatch   string               `json:"bag_batch"`
	Generation int                  `json:"generation"`
	State      inspection.TaskState `json:"state"`
}

// Report assembles the operational summary.
func (s *Service) Report() (Report, error) {
	recovery, err := s.store.Recover(ctx())
	if err != nil {
		return Report{}, err
	}
	byState, err := s.store.CountTasksByState(ctx())
	if err != nil {
		return Report{}, err
	}
	tasks, err := s.store.ListTasks(ctx())
	if err != nil {
		return Report{}, err
	}
	summaries := make([]TaskSummary, 0, len(tasks))
	for _, t := range tasks {
		summaries = append(summaries, TaskSummary{
			TaskID:     string(t.ID),
			BagBatch:   string(t.BagBatch),
			Generation: int(t.Generation),
			State:      t.State,
		})
	}
	return Report{
		Recovery:       recovery,
		TasksByState:   byState,
		Tasks:          summaries,
		PendingRetries: recovery.PendingDeviceRetries,
	}, nil
}
