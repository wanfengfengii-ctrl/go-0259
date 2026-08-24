// Package occupancy implements the bag-position sample and rack/probe occupancy
// ledger: one-shot locks and concurrency leases over bag batches, sample bag
// positions, culture racks and probe windows.
package occupancy

import "mycocycle-growbag-transfer-gate/internal/domain"

// ResourceType identifies the kind of exclusively leased resource.
type ResourceType string

const (
	ResourceBagBatch    ResourceType = "bag_batch"    // 菌包批号
	ResourceBagPosition ResourceType = "bag_position" // 抽检袋位
	ResourceRack        ResourceType = "rack"         // 培养架位
	ResourceProbeWindow ResourceType = "probe_window" // 探头窗口
)

// LeaseState is the lifecycle state of a lease.
type LeaseState string

const (
	LeaseActive   LeaseState = "active"
	LeaseReleased LeaseState = "released"
)

// Lease is an exclusive, concurrency-adjudicated occupancy lease.
type Lease struct {
	ResourceType  ResourceType       `json:"resource_type"`
	ResourceID    string             `json:"resource_id"` // 资源编号
	Position      domain.BagPosition `json:"position"`    // 袋位或探头编号（架位/批号时为空）
	WindowStart   domain.LogicalTime `json:"window_start"`
	WindowEnd     domain.LogicalTime `json:"window_end"`
	TaskID        domain.TaskID      `json:"task_id"`
	Generation    domain.Generation  `json:"generation"`
	State         LeaseState         `json:"state"`
	ReleaseReason string             `json:"release_reason,omitempty"`
}
