// Package inspection implements the grow-bag transfer inspection task
// aggregate, whose consistency boundary is the bag batch and task generation.
package inspection

import (
	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/domain"
)

// TaskState is the lifecycle state of a transfer inspection task aggregate.
type TaskState string

const (
	StatePendingLock            TaskState = "pending_lock"            // 待锁定
	StatePendingSampling        TaskState = "pending_sampling"        // 待抽检确认
	StateSealingSamples         TaskState = "sealing_samples"         // 样本封存中
	StateOccupying              TaskState = "occupying"               // 架位占用中
	StateObserving              TaskState = "observing"               // 日龄观察中
	StateVerifyingContamination TaskState = "verifying_contamination" // 污染核验中
	StateVerifyingPhysChem      TaskState = "verifying_physchem"      // 理化复测中
	StatePendingReview          TaskState = "pending_review"          // 待独立复核
	StateTransferable           TaskState = "transferable"            // 可转房
	StateTransferred            TaskState = "transferred"             // 已转房
	StateContaminationIsolated  TaskState = "contamination_isolated"  // 污染隔离
	StateCancelled              TaskState = "cancelled"               // 已取消
)

// IsFinal reports whether the state is terminal. Once a task reaches a final
// state, later reads, rechecks, reviews and ordinary operations are rejected.
func (s TaskState) IsFinal() bool {
	switch s {
	case StateTransferable, StateTransferred, StateContaminationIsolated, StateCancelled:
		return true
	default:
		return false
	}
}

// Task is the grow-bag transfer inspection task aggregate.
type Task struct {
	ID           domain.TaskID
	BagBatch     domain.BagBatch
	Generation   domain.Generation
	State        TaskState
	Snapshot     catalog.Snapshot
	CreatedAt    domain.LogicalTime
	FinalVersion int
}
