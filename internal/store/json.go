package store

import (
	"encoding/json"

	"mycocycle-growbag-transfer-gate/internal/domain"
)

// Small JSON and boolean helpers shared by the persistence layer.

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intBool(i int) bool {
	return i != 0
}

func marshalStringSlice(ss []string) string {
	b, _ := json.Marshal(ss)
	return string(b)
}

func unmarshalStringSlice(s string) []string {
	var out []string
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

func marshalDayAges(ds []domain.DayAge) string {
	out := make([]int, len(ds))
	for i, d := range ds {
		out[i] = int(d)
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func unmarshalDayAges(s string) []domain.DayAge {
	var ints []int
	_ = json.Unmarshal([]byte(s), &ints)
	out := make([]domain.DayAge, len(ints))
	for i, d := range ints {
		out[i] = domain.DayAge(d)
	}
	return out
}
