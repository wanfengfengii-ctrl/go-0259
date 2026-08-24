package inspection

import "mycocycle-growbag-transfer-gate/internal/domain"

// SamplingConfirmation is a single person's confirmation of the sampled bag
// positions, bag batch and task generation. Two distinct qualified persons must
// confirm before sample sealing may begin.
type SamplingConfirmation struct {
	TaskID          domain.TaskID
	Generation      domain.Generation
	Person          domain.PersonID
	Batch           domain.BagBatch
	PositionsDigest string
	Operation       domain.OperationID
	At              domain.LogicalTime
}
