package service

import (
	"encoding/json"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
	"mycocycle-growbag-transfer-gate/internal/maturity"
	"mycocycle-growbag-transfer-gate/internal/store"
)

// loadTask fetches a task, mapping a not-found error to a stable domain error.
func (s *Service) loadTask(id domain.TaskID) (*inspection.Task, error) {
	t, err := s.store.LoadTask(ctx(), id)
	if err == store.ErrNotFound {
		return nil, domain.NewError(domain.CodeFinalStateRejected, "task not found", string(id))
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// guard enforces the final-state barrier and generation match for every
// mutating operation.
func (s *Service) guard(t *inspection.Task, gen domain.Generation) error {
	if t.State.IsFinal() {
		return domain.NewError(domain.CodeFinalStateRejected, "task is in a terminal state", string(t.State))
	}
	if t.Generation != gen {
		return domain.NewError(domain.CodeGenerationConflict, "generation conflict",
			string(t.ID))
	}
	return nil
}

// audit appends an audit event.
func (s *Service) audit(taskID domain.TaskID, gen domain.Generation, op domain.OperationID, action, detail string) {
	_ = s.store.SaveAudit(ctx(), &domain.AuditEvent{
		TaskID: taskID, Generation: gen, At: s.now(), Operation: op, Action: action, Detail: detail,
	})
}

// resolveIdempotency checks an operation id. It returns the stored result JSON
// and replay=true when the same content was already processed, or a conflict
// error when the same id was used with different content.
func (s *Service) resolveIdempotency(taskID domain.TaskID, gen domain.Generation, op domain.OperationID, dgst string) (string, bool, error) {
	rec, err := s.store.LoadOperation(ctx(), taskID, gen, op)
	if err == store.ErrNotFound {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if rec.Digest != dgst {
		return "", false, domain.NewError(domain.CodeIdempotencyConflict, "operation content conflict", string(op))
	}
	return rec.ResultJSON, true, nil
}

// recordOperation stores an idempotency record with its result.
func (s *Service) recordOperation(taskID domain.TaskID, gen domain.Generation, op domain.OperationID, dgst string, result any) error {
	b, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return s.store.SaveOperation(ctx(), &inspection.OperationRecord{
		TaskID: taskID, Generation: gen, Operation: op, Digest: dgst,
		ResultJSON: string(b), At: s.now(),
	})
}

// replayResult decodes a stored idempotency result.
func replayResult[T any](resultJSON string, out *T) error {
	return json.Unmarshal([]byte(resultJSON), out)
}

// advanceState validates and persists a state transition.
func (s *Service) advanceState(id domain.TaskID, t *inspection.Task, to inspection.TaskState) error {
	if err := inspection.Advance(t, to); err != nil {
		return err
	}
	return s.store.UpdateTaskState(ctx(), id, to, t.FinalVersion)
}

// isVerifying reports whether the task is in a post-observation verification
// state where rechecks, readings, reviews and finalization are allowed.
func isVerifying(s inspection.TaskState) bool {
	switch s {
	case inspection.StateVerifyingContamination, inspection.StateVerifyingPhysChem, inspection.StatePendingReview:
		return true
	default:
		return false
	}
}

// physChemCollected reports whether every bag position has an accepted moisture
// and pH reading.
func physChemCollected(positions []domain.BagPosition, readings []maturity.PhysChemReading) bool {
	byPos := make(map[domain.BagPosition]map[domain.Metric]bool)
	for _, r := range readings {
		if !r.Accepted() {
			continue
		}
		if byPos[r.Position] == nil {
			byPos[r.Position] = make(map[domain.Metric]bool)
		}
		byPos[r.Position][r.Metric] = true
	}
	for _, p := range positions {
		m := byPos[p]
		if !m[domain.MetricMoisture] || !m[domain.MetricPH] {
			return false
		}
	}
	return true
}
