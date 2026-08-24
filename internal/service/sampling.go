package service

import (
	"sort"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
	"mycocycle-growbag-transfer-gate/internal/occupancy"
)

// SamplingConfirm records one person's sampling confirmation. Two distinct
// qualified persons must confirm before sealing may begin.
func (s *Service) SamplingConfirm(id domain.TaskID, req SamplingConfirmRequest) (SamplingConfirmResult, error) {
	t, err := s.loadTask(id)
	if err != nil {
		return SamplingConfirmResult{}, err
	}
	if err := s.guard(t, req.Generation); err != nil {
		return SamplingConfirmResult{}, err
	}
	dg := digest(req)
	if stored, replay, err := s.resolveIdempotency(id, req.Generation, req.Operation, dg); err != nil {
		return SamplingConfirmResult{}, err
	} else if replay {
		var out SamplingConfirmResult
		if err := replayResult(stored, &out); err != nil {
			return SamplingConfirmResult{}, err
		}
		return out, nil
	}

	if !sameBagPositions(req.Positions, t.Snapshot.BagPositions) {
		return SamplingConfirmResult{}, domain.NewError(domain.CodeSubstrateMismatch,
			"bag positions do not match locked snapshot")
	}
	if req.Batch != t.BagBatch {
		return SamplingConfirmResult{}, domain.NewError(domain.CodeSubstrateMismatch,
			"bag batch does not match task")
	}

	rv, ok := s.catalog.Reviewer(req.Person)
	if !ok || !rv.IsQualified() {
		return SamplingConfirmResult{}, domain.NewError(domain.CodeRoleOverlap,
			"unqualified sampler", string(req.Person))
	}

	existing, err := s.store.LoadConfirmations(ctx(), id)
	if err != nil {
		return SamplingConfirmResult{}, err
	}
	for _, c := range existing {
		if c.Person == req.Person {
			return SamplingConfirmResult{}, domain.NewError(domain.CodeRoleOverlap,
				"duplicate sampling person", string(req.Person))
		}
	}

	conf := &inspection.SamplingConfirmation{
		TaskID: id, Generation: req.Generation, Person: req.Person,
		Batch: req.Batch, PositionsDigest: positionsDigest(req.Positions),
		Operation: req.Operation, At: s.tick(),
	}
	if err := s.store.SaveConfirmation(ctx(), conf); err != nil {
		return SamplingConfirmResult{}, err
	}

	state := t.State
	if len(existing)+1 >= 2 {
		if err := s.advanceState(id, t, inspection.StateSealingSamples); err != nil {
			return SamplingConfirmResult{}, err
		}
		state = inspection.StateSealingSamples
	}

	out := SamplingConfirmResult{
		State: state, Confirmations: len(existing) + 1,
	}
	out.ConfirmedBy = append(out.ConfirmedBy, req.Person)
	for _, c := range existing {
		out.ConfirmedBy = append(out.ConfirmedBy, c.Person)
	}
	sort.Slice(out.ConfirmedBy, func(i, j int) bool { return out.ConfirmedBy[i] < out.ConfirmedBy[j] })

	if err := s.recordOperation(id, req.Generation, req.Operation, dg, out); err != nil {
		return SamplingConfirmResult{}, err
	}
	s.audit(id, req.Generation, req.Operation, domain.AuditSamplingConfirmed, string(req.Person))
	return out, nil
}

// SampleSeal completes bag-position sample sealing for the given positions.
func (s *Service) SampleSeal(id domain.TaskID, req SampleSealRequest) (SampleSealResult, error) {
	t, err := s.loadTask(id)
	if err != nil {
		return SampleSealResult{}, err
	}
	if err := s.guard(t, req.Generation); err != nil {
		return SampleSealResult{}, err
	}
	dg := digest(req)
	if stored, replay, err := s.resolveIdempotency(id, req.Generation, req.Operation, dg); err != nil {
		return SampleSealResult{}, err
	} else if replay {
		var out SampleSealResult
		if err := replayResult(stored, &out); err != nil {
			return SampleSealResult{}, err
		}
		return out, nil
	}
	if t.State != inspection.StateSealingSamples {
		return SampleSealResult{}, domain.NewError(domain.CodeFinalStateRejected,
			"task not in sealing state", string(t.State))
	}
	rv, ok := s.catalog.Reviewer(req.Person)
	if !ok || !rv.IsQualified() {
		return SampleSealResult{}, domain.NewError(domain.CodeRoleOverlap,
			"unqualified sealer", string(req.Person))
	}

	positions, err := s.store.LoadBagPositions(ctx(), id)
	if err != nil {
		return SampleSealResult{}, err
	}
	sealSet := make(map[domain.BagPosition]bool, len(req.Positions))
	for _, p := range req.Positions {
		sealSet[p] = true
	}
	at := s.tick()
	updates := make([]occupancy.BagPositionRecord, 0, len(req.Positions))
	sealed := make([]SealedPosition, 0, len(req.Positions))
	for _, p := range positions {
		if sealSet[p.Position] {
			updates = append(updates, occupancy.BagPositionRecord{
				TaskID: id, Batch: p.Batch, Position: p.Position,
				Sealed: true, SealedBy: req.Person, SealedAt: at,
			})
			sealed = append(sealed, SealedPosition{Position: p.Position, SealedBy: req.Person})
		}
	}
	if len(updates) == 0 {
		return SampleSealResult{}, domain.NewError(domain.CodeDuplicateBagPosition,
			"no matching bag positions to seal")
	}
	if err := s.store.SealBagPositions(ctx(), id, updates); err != nil {
		return SampleSealResult{}, err
	}

	// After sealing, all locked positions must be sealed to advance.
	all, err := s.store.LoadBagPositions(ctx(), id)
	if err != nil {
		return SampleSealResult{}, err
	}
	allSealed := true
	for _, p := range all {
		if !p.Sealed {
			allSealed = false
			break
		}
	}
	state := t.State
	if allSealed {
		if err := s.advanceState(id, t, inspection.StateOccupying); err != nil {
			return SampleSealResult{}, err
		}
		state = inspection.StateOccupying
	}

	sort.Slice(sealed, func(i, j int) bool { return sealed[i].Position < sealed[j].Position })
	out := SampleSealResult{State: state, Sealed: sealed}
	if err := s.recordOperation(id, req.Generation, req.Operation, dg, out); err != nil {
		return SampleSealResult{}, err
	}
	s.audit(id, req.Generation, req.Operation, domain.AuditSampleSealed, string(req.Person))
	return out, nil
}

func sameBagPositions(a, b []domain.BagPosition) bool {
	return domain.EqualBagPositions(a, b)
}

func positionsDigest(positions []domain.BagPosition) string {
	return digest(domain.SortBagPositions(positions))
}
