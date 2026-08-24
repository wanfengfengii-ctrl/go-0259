package domain

import "testing"

func TestErrorSortedReasons(t *testing.T) {
	e := NewError(CodeDuplicateBagPosition, "duplicate", "B2", "A1", "B1")
	want := []string{"A1", "B1", "B2"}
	if len(e.Reasons) != len(want) {
		t.Fatalf("reasons = %v, want %v", e.Reasons, want)
	}
	for i := range want {
		if e.Reasons[i] != want[i] {
			t.Fatalf("reason[%d] = %q, want %q", i, e.Reasons[i], want[i])
		}
	}
}

func TestErrorString(t *testing.T) {
	e := NewError(CodeFinalStateRejected, "already final")
	if e.Error() == "" {
		t.Fatal("empty error string")
	}
}
