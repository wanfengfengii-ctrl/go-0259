package domain

// AuditEvent is an append-only audit record describing a domain operation.
type AuditEvent struct {
	TaskID     TaskID
	Generation Generation
	At         LogicalTime
	Operation  OperationID
	Action     string
	Detail     string
}

// AuditAction constants describe the kinds of audit events emitted.
const (
	AuditLocked            = "task.locked"
	AuditSamplingConfirmed = "sampling.confirmed"
	AuditSampleSealed      = "sample.sealed"
	AuditOccupancyStarted  = "occupancy.started"
	AuditOccupancyMoved    = "occupancy.moved"
	AuditObserved          = "observation.recorded"
	AuditReadingRecorded   = "reading.recorded"
	AuditDeviceAttempt     = "device.attempt"
	AuditEvidenceRecorded  = "evidence.recorded"
	AuditReviewed          = "review.recorded"
	AuditFinalized         = "task.finalized"
	AuditLateReading       = "late_reading.audited"
)
