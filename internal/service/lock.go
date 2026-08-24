package service

import (
	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
	"mycocycle-growbag-transfer-gate/internal/occupancy"
	"mycocycle-growbag-transfer-gate/internal/store"
)

// Lock creates and locks a transfer inspection task, freezing the full
// snapshot and claiming the bag batch and bag positions exclusively.
func (s *Service) Lock(req catalog.LockRequest) (LockResult, error) {
	at := s.now()
	if err := s.catalog.ValidateLock(req, at); err != nil {
		return LockResult{}, err
	}

	id := newTaskID()
	snap := catalog.NewSnapshot(req)
	task := &inspection.Task{
		ID:         id,
		BagBatch:   req.BagBatch,
		Generation: 1,
		State:      inspection.StatePendingSampling,
		Snapshot:   snap,
		CreatedAt:  at,
	}

	positions := make([]occupancy.BagPositionRecord, 0, len(req.BagPositions))
	for _, p := range req.BagPositions {
		positions = append(positions, occupancy.BagPositionRecord{
			TaskID: id, Batch: req.BagBatch, Position: p,
		})
	}

	leases := make([]occupancy.Lease, 0, len(req.BagPositions)+1)
	leases = append(leases, occupancy.Lease{
		ResourceType: occupancy.ResourceBagBatch,
		ResourceID:   string(req.BagBatch),
		TaskID:       id,
		Generation:   1,
		State:        occupancy.LeaseActive,
	})
	for _, p := range req.BagPositions {
		leases = append(leases, occupancy.Lease{
			ResourceType: occupancy.ResourceBagPosition,
			ResourceID:   string(p),
			Position:     p,
			TaskID:       id,
			Generation:   1,
			State:        occupancy.LeaseActive,
		})
	}

	if err := s.store.CreateTask(ctx(), task, positions, leases); err != nil {
		if err == store.ErrConflict {
			return LockResult{}, domain.NewError(domain.CodeResourceWindowConflict,
				"bag batch or position already occupied")
		}
		return LockResult{}, err
	}
	s.tick()
	s.audit(id, 1, "", domain.AuditLocked, "locked task")

	return LockResult{
		TaskID:     id,
		Generation: 1,
		State:      inspection.StatePendingSampling,
		Summary:    snap,
	}, nil
}
