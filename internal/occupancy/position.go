package occupancy

import "mycocycle-growbag-transfer-gate/internal/domain"

// BagPositionRecord is the locked_bag_positions row: a sampled bag position
// bound to a task, with its sample-seal status, sealing person and seal time.
type BagPositionRecord struct {
	TaskID   domain.TaskID      `json:"task_id"`
	Batch    domain.BagBatch    `json:"batch"`
	Position domain.BagPosition `json:"position"`
	Sealed   bool               `json:"sealed"`
	SealedBy domain.PersonID    `json:"sealed_by"`
	SealedAt domain.LogicalTime `json:"sealed_at"`
}
