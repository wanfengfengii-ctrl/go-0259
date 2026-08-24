package service

import (
	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

// LockResult is returned when a transfer inspection task is locked.
type LockResult struct {
	TaskID     domain.TaskID        `json:"task_id"`
	Generation domain.Generation    `json:"generation"`
	State      inspection.TaskState `json:"state"`
	Summary    catalog.Snapshot     `json:"summary"`
}

// SamplingConfirmRequest submits one person's sampling confirmation.
type SamplingConfirmRequest struct {
	Operation  domain.OperationID   `json:"operation"`
	Generation domain.Generation    `json:"generation"`
	Person     domain.PersonID      `json:"person"`
	Batch      domain.BagBatch      `json:"bag_batch"`
	Positions  []domain.BagPosition `json:"bag_positions"`
}

// SamplingConfirmResult reports a confirmation and the resulting state.
type SamplingConfirmResult struct {
	State         inspection.TaskState `json:"state"`
	ConfirmedBy   []domain.PersonID    `json:"confirmed_by"`
	Confirmations int                  `json:"confirmations"`
}

// SampleSealRequest completes bag-position sample sealing.
type SampleSealRequest struct {
	Operation  domain.OperationID   `json:"operation"`
	Generation domain.Generation    `json:"generation"`
	Person     domain.PersonID      `json:"person"`
	Positions  []domain.BagPosition `json:"bag_positions"`
}

// SampleSealResult reports the sealed positions and resulting state.
type SampleSealResult struct {
	State  inspection.TaskState `json:"state"`
	Sealed []SealedPosition     `json:"sealed"`
}

// SealedPosition is one sealed bag position.
type SealedPosition struct {
	Position domain.BagPosition `json:"position"`
	SealedBy domain.PersonID    `json:"sealed_by"`
}

// OccupancyStartRequest starts rack and probe window occupancy.
type OccupancyStartRequest struct {
	Operation  domain.OperationID `json:"operation"`
	Generation domain.Generation  `json:"generation"`
	RackID     domain.RackID      `json:"rack_id"`
	ProbeID    domain.ProbeID     `json:"probe_id"`
	ProbeStart domain.LogicalTime `json:"probe_start"`
	ProbeEnd   domain.LogicalTime `json:"probe_end"`
}

// OccupancyMoveRequest moves a task to a different rack/window.
type OccupancyMoveRequest struct {
	Operation  domain.OperationID `json:"operation"`
	Generation domain.Generation  `json:"generation"`
	RackID     domain.RackID      `json:"rack_id"`
	ProbeID    domain.ProbeID     `json:"probe_id"`
	ProbeStart domain.LogicalTime `json:"probe_start"`
	ProbeEnd   domain.LogicalTime `json:"probe_end"`
}

// OccupancyResult reports the granted leases and resulting state.
type OccupancyResult struct {
	State  inspection.TaskState `json:"state"`
	RackID domain.RackID        `json:"rack_id"`
	Probe  domain.ProbeID       `json:"probe_id"`
}

// ObservationRequest submits a single day-age x position coverage cell.
type ObservationRequest struct {
	Operation          domain.OperationID `json:"operation"`
	Generation         domain.Generation  `json:"generation"`
	DayAge             domain.DayAge      `json:"day_age"`
	Position           domain.BagPosition `json:"position"`
	MyceliumCoverage   string             `json:"mycelium_coverage"`
	ContaminationCount int                `json:"contamination_count"`
	BagDamage          bool               `json:"bag_damage"`
	Missing            bool               `json:"missing"`
	Summary            string             `json:"summary"`
	Observer           domain.PersonID    `json:"observer"`
}

// ObservationResult reports the resulting state after an observation.
type ObservationResult struct {
	State   inspection.TaskState `json:"state"`
	Missing int                  `json:"missing_cells"`
}

// DeviceReadingRequest submits a single device reading.
type DeviceReadingRequest struct {
	Operation  domain.OperationID `json:"operation"`
	Generation domain.Generation  `json:"generation"`
	DeviceType domain.DeviceType  `json:"device_type"`
	DeviceID   domain.DeviceID    `json:"device_id"`
	Metric     domain.Metric      `json:"metric"`
	Value      string             `json:"value"`
	Position   domain.BagPosition `json:"position"`
	DayAge     domain.DayAge      `json:"day_age"`
	ScriptSeq  int                `json:"script_seq"`
}

// DeviceReadingResult reports a device invocation outcome.
type DeviceReadingResult struct {
	Result     domain.AttemptResult `json:"result"`
	RetryCount int                  `json:"retry_count"`
	ReadingID  int64                `json:"reading_id,omitempty"`
	State      inspection.TaskState `json:"state"`
}

// ContaminationRecheckRequest records contamination recheck evidence.
type ContaminationRecheckRequest struct {
	Operation         domain.OperationID `json:"operation"`
	Generation        domain.Generation  `json:"generation"`
	RecheckGeneration domain.Generation  `json:"recheck_generation"`
	Position          domain.BagPosition `json:"position"`
	DayAge            domain.DayAge      `json:"day_age"`
	Well              string             `json:"well"`
	Source            string             `json:"source"`
	Positive          bool               `json:"positive"`
	Summary           string             `json:"summary"`
}

// ContaminationRecheckResult reports the recorded evidence.
type ContaminationRecheckResult struct {
	Version int `json:"version"`
}

// ReviewRequest submits an independent review.
type ReviewRequest struct {
	Operation  domain.OperationID `json:"operation"`
	Generation domain.Generation  `json:"generation"`
	Person     domain.PersonID    `json:"person"`
	Conclusion string             `json:"conclusion"`
}

// ReviewResult reports the recorded review.
type ReviewResult struct {
	State inspection.TaskState `json:"state"`
}

// FinalizeRequest competes to write the single terminal conclusion.
type FinalizeRequest struct {
	Operation  domain.OperationID `json:"operation"`
	Generation domain.Generation  `json:"generation"`
	Conclusion string             `json:"conclusion"`
}

// FinalizeResult reports the terminal decision.
type FinalizeResult struct {
	FinalType  string               `json:"final_type"`
	Credential string               `json:"credential"`
	State      inspection.TaskState `json:"state"`
}

// TaskView is the read model returned by task queries.
type TaskView struct {
	TaskID     domain.TaskID        `json:"task_id"`
	BagBatch   domain.BagBatch      `json:"bag_batch"`
	Generation domain.Generation    `json:"generation"`
	State      inspection.TaskState `json:"state"`
	Summary    catalog.Snapshot     `json:"summary"`
	FinalType  string               `json:"final_type,omitempty"`
	Credential string               `json:"credential,omitempty"`
}
