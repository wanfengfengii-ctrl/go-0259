package arbiter

import "mycocycle-growbag-transfer-gate/internal/domain"

// Conclusion is the operator-requested terminal conclusion for a finalize call.
type Conclusion string

const (
	ConclusionTransfer Conclusion = "transfer" // 允许转房
	ConclusionIsolate  Conclusion = "isolate"  // 污染隔离
	ConclusionCancel   Conclusion = "cancel"   // 取消
)

// Review is an independent review record (复核记录).
type Review struct {
	TaskID        domain.TaskID          `json:"task_id"`
	Generation    domain.Generation      `json:"generation"`
	Person        domain.PersonID        `json:"person"`
	Qualification domain.QualificationID `json:"qualification"`
	Conclusion    string                 `json:"conclusion"`
	Digest        string                 `json:"digest"`
	Operation     domain.OperationID     `json:"operation"`
	At            domain.LogicalTime     `json:"at"`
}
