package inspection

import "mycocycle-growbag-transfer-gate/internal/domain"

// OperationRecord is the idempotency ledger entry for one operation id within a
// task generation. The digest captures the request content so that a same-id
// same-content retry returns the stored result while a same-id different-content
// request is rejected without changing business state.
type OperationRecord struct {
	TaskID     domain.TaskID
	Generation domain.Generation
	Operation  domain.OperationID
	Digest     string
	ResultJSON string
	At         domain.LogicalTime
}
