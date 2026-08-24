package service

import (
	"fmt"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
	"mycocycle-growbag-transfer-gate/internal/maturity"
)

// Observation records a single day-age x bag-position coverage cell.
func (s *Service) Observation(id domain.TaskID, req ObservationRequest) (ObservationResult, error) {
	t, err := s.loadTask(id)
	if err != nil {
		return ObservationResult{}, err
	}
	if err := s.guard(t, req.Generation); err != nil {
		return ObservationResult{}, err
	}
	dg := digest(req)
	if stored, replay, err := s.resolveIdempotency(id, req.Generation, req.Operation, dg); err != nil {
		return ObservationResult{}, err
	} else if replay {
		var out ObservationResult
		if err := replayResult(stored, &out); err != nil {
			return ObservationResult{}, err
		}
		return out, nil
	}
	if t.State != inspection.StateObserving {
		return ObservationResult{}, domain.NewError(domain.CodeFinalStateRejected,
			"task not observing", string(t.State))
	}
	if !t.Snapshot.Schedule.ContainsDayAge(req.DayAge) {
		return ObservationResult{}, domain.NewError(domain.CodeReadingOutOfRange,
			"day age outside locked schedule", fmt.Sprintf("%d", req.DayAge))
	}
	if !positionIn(req.Position, t.Snapshot.BagPositions) {
		return ObservationResult{}, domain.NewError(domain.CodeReadingOutOfRange,
			"bag position outside locked set", string(req.Position))
	}
	if req.ContaminationCount < 0 {
		return ObservationResult{}, domain.NewError(domain.CodeReadingOutOfRange,
			"negative contamination count")
	}
	rv, ok := s.catalog.Reviewer(req.Observer)
	if !ok || !rv.IsQualified() {
		return ObservationResult{}, domain.NewError(domain.CodeRoleOverlap,
			"unqualified observer", string(req.Observer))
	}

	cell := &maturity.ObservationCell{
		TaskID:             id,
		Generation:         req.Generation,
		DayAge:             req.DayAge,
		Position:           req.Position,
		ContaminationCount: req.ContaminationCount,
		BagDamage:          req.BagDamage,
		Missing:            req.Missing,
		ObservationSummary: req.Summary,
		Observer:           req.Observer,
	}

	if !req.Missing {
		cov, err := domain.ParseFixedNonNegative(req.MyceliumCoverage, domain.CoverageScale)
		if err != nil {
			return ObservationResult{}, domain.NewError(domain.CodeReadingOutOfRange,
				"invalid mycelium coverage", err.Error())
		}
		if cov.Value > 1000 {
			return ObservationResult{}, domain.NewError(domain.CodeReadingOutOfRange,
				"mycelium coverage out of range")
		}
		cell.MyceliumCoverage = cov
	}

	if err := s.store.SaveCell(ctx(), cell); err != nil {
		return ObservationResult{}, domain.NewError(domain.CodeIdempotencyConflict,
			"cell already validly written", string(req.Position))
	}

	cells, err := s.store.LoadCells(ctx(), id, req.Generation)
	if err != nil {
		return ObservationResult{}, err
	}
	missing := maturity.MissingCells(t.Snapshot.Schedule.DayAges, t.Snapshot.BagPositions, cells)
	state := t.State
	if len(missing) == 0 {
		if err := s.advanceState(id, t, inspection.StateVerifyingContamination); err != nil {
			return ObservationResult{}, err
		}
		state = inspection.StateVerifyingContamination
	}
	out := ObservationResult{State: state, Missing: len(missing)}
	if err := s.recordOperation(id, req.Generation, req.Operation, dg, out); err != nil {
		return ObservationResult{}, err
	}
	s.audit(id, req.Generation, req.Operation, domain.AuditObserved, string(req.Position))
	return out, nil
}

func positionIn(p domain.BagPosition, set []domain.BagPosition) bool {
	for _, x := range set {
		if x == p {
			return true
		}
	}
	return false
}
