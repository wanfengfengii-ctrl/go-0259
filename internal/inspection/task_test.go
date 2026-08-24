package inspection

import "testing"

func TestTaskStateIsFinal(t *testing.T) {
	finals := []TaskState{
		StateTransferable,
		StateTransferred,
		StateContaminationIsolated,
		StateCancelled,
	}
	for _, s := range finals {
		if !s.IsFinal() {
			t.Fatalf("%s should be final", s)
		}
	}
	nonFinals := []TaskState{
		StatePendingLock,
		StatePendingSampling,
		StateSealingSamples,
		StateOccupying,
		StateObserving,
		StateVerifyingContamination,
		StateVerifyingPhysChem,
		StatePendingReview,
	}
	for _, s := range nonFinals {
		if s.IsFinal() {
			t.Fatalf("%s should not be final", s)
		}
	}
}
