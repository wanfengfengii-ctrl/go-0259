package service

import (
	"mycocycle-growbag-transfer-gate/internal/domain"
)

// GetTask returns the task read model including any terminal decision.
func (s *Service) GetTask(id domain.TaskID) (TaskView, error) {
	t, err := s.loadTask(id)
	if err != nil {
		return TaskView{}, err
	}
	view := TaskView{
		TaskID:     t.ID,
		BagBatch:   t.BagBatch,
		Generation: t.Generation,
		State:      t.State,
		Summary:    t.Snapshot,
	}
	if d, err := s.store.LoadDecision(ctx(), id); err == nil {
		view.FinalType = string(d.FinalType)
		view.Credential = d.Credential
	}
	return view, nil
}

// GetAudit returns the ordered audit trail for a task.
func (s *Service) GetAudit(id domain.TaskID) ([]domain.AuditEvent, error) {
	events, err := s.store.LoadAudit(ctx(), id)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []domain.AuditEvent{}
	}
	return events, nil
}
