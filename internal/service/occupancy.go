package service

import (
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
	"mycocycle-growbag-transfer-gate/internal/occupancy"
)

// OccupancyStart atomically claims the rack and probe window leases before
// culture observation begins.
func (s *Service) OccupancyStart(id domain.TaskID, req OccupancyStartRequest) (OccupancyResult, error) {
	t, err := s.loadTask(id)
	if err != nil {
		return OccupancyResult{}, err
	}
	if err := s.guard(t, req.Generation); err != nil {
		return OccupancyResult{}, err
	}
	dg := digest(req)
	if stored, replay, err := s.resolveIdempotency(id, req.Generation, req.Operation, dg); err != nil {
		return OccupancyResult{}, err
	} else if replay {
		var out OccupancyResult
		if err := replayResult(stored, &out); err != nil {
			return OccupancyResult{}, err
		}
		return out, nil
	}
	if t.State != inspection.StateOccupying {
		return OccupancyResult{}, domain.NewError(domain.CodeFinalStateRejected,
			"task not ready for occupancy", string(t.State))
	}
	if req.ProbeEnd <= req.ProbeStart {
		return OccupancyResult{}, domain.NewError(domain.CodeResourceWindowConflict,
			"probe window end before start")
	}

	leases := []occupancy.Lease{
		{
			ResourceType: occupancy.ResourceRack,
			ResourceID:   string(req.RackID),
			TaskID:       id,
			Generation:   req.Generation,
			State:        occupancy.LeaseActive,
		},
		{
			ResourceType: occupancy.ResourceProbeWindow,
			ResourceID:   string(req.ProbeID),
			WindowStart:  req.ProbeStart,
			WindowEnd:    req.ProbeEnd,
			TaskID:       id,
			Generation:   req.Generation,
			State:        occupancy.LeaseActive,
		},
	}
	if err := s.store.AcquireLeases(ctx(), leases); err != nil {
		reasons := s.occupancyConflictReasons(req.RackID, req.ProbeID, req.ProbeStart, req.ProbeEnd)
		return OccupancyResult{}, domain.NewError(domain.CodeResourceWindowConflict,
			"rack or probe window conflict", reasons...)
	}
	if err := s.advanceState(id, t, inspection.StateObserving); err != nil {
		return OccupancyResult{}, err
	}
	out := OccupancyResult{State: inspection.StateObserving, RackID: req.RackID, Probe: req.ProbeID}
	if err := s.recordOperation(id, req.Generation, req.Operation, dg, out); err != nil {
		return OccupancyResult{}, err
	}
	s.audit(id, req.Generation, req.Operation, domain.AuditOccupancyStarted, string(req.RackID))
	return out, nil
}

// OccupancyMoveRack atomically releases the current rack/probe leases and
// acquires new ones.
func (s *Service) OccupancyMoveRack(id domain.TaskID, req OccupancyMoveRequest) (OccupancyResult, error) {
	t, err := s.loadTask(id)
	if err != nil {
		return OccupancyResult{}, err
	}
	if err := s.guard(t, req.Generation); err != nil {
		return OccupancyResult{}, err
	}
	dg := digest(req)
	if stored, replay, err := s.resolveIdempotency(id, req.Generation, req.Operation, dg); err != nil {
		return OccupancyResult{}, err
	} else if replay {
		var out OccupancyResult
		if err := replayResult(stored, &out); err != nil {
			return OccupancyResult{}, err
		}
		return out, nil
	}
	if t.State != inspection.StateObserving {
		return OccupancyResult{}, domain.NewError(domain.CodeFinalStateRejected,
			"task not observing", string(t.State))
	}
	if req.ProbeEnd <= req.ProbeStart {
		return OccupancyResult{}, domain.NewError(domain.CodeResourceWindowConflict,
			"probe window end before start")
	}

	acquire := []occupancy.Lease{
		{
			ResourceType: occupancy.ResourceRack,
			ResourceID:   string(req.RackID),
			TaskID:       id,
			Generation:   req.Generation,
			State:        occupancy.LeaseActive,
		},
		{
			ResourceType: occupancy.ResourceProbeWindow,
			ResourceID:   string(req.ProbeID),
			WindowStart:  req.ProbeStart,
			WindowEnd:    req.ProbeEnd,
			TaskID:       id,
			Generation:   req.Generation,
			State:        occupancy.LeaseActive,
		},
	}
	release := []occupancy.ResourceType{occupancy.ResourceRack, occupancy.ResourceProbeWindow}
	if err := s.store.TransitionOccupancy(ctx(), id, release, acquire); err != nil {
		reasons := s.occupancyConflictReasons(req.RackID, req.ProbeID, req.ProbeStart, req.ProbeEnd)
		return OccupancyResult{}, domain.NewError(domain.CodeResourceWindowConflict,
			"rack or probe window conflict", reasons...)
	}
	out := OccupancyResult{State: t.State, RackID: req.RackID, Probe: req.ProbeID}
	if err := s.recordOperation(id, req.Generation, req.Operation, dg, out); err != nil {
		return OccupancyResult{}, err
	}
	s.audit(id, req.Generation, req.Operation, domain.AuditOccupancyMoved, string(req.RackID))
	return out, nil
}

func (s *Service) occupancyConflictReasons(rack domain.RackID, probe domain.ProbeID, start, end domain.LogicalTime) []string {
	candidates := []occupancy.Lease{
		{ResourceType: occupancy.ResourceRack, ResourceID: string(rack)},
		{ResourceType: occupancy.ResourceProbeWindow, ResourceID: string(probe), WindowStart: start, WindowEnd: end},
	}
	var existing []occupancy.Lease
	if l, err := s.store.ActiveLease(ctx(), occupancy.ResourceRack, string(rack)); err == nil {
		existing = append(existing, *l)
	}
	if ls, err := s.store.ProbeWindowLeases(ctx(), probe); err == nil {
		existing = append(existing, ls...)
	}
	return occupancy.ConflictReasons(existing, candidates)
}
