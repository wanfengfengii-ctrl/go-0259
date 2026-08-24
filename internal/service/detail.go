package service

import (
	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/contamination"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/maturity"
	"mycocycle-growbag-transfer-gate/internal/occupancy"
)

// TaskDetail is the full read model of a task: the aggregate, its locked bag
// positions, leases, coverage cells, physico-chemical readings, contamination
// evidence, reviews and any terminal decision.
type TaskDetail struct {
	TaskID       domain.TaskID                 `json:"task_id"`
	BagBatch     domain.BagBatch               `json:"bag_batch"`
	Generation   domain.Generation             `json:"generation"`
	State        string                        `json:"state"`
	Snapshot     TaskSnapshot                  `json:"snapshot"`
	BagPositions []occupancy.BagPositionRecord `json:"bag_positions"`
	Leases       []occupancy.Lease             `json:"leases"`
	Cells        []maturity.ObservationCell    `json:"cells"`
	Readings     []maturity.PhysChemReading    `json:"readings"`
	Evidence     []contamination.Evidence      `json:"evidence"`
	Reviews      []arbiter.Review              `json:"reviews"`
	Decision     *arbiter.Decision             `json:"decision,omitempty"`
	Audit        []domain.AuditEvent           `json:"audit"`
}

// TaskSnapshot mirrors the locked snapshot fields in a stable JSON shape.
type TaskSnapshot struct {
	StrainRevision    string               `json:"strain_revision"`
	SubstrateRevision string               `json:"substrate_revision"`
	SubstrateSummary  string               `json:"substrate_summary"`
	SterilizerSummary string               `json:"sterilizer_summary"`
	InoculationLine   string               `json:"inoculation_line"`
	BagBatch          domain.BagBatch      `json:"bag_batch"`
	BagPositions      []domain.BagPosition `json:"bag_positions"`
	RackID            domain.RackID        `json:"rack_id"`
	ProbeWindow       probeWindowView      `json:"probe_window"`
	DayAges           []domain.DayAge      `json:"day_ages"`
	Thresholds        thresholdsView       `json:"thresholds"`
	Reviewers         []domain.PersonID    `json:"reviewers"`
}

type probeWindowView struct {
	ProbeID domain.ProbeID     `json:"probe_id"`
	Start   domain.LogicalTime `json:"start"`
	End     domain.LogicalTime `json:"end"`
}

type thresholdsView struct {
	Contamination domain.Fixed `json:"contamination"`
	MaturityMin   domain.Fixed `json:"maturity_min"`
	MaturityMax   domain.Fixed `json:"maturity_max"`
	MoistureMin   domain.Fixed `json:"moisture_min"`
	MoistureMax   domain.Fixed `json:"moisture_max"`
	PHMin         domain.Fixed `json:"ph_min"`
	PHMax         domain.Fixed `json:"ph_max"`
}

// GetTaskDetail assembles the full task read model from every persisted table.
func (s *Service) GetTaskDetail(id domain.TaskID) (TaskDetail, error) {
	t, err := s.loadTask(id)
	if err != nil {
		return TaskDetail{}, err
	}
	d := TaskDetail{
		TaskID:     t.ID,
		BagBatch:   t.BagBatch,
		Generation: t.Generation,
		State:      string(t.State),
		Snapshot:   snapshotView(t.Snapshot),
	}
	if positions, err := s.store.LoadBagPositions(ctx(), id); err == nil {
		d.BagPositions = positions
	}
	if leases, err := s.store.LoadTaskLeases(ctx(), id); err == nil {
		d.Leases = leases
	}
	if cells, err := s.store.LoadCells(ctx(), id, t.Generation); err == nil {
		d.Cells = cells
	}
	if readings, err := s.store.LoadReadings(ctx(), id); err == nil {
		d.Readings = readings
	}
	if evidence, err := s.store.LoadEvidence(ctx(), id); err == nil {
		d.Evidence = evidence
	}
	if reviews, err := s.store.LoadReviews(ctx(), id); err == nil {
		d.Reviews = reviews
	}
	decision, err := s.store.LoadDecision(ctx(), id)
	if err != nil {
		return TaskDetail{}, err
	}
	d.Decision = decision
	if audit, err := s.store.LoadAudit(ctx(), id); err == nil {
		d.Audit = audit
	}
	return d, nil
}

func snapshotView(snap catalog.Snapshot) TaskSnapshot {
	return TaskSnapshot{
		StrainRevision:    snap.StrainRevision,
		SubstrateRevision: snap.SubstrateRevision,
		SubstrateSummary:  snap.SubstrateSummary,
		SterilizerSummary: snap.SterilizerSummary,
		InoculationLine:   snap.InoculationLine,
		BagBatch:          snap.BagBatch,
		BagPositions:      snap.BagPositions,
		RackID:            snap.RackID,
		ProbeWindow: probeWindowView{
			ProbeID: snap.ProbeWindow.ProbeID,
			Start:   snap.ProbeWindow.Start,
			End:     snap.ProbeWindow.End,
		},
		DayAges: snap.Schedule.DayAges,
		Thresholds: thresholdsView{
			Contamination: snap.Thresholds.Contamination,
			MaturityMin:   snap.Thresholds.MaturityMin,
			MaturityMax:   snap.Thresholds.MaturityMax,
			MoistureMin:   snap.Thresholds.MoistureMin,
			MoistureMax:   snap.Thresholds.MoistureMax,
			PHMin:         snap.Thresholds.PHMin,
			PHMax:         snap.Thresholds.PHMax,
		},
		Reviewers: snap.Reviewers,
	}
}
