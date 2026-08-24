package service

import (
	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
	"mycocycle-growbag-transfer-gate/internal/store"
)

// Finalize competes to write the single terminal conclusion under the
// single-writer barrier. Only one of transferable, contamination-isolated and
// cancelled can ever be written for a task.
func (s *Service) Finalize(id domain.TaskID, req FinalizeRequest) (FinalizeResult, error) {
	t, err := s.loadTask(id)
	if err != nil {
		return FinalizeResult{}, err
	}
	if err := s.guard(t, req.Generation); err != nil {
		return FinalizeResult{}, err
	}
	if !isVerifying(t.State) {
		return FinalizeResult{}, domain.NewError(domain.CodeFinalStateRejected,
			"task not ready for finalization", string(t.State))
	}

	conclusion, ok := mapConclusion(req.Conclusion)
	if !ok {
		return FinalizeResult{}, domain.NewError(domain.CodeReadingOutOfRange,
			"unknown conclusion", req.Conclusion)
	}

	cells, err := s.store.LoadCells(ctx(), id, req.Generation)
	if err != nil {
		return FinalizeResult{}, err
	}
	readings, err := s.store.LoadReadings(ctx(), id)
	if err != nil {
		return FinalizeResult{}, err
	}
	evidence, err := s.store.LoadEvidence(ctx(), id)
	if err != nil {
		return FinalizeResult{}, err
	}
	reviews, err := s.store.LoadReviews(ctx(), id)
	if err != nil {
		return FinalizeResult{}, err
	}

	eval := arbiter.EvaluateRequest{
		Snapshot:   t.Snapshot,
		Cells:      cells,
		Readings:   readings,
		Evidence:   evidence,
		Reviews:    reviews,
		Conclusion: conclusion,
	}
	finalType, _, derr := arbiter.Evaluate(eval)
	if derr != nil {
		return FinalizeResult{}, derr
	}

	version := t.FinalVersion + 1
	credential := arbiter.Credential(id, version)
	decision := &arbiter.Decision{
		TaskID:           id,
		FinalType:        finalType,
		Credential:       credential,
		WinningOperation: req.Operation,
		WrittenAt:        s.tick(),
		Summary:          string(finalType),
	}
	if err := s.store.FinalizeTask(ctx(), decision); err != nil {
		if err == store.ErrConflict {
			return FinalizeResult{}, domain.NewError(domain.CodeFinalStateRejected,
				"task already finalized")
		}
		return FinalizeResult{}, err
	}

	finalState := stateForFinalType(finalType)
	if err := s.store.UpdateTaskState(ctx(), id, finalState, version); err != nil {
		return FinalizeResult{}, err
	}
	s.audit(id, req.Generation, req.Operation, domain.AuditFinalized, string(finalType))

	return FinalizeResult{
		FinalType:  string(finalType),
		Credential: credential,
		State:      finalState,
	}, nil
}

func mapConclusion(c string) (arbiter.Conclusion, bool) {
	switch c {
	case "transfer":
		return arbiter.ConclusionTransfer, true
	case "isolate":
		return arbiter.ConclusionIsolate, true
	case "cancel":
		return arbiter.ConclusionCancel, true
	default:
		return "", false
	}
}

func stateForFinalType(f arbiter.FinalType) inspection.TaskState {
	switch f {
	case arbiter.FinalTransferable:
		return inspection.StateTransferable
	case arbiter.FinalTransferred:
		return inspection.StateTransferred
	case arbiter.FinalContaminationIsolated:
		return inspection.StateContaminationIsolated
	case arbiter.FinalCancelled:
		return inspection.StateCancelled
	default:
		return inspection.StateCancelled
	}
}
