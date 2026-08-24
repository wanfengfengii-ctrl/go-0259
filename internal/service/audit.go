package service

import "mycocycle-growbag-transfer-gate/internal/domain"

// FilterAudit returns the audit trail for a task, optionally filtered to a
// single action name.
func (s *Service) FilterAudit(id domain.TaskID, action string) ([]domain.AuditEvent, error) {
	events, err := s.store.LoadAudit(ctx(), id)
	if err != nil {
		return nil, err
	}
	if action == "" {
		if events == nil {
			events = []domain.AuditEvent{}
		}
		return events, nil
	}
	out := make([]domain.AuditEvent, 0, len(events))
	for _, e := range events {
		if e.Action == action {
			out = append(out, e)
		}
	}
	return out, nil
}
