package service

import (
	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/contamination"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

// ContaminationRecheck records a contamination recheck evidence record. Evidence
// is versioned and append-only; an old recheck generation is audited only and
// never participates in the current conclusion.
func (s *Service) ContaminationRecheck(id domain.TaskID, req ContaminationRecheckRequest) (ContaminationRecheckResult, error) {
	t, err := s.loadTask(id)
	if err != nil {
		return ContaminationRecheckResult{}, err
	}
	if t.State.IsFinal() {
		return ContaminationRecheckResult{}, domain.NewError(domain.CodeFinalStateRejected,
			"task is in a terminal state", string(t.State))
	}

	// Old recheck generation: audit only, no evidence.
	if req.RecheckGeneration < t.Generation {
		s.audit(id, req.Generation, req.Operation, domain.AuditLateReading, "late recheck generation")
		return ContaminationRecheckResult{}, nil
	}
	if req.RecheckGeneration > t.Generation {
		return ContaminationRecheckResult{}, domain.NewError(domain.CodeGenerationConflict,
			"recheck generation conflict")
	}
	if !isVerifying(t.State) {
		return ContaminationRecheckResult{}, domain.NewError(domain.CodeFinalStateRejected,
			"task not ready for recheck", string(t.State))
	}
	if !positionIn(req.Position, t.Snapshot.BagPositions) {
		return ContaminationRecheckResult{}, domain.NewError(domain.CodeReadingOutOfRange,
			"bag position outside locked set", string(req.Position))
	}
	if !t.Snapshot.Schedule.ContainsDayAge(req.DayAge) {
		return ContaminationRecheckResult{}, domain.NewError(domain.CodeReadingOutOfRange,
			"day age outside locked schedule")
	}
	if err := contamination.ValidateWell(req.Well); err != nil {
		return ContaminationRecheckResult{}, domain.NewError(domain.CodeRecheckInsufficient,
			"invalid detection well", req.Well)
	}

	existing, err := s.store.LoadEvidence(ctx(), id)
	if err != nil {
		return ContaminationRecheckResult{}, err
	}
	version := contamination.NextVersion(existing, id, req.RecheckGeneration, req.Position, req.DayAge)

	evidence := &contamination.Evidence{
		TaskID:            id,
		RecheckGeneration: req.RecheckGeneration,
		Version:           version,
		Position:          req.Position,
		DayAge:            req.DayAge,
		Well:              req.Well,
		Source:            req.Source,
		Positive:          req.Positive,
		Summary:           req.Summary,
	}
	// Advance from contamination verification to physchem verification once the
	// recheck coverage is closed.
	if t.State == inspection.StateVerifyingContamination {
		if s.recheckClosed(id, t) {
			_ = s.advanceState(id, t, inspection.StateVerifyingPhysChem)
		}
	}
	if err := s.store.SaveEvidence(ctx(), evidence); err != nil {
		return ContaminationRecheckResult{}, err
	}
	s.audit(id, req.Generation, req.Operation, domain.AuditEvidenceRecorded, req.Well)

	return ContaminationRecheckResult{Version: version}, nil
}

// recheckClosed reports whether every contamination signal has a covering
// evidence record.
func (s *Service) recheckClosed(id domain.TaskID, t *inspection.Task) bool {
	cells, err := s.store.LoadCells(ctx(), id, t.Generation)
	if err != nil {
		return false
	}
	evidence, err := s.store.LoadEvidence(ctx(), id)
	if err != nil {
		return false
	}
	signals := arbiter.DetectSignals(t.Snapshot, cells)
	affected := make([]contamination.EvidenceKey, 0, len(signals))
	for _, sig := range signals {
		affected = append(affected, contamination.EvidenceKey{Position: sig.Position, DayAge: sig.DayAge})
	}
	return len(contamination.Covered(evidence, affected)) == 0
}
