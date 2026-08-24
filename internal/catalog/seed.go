package catalog

import "mycocycle-growbag-transfer-gate/internal/domain"

// Seed populates a repository with the deterministic fictional catalog used by
// the service. The strain lineages, substrate revisions, sterilizer runs and
// reviewer qualifications are stable across restarts so that lock validation
// and restart recovery behave identically.
func Seed(r *Repo) {
	// Strains and their allowed substrate revisions.
	r.AddStrain(Strain{
		ID:                "pleurotus-ostreatus",
		Revision:          "strain-po-2024.03",
		AllowedSubstrates: []string{"sub-hw-01", "sub-cs-02"},
		DefaultSchedule:   ScheduleTemplate{DayAges: []domain.DayAge{1, 3, 5, 7}},
		MaturityMin:       domain.MustFixed(700, 1),
		MaturityMax:       domain.MustFixed(1000, 1),
	})
	r.AddStrain(Strain{
		ID:                "hypsizygus-marmoreus",
		Revision:          "strain-hm-2024.02",
		AllowedSubstrates: []string{"sub-hw-01", "sub-cs-02"},
		DefaultSchedule:   ScheduleTemplate{DayAges: []domain.DayAge{2, 5, 8, 11}},
		MaturityMin:       domain.MustFixed(680, 1),
		MaturityMax:       domain.MustFixed(1000, 1),
	})
	r.AddStrain(Strain{
		ID:                "flammulina-velutipes",
		Revision:          "strain-fv-2024.01",
		AllowedSubstrates: []string{"sub-cs-02"},
		DefaultSchedule:   ScheduleTemplate{DayAges: []domain.DayAge{1, 4, 7}},
		MaturityMin:       domain.MustFixed(660, 1),
		MaturityMax:       domain.MustFixed(1000, 1),
	})
	r.AddStrain(Strain{
		ID:                "lentinula-edodes",
		Revision:          "strain-le-2024.01",
		AllowedSubstrates: []string{"sub-hw-01"},
		DefaultSchedule:   ScheduleTemplate{DayAges: []domain.DayAge{2, 4, 6}},
		MaturityMin:       domain.MustFixed(650, 1),
		MaturityMax:       domain.MustFixed(1000, 1),
	})

	// Substrate formulation revisions.
	r.AddSubstrate(Substrate{
		ID:             "hardwood-sawdust",
		Revision:       "sub-hw-01",
		Summary:        "hardwood sawdust + wheat bran 78:20",
		MoistureTarget: domain.MustFixed(650, 1),
		PHTarget:       domain.MustFixed(600, 2),
		ValidFrom:      1,
		Voided:         false,
	})
	r.AddSubstrate(Substrate{
		ID:             "corncob-sawdust",
		Revision:       "sub-cs-02",
		Summary:        "corncob + sawdust 60:38",
		MoistureTarget: domain.MustFixed(620, 1),
		PHTarget:       domain.MustFixed(610, 2),
		ValidFrom:      1,
		Voided:         false,
	})

	// An additional substrate revision for richer catalog matching.
	r.AddSubstrate(Substrate{
		ID:             "cottonseed-hull",
		Revision:       "sub-ch-03",
		Summary:        "cottonseed hull + sawdust 55:43",
		MoistureTarget: domain.MustFixed(640, 1),
		PHTarget:       domain.MustFixed(605, 2),
		ValidFrom:      1,
		Voided:         false,
	})

	// Sterilizer runs. run-A and run-B are fresh at logical time zero so a lock
	// can succeed immediately; run-stale is already outside its freshness window.
	r.AddSterilizerRun(SterilizerRun{
		ID:              "autoclave-A",
		Summary:         "run-A-2026-08-21",
		CompletedAt:     0,
		InoculationLine: "line-1",
		FreshnessWindow: 10000,
	})
	r.AddSterilizerRun(SterilizerRun{
		ID:              "autoclave-B",
		Summary:         "run-B-2026-08-22",
		CompletedAt:     0,
		InoculationLine: "line-2",
		FreshnessWindow: 10000,
	})
	r.AddSterilizerRun(SterilizerRun{
		ID:              "autoclave-C",
		Summary:         "run-C-2026-08-23",
		CompletedAt:     0,
		InoculationLine: "line-2",
		FreshnessWindow: 10000,
	})
	r.AddSterilizerRun(SterilizerRun{
		ID:              "autoclave-stale",
		Summary:         "run-stale-2026-01-01",
		CompletedAt:     -100,
		InoculationLine: "line-1",
		FreshnessWindow: 50,
	})

	// Inoculation line capabilities.
	r.AddInoculationLine(InoculationLine{ID: "line-1", AllowedStrains: []string{"strain-po-2024.03"}})
	r.AddInoculationLine(InoculationLine{ID: "line-2", AllowedStrains: []string{"strain-le-2024.01"}})

	// Reviewer qualifications.
	r.AddReviewer(Reviewer{Person: "alice", Qualification: "qc-qa-01", Valid: true})
	r.AddReviewer(Reviewer{Person: "bob", Qualification: "qc-qa-02", Valid: true})
	r.AddReviewer(Reviewer{Person: "carol", Qualification: "qc-qa-03", Valid: true})
	r.AddReviewer(Reviewer{Person: "dave", Qualification: "qc-qa-04", Valid: true})
	r.AddReviewer(Reviewer{Person: "eve", Qualification: "qc-qa-05", Valid: false})
	r.AddReviewer(Reviewer{Person: "frank", Qualification: "qc-qa-06", Valid: true})
	r.AddReviewer(Reviewer{Person: "grace", Qualification: "qc-qa-07", Valid: true})
}
