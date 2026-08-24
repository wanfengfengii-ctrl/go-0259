// Package catalog implements the strain and culture rule directory: fictional
// strain lineages, substrate formulation revisions, sterilizer run summaries,
// culture schedule templates, thresholds and reviewer qualifications, plus
// summary matching, freshness checks and immutable lock-snapshot construction.
package catalog

import "mycocycle-growbag-transfer-gate/internal/domain"

// Strain is a fictional strain lineage entry (菇种谱系).
type Strain struct {
	ID                string           `json:"id"`
	Revision          string           `json:"revision"`
	AllowedSubstrates []string         `json:"allowed_substrates"`
	DefaultSchedule   ScheduleTemplate `json:"default_schedule"`
	MaturityMin       domain.Fixed     `json:"maturity_min"`
	MaturityMax       domain.Fixed     `json:"maturity_max"`
}

// Substrate is a substrate formulation revision (培养料配方修订).
type Substrate struct {
	ID             string             `json:"id"`
	Revision       string             `json:"revision"`
	Summary        string             `json:"summary"`
	MoistureTarget domain.Fixed       `json:"moisture_target"`
	PHTarget       domain.Fixed       `json:"ph_target"`
	ValidFrom      domain.LogicalTime `json:"valid_from"`
	Voided         bool               `json:"voided"`
}

// SterilizerRun is a sterilizer run summary (灭菌釜次).
type SterilizerRun struct {
	ID              string             `json:"id"`
	Summary         string             `json:"summary"`
	CompletedAt     domain.LogicalTime `json:"completed_at"`
	InoculationLine string             `json:"inoculation_line"`
	FreshnessWindow domain.LogicalTime `json:"freshness_window"`
}

// ScheduleTemplate is a culture day-age schedule (培养日程模板).
type ScheduleTemplate struct {
	DayAges []domain.DayAge `json:"day_ages"`
}

// Thresholds holds every threshold frozen into a task snapshot.
type Thresholds struct {
	Contamination domain.Fixed `json:"contamination"`
	MaturityMin   domain.Fixed `json:"maturity_min"`
	MaturityMax   domain.Fixed `json:"maturity_max"`
	MoistureMin   domain.Fixed `json:"moisture_min"`
	MoistureMax   domain.Fixed `json:"moisture_max"`
	PHMin         domain.Fixed `json:"ph_min"`
	PHMax         domain.Fixed `json:"ph_max"`
}

// ProbeWindow describes the temperature/humidity probe observation window.
type ProbeWindow struct {
	ProbeID domain.ProbeID     `json:"probe_id"`
	Start   domain.LogicalTime `json:"start"`
	End     domain.LogicalTime `json:"end"`
}

// Snapshot is the immutable lock snapshot (锁定快照). Once built it may only be
// referenced by later steps, never overwritten.
type Snapshot struct {
	StrainRevision    string               `json:"strain_revision"`
	SubstrateRevision string               `json:"substrate_revision"`
	SubstrateSummary  string               `json:"substrate_summary"`
	SterilizerSummary string               `json:"sterilizer_summary"`
	InoculationLine   string               `json:"inoculation_line"`
	BagBatch          domain.BagBatch      `json:"bag_batch"`
	BagPositions      []domain.BagPosition `json:"bag_positions"`
	RackID            domain.RackID        `json:"rack_id"`
	ProbeWindow       ProbeWindow          `json:"probe_window"`
	Schedule          ScheduleTemplate     `json:"schedule"`
	Thresholds        Thresholds           `json:"thresholds"`
	Reviewers         []domain.PersonID    `json:"reviewers"`
}

// LockRequest carries everything submitted when creating and locking a task.
type LockRequest struct {
	StrainRevision    string               `json:"strain_revision"`
	SubstrateRevision string               `json:"substrate_revision"`
	SubstrateSummary  string               `json:"substrate_summary"`
	SterilizerSummary string               `json:"sterilizer_summary"`
	InoculationLine   string               `json:"inoculation_line"`
	BagBatch          domain.BagBatch      `json:"bag_batch"`
	BagPositions      []domain.BagPosition `json:"bag_positions"`
	RackID            domain.RackID        `json:"rack_id"`
	ProbeWindow       ProbeWindow          `json:"probe_window"`
	Schedule          ScheduleTemplate     `json:"schedule"`
	Thresholds        Thresholds           `json:"thresholds"`
	Reviewers         []domain.PersonID    `json:"reviewers"`
}

// NewSnapshot freezes a LockRequest into an immutable Snapshot. All slices are
// copied so later mutation of the request cannot alter the snapshot.
func NewSnapshot(req LockRequest) Snapshot {
	return Snapshot{
		StrainRevision:    req.StrainRevision,
		SubstrateRevision: req.SubstrateRevision,
		SubstrateSummary:  req.SubstrateSummary,
		SterilizerSummary: req.SterilizerSummary,
		InoculationLine:   req.InoculationLine,
		BagBatch:          req.BagBatch,
		BagPositions:      append([]domain.BagPosition(nil), req.BagPositions...),
		RackID:            req.RackID,
		ProbeWindow:       req.ProbeWindow,
		Schedule:          ScheduleTemplate{DayAges: append([]domain.DayAge(nil), req.Schedule.DayAges...)},
		Thresholds:        req.Thresholds,
		Reviewers:         append([]domain.PersonID(nil), req.Reviewers...),
	}
}

// Catalog resolves strain, substrate, sterilizer-run and reviewer metadata and
// performs summary matching, freshness and lock-request validation.
type Catalog interface {
	Strain(revision string) (Strain, bool)
	Substrate(revision string) (Substrate, bool)
	SterilizerRun(summary string) (SterilizerRun, bool)
	Reviewer(person domain.PersonID) (Reviewer, bool)
	SterilizerFresh(summary string, at domain.LogicalTime) (SterilizerRun, bool)
	ValidateLock(req LockRequest, at domain.LogicalTime) error

	Strains() []Strain
	Substrates() []Substrate
	SterilizerRuns() []SterilizerRun
	Reviewers() []Reviewer
	InoculationLines() []InoculationLine
}

// Compile-time assertion that Repo satisfies Catalog.
var _ Catalog = (*Repo)(nil)
