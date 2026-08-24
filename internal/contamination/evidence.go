// Package contamination implements the contamination recheck evidence chain:
// non-overwritable contamination evidence versioned by recheck generation.
package contamination

import "mycocycle-growbag-transfer-gate/internal/domain"

// Evidence is an immutable contamination recheck evidence record.
type Evidence struct {
	TaskID            domain.TaskID      `json:"task_id"`
	RecheckGeneration domain.Generation  `json:"recheck_generation"`
	Version           int                `json:"version"`
	Position          domain.BagPosition `json:"position"`
	DayAge            domain.DayAge      `json:"day_age"`
	Well              string             `json:"well"`
	Source            string             `json:"source"`
	Positive          bool               `json:"positive"`
	Summary           string             `json:"summary"`
}
