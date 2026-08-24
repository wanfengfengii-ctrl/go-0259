package inspection

import "mycocycle-growbag-transfer-gate/internal/domain"

// orderedTransitions defines the single legal forward path through the task
// lifecycle. Every advance must follow an edge here; terminal states have no
// outgoing edges and therefore no further transitions.
var orderedTransitions = map[TaskState]TaskState{
	StatePendingLock:            StatePendingSampling,
	StatePendingSampling:        StateSealingSamples,
	StateSealingSamples:         StateOccupying,
	StateOccupying:              StateObserving,
	StateObserving:              StateVerifyingContamination,
	StateVerifyingContamination: StateVerifyingPhysChem,
	StateVerifyingPhysChem:      StatePendingReview,
	StatePendingReview:          StateTransferable,
	StateTransferable:           StateTransferred,
}

// CanAdvance reports whether the state machine permits moving from `from` to
// `to`. Terminal states reject every transition.
func CanAdvance(from, to TaskState) bool {
	if from.IsFinal() {
		return false
	}
	next, ok := orderedTransitions[from]
	return ok && next == to
}

// Advance moves a task to the requested state, rejecting an illegal transition
// with a stable generation/state error.
func Advance(t *Task, to TaskState) error {
	if !CanAdvance(t.State, to) {
		return domain.NewError(domain.CodeFinalStateRejected,
			"illegal state transition", string(t.State), string(to))
	}
	t.State = to
	return nil
}
