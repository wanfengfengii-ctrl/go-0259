// Package store is the persistence boundary for the SQLite WAL store. Every
// write occurs within a single transaction; a failed write never leaves partial
// samples, leases, coverage cells, derived evidence or terminal conclusions
// behind.
package store

import (
	"context"
	"errors"

	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/contamination"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
	"mycocycle-growbag-transfer-gate/internal/maturity"
	"mycocycle-growbag-transfer-gate/internal/occupancy"
)

// ErrConflict is returned when a uniqueness or single-writer adjudication loses
// to a competing transaction.
var ErrConflict = errors.New("store: conflict")

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("store: not found")

// Store is the persistence contract consumed by the API and domain services.
type Store interface {
	// Catalog reference data.
	LoadCatalog() (*catalog.Repo, error)

	// Task aggregate.
	CreateTask(ctx context.Context, t *inspection.Task, positions []occupancy.BagPositionRecord, leases []occupancy.Lease) error
	LoadTask(ctx context.Context, id domain.TaskID) (*inspection.Task, error)
	UpdateTaskState(ctx context.Context, id domain.TaskID, state inspection.TaskState, finalVersion int) error

	// Bag positions.
	LoadBagPositions(ctx context.Context, id domain.TaskID) ([]occupancy.BagPositionRecord, error)
	SealBagPositions(ctx context.Context, id domain.TaskID, positions []occupancy.BagPositionRecord) error

	// Sampling confirmations.
	SaveConfirmation(ctx context.Context, c *inspection.SamplingConfirmation) error
	LoadConfirmations(ctx context.Context, id domain.TaskID) ([]inspection.SamplingConfirmation, error)

	// Occupancy leases.
	AcquireLease(ctx context.Context, l *occupancy.Lease) error
	AcquireLeases(ctx context.Context, leases []occupancy.Lease) error
	TransitionOccupancy(ctx context.Context, id domain.TaskID, release []occupancy.ResourceType, acquire []occupancy.Lease) error
	ReleaseLease(ctx context.Context, id domain.TaskID, rt occupancy.ResourceType, resourceID string) error
	LoadTaskLeases(ctx context.Context, id domain.TaskID) ([]occupancy.Lease, error)
	ActiveLease(ctx context.Context, rt occupancy.ResourceType, resourceID string) (*occupancy.Lease, error)
	ProbeWindowLeases(ctx context.Context, probe domain.ProbeID) ([]occupancy.Lease, error)

	// Maturity cells.
	SaveCell(ctx context.Context, c *maturity.ObservationCell) error
	LoadCells(ctx context.Context, id domain.TaskID, gen domain.Generation) ([]maturity.ObservationCell, error)

	// Physico-chemical readings and device attempts.
	SaveReading(ctx context.Context, r *maturity.PhysChemReading) error
	LoadReadings(ctx context.Context, id domain.TaskID) ([]maturity.PhysChemReading, error)
	SaveAttempt(ctx context.Context, a *domain.DeviceAttempt) error
	LoadAttempts(ctx context.Context, id domain.TaskID) ([]domain.DeviceAttempt, error)

	// Contamination evidence.
	SaveEvidence(ctx context.Context, e *contamination.Evidence) error
	LoadEvidence(ctx context.Context, id domain.TaskID) ([]contamination.Evidence, error)

	// Reviews.
	SaveReview(ctx context.Context, r *arbiter.Review) error
	LoadReviews(ctx context.Context, id domain.TaskID) ([]arbiter.Review, error)

	// Final decisions (single-writer barrier).
	FinalizeTask(ctx context.Context, d *arbiter.Decision) error
	LoadDecision(ctx context.Context, id domain.TaskID) (*arbiter.Decision, error)

	// Idempotency.
	SaveOperation(ctx context.Context, op *inspection.OperationRecord) error
	LoadOperation(ctx context.Context, id domain.TaskID, gen domain.Generation, op domain.OperationID) (*inspection.OperationRecord, error)

	// Audit.
	SaveAudit(ctx context.Context, e *domain.AuditEvent) error
	LoadAudit(ctx context.Context, id domain.TaskID) ([]domain.AuditEvent, error)

	// Listing.
	ListTasks(ctx context.Context) ([]inspection.Task, error)
	LoadTasksByBatch(ctx context.Context, batch domain.BagBatch) ([]inspection.Task, error)
	CountTasksByState(ctx context.Context) (map[inspection.TaskState]int, error)

	// Health.
	Ping(ctx context.Context) error

	// Recovery.
	Recover(ctx context.Context) (RecoveryReport, error)
	PendingDeviceRetries(ctx context.Context) ([]domain.DeviceAttempt, error)

	Close() error
}
