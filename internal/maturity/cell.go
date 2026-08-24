// Package maturity implements the day-age maturity and physico-chemical
// collection ledger: the culture day-age coverage matrix and fixed-point
// physico-chemical readings.
package maturity

import "mycocycle-growbag-transfer-gate/internal/domain"

// ObservationCell is a single day-age x bag-position coverage cell.
type ObservationCell struct {
	TaskID             domain.TaskID      `json:"task_id"`
	Generation         domain.Generation  `json:"generation"`
	DayAge             domain.DayAge      `json:"day_age"`
	Position           domain.BagPosition `json:"position"`
	MyceliumCoverage   domain.Fixed       `json:"mycelium_coverage"` // 菌丝覆盖率整数
	ContaminationCount int                `json:"contamination_count"`
	BagDamage          bool               `json:"bag_damage"`
	Missing            bool               `json:"missing"`
	ObservationSummary string             `json:"summary"`
	Observer           domain.PersonID    `json:"observer"`
}

// ReadingStatus is the parse/derivation status of a physico-chemical reading.
type ReadingStatus string

const (
	ReadingAccepted   ReadingStatus = "accepted"
	ReadingOutOfRange ReadingStatus = "out_of_range"
	ReadingParseError ReadingStatus = "parse_error"
	ReadingOverflow   ReadingStatus = "overflow"
)

// PhysChemReading is a fixed-point physico-chemical reading.
type PhysChemReading struct {
	TaskID       domain.TaskID      `json:"task_id"`
	Generation   domain.Generation  `json:"generation"`
	Position     domain.BagPosition `json:"position"`
	DayAge       domain.DayAge      `json:"day_age"`
	Metric       domain.Metric      `json:"metric"`
	Value        domain.Fixed       `json:"value"`
	SourceDevice domain.DeviceID    `json:"source_device"`
	Status       ReadingStatus      `json:"status"`
	Derived      DerivedJudgment    `json:"derived"` // 派生判定
}

// DerivedJudgment records whether a reading was derived into the task's
// evidence. Only accepted readings within thresholds derive a pass; every
// failure path leaves a non-pass judgment so that derived evidence is never
// written for a rejected or failed value.
type DerivedJudgment string

const (
	DerivedNone DerivedJudgment = "none" // 未派生（解析/范围/算术失败）
	DerivedPass DerivedJudgment = "pass" // 派生通过
	DerivedFail DerivedJudgment = "fail" // 派生不通过
)

// Accepted reports whether the reading parsed and validated successfully.
func (r PhysChemReading) Accepted() bool {
	return r.Status == ReadingAccepted
}
