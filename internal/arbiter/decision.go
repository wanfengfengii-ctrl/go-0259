// Package arbiter implements the contamination recheck and final arbiter: the
// single-writer terminal decision barrier over transferable, transferred,
// contamination-isolated and cancelled conclusions.
package arbiter

import "mycocycle-growbag-transfer-gate/internal/domain"

// FinalType is the single terminal conclusion of a task.
type FinalType string

const (
	FinalTransferable          FinalType = "transferable"           // 允许转房
	FinalTransferred           FinalType = "transferred"            // 已转房
	FinalContaminationIsolated FinalType = "contamination_isolated" // 污染隔离
	FinalCancelled             FinalType = "cancelled"              // 已取消
)

// Decision is the terminal decision written by the single-writer barrier.
type Decision struct {
	TaskID           domain.TaskID      `json:"task_id"`
	FinalType        FinalType          `json:"final_type"`
	Credential       string             `json:"credential"`
	WinningOperation domain.OperationID `json:"winning_operation"`
	WrittenAt        domain.LogicalTime `json:"written_at"`
	Summary          string             `json:"summary"`
}
