package catalog

import (
	"mycocycle-growbag-transfer-gate/internal/domain"
)

// Repo is the catalog repository. It holds strain lineages, substrate
// formulation revisions, sterilizer run summaries and reviewer qualifications,
// and provides the matching and freshness checks used when a task is locked.
type Repo struct {
	strains          map[string]Strain
	substrates       map[string]Substrate
	sterilizers      map[string]SterilizerRun
	reviewers        map[domain.PersonID]Reviewer
	inoculationLines map[string]InoculationLine
}

// NewRepo returns an empty repository.
func NewRepo() *Repo {
	return &Repo{
		strains:          make(map[string]Strain),
		substrates:       make(map[string]Substrate),
		sterilizers:      make(map[string]SterilizerRun),
		reviewers:        make(map[domain.PersonID]Reviewer),
		inoculationLines: make(map[string]InoculationLine),
	}
}

// AddStrain registers a strain lineage.
func (r *Repo) AddStrain(s Strain) { r.strains[s.Revision] = s }

// AddSubstrate registers a substrate revision.
func (r *Repo) AddSubstrate(s Substrate) { r.substrates[s.Revision] = s }

// AddSterilizerRun registers a sterilizer run summary.
func (r *Repo) AddSterilizerRun(s SterilizerRun) { r.sterilizers[s.Summary] = s }

// AddReviewer registers a reviewer qualification.
func (r *Repo) AddReviewer(v Reviewer) { r.reviewers[v.Person] = v }

// Strain returns the strain lineage with the given revision.
func (r *Repo) Strain(revision string) (Strain, bool) {
	s, ok := r.strains[revision]
	return s, ok
}

// Substrate returns the substrate revision with the given identifier.
func (r *Repo) Substrate(revision string) (Substrate, bool) {
	s, ok := r.substrates[revision]
	return s, ok
}

// SterilizerRun returns the sterilizer run with the given summary.
func (r *Repo) SterilizerRun(summary string) (SterilizerRun, bool) {
	s, ok := r.sterilizers[summary]
	return s, ok
}

// Reviewer returns the reviewer record for a person.
func (r *Repo) Reviewer(person domain.PersonID) (Reviewer, bool) {
	v, ok := r.reviewers[person]
	return v, ok
}

// SterilizerFresh reports whether the sterilizer run with the given summary is
// still within its freshness window at the given logical time.
func (r *Repo) SterilizerFresh(summary string, at domain.LogicalTime) (SterilizerRun, bool) {
	s, ok := r.sterilizers[summary]
	if !ok {
		return SterilizerRun{}, false
	}
	if at < s.CompletedAt {
		return SterilizerRun{}, false
	}
	if at > s.CompletedAt+s.FreshnessWindow {
		return SterilizerRun{}, false
	}
	return s, true
}

// Strains returns a snapshot of all strain lineages (used for seeding and
// audit introspection).
func (r *Repo) Strains() []Strain {
	out := make([]Strain, 0, len(r.strains))
	for _, s := range r.strains {
		out = append(out, s)
	}
	return out
}

// Substrates returns all substrate revisions.
func (r *Repo) Substrates() []Substrate {
	out := make([]Substrate, 0, len(r.substrates))
	for _, s := range r.substrates {
		out = append(out, s)
	}
	return out
}

// SterilizerRuns returns all sterilizer runs.
func (r *Repo) SterilizerRuns() []SterilizerRun {
	out := make([]SterilizerRun, 0, len(r.sterilizers))
	for _, s := range r.sterilizers {
		out = append(out, s)
	}
	return out
}

// Reviewers returns all reviewer records.
func (r *Repo) Reviewers() []Reviewer {
	out := make([]Reviewer, 0, len(r.reviewers))
	for _, v := range r.reviewers {
		out = append(out, v)
	}
	return out
}
