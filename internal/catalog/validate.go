package catalog

import (
	"fmt"

	"mycocycle-growbag-transfer-gate/internal/domain"
)

// ValidateLock checks a lock request against the catalog at the given logical
// time, enforcing revision matching, substrate summary matching, sterilizer
// freshness, bag-position uniqueness, schedule/threshold validity and reviewer
// qualification. It returns a stable domain error carrying deterministically
// sorted reasons.
func (r *Repo) ValidateLock(req LockRequest, at domain.LogicalTime) error {
	// Strain lineage must exist and permit the requested substrate revision.
	strain, ok := r.Strain(req.StrainRevision)
	if !ok {
		return newCatalogError(domain.CodeStrainMismatch, "unknown strain revision",
			req.StrainRevision)
	}
	if !strainAllowsSubstrate(strain, req.SubstrateRevision) {
		return newCatalogError(domain.CodeSubstrateMismatch, "substrate not allowed for strain",
			req.SubstrateRevision)
	}

	// Substrate revision must exist, be non-voided and match its summary.
	sub, ok := r.Substrate(req.SubstrateRevision)
	if !ok {
		return newCatalogError(domain.CodeSubstrateMismatch, "unknown substrate revision",
			req.SubstrateRevision)
	}
	if sub.Voided {
		return newCatalogError(domain.CodeSubstrateMismatch, "substrate revision voided",
			req.SubstrateRevision)
	}
	if sub.Summary != req.SubstrateSummary {
		return newCatalogError(domain.CodeSubstrateMismatch, "substrate summary mismatch",
			req.SubstrateSummary, sub.Summary)
	}

	// Sterilizer run summary must be known and fresh.
	run, ok := r.SterilizerFresh(req.SterilizerSummary, at)
	if !ok {
		return newCatalogError(domain.CodeStaleSterilizer, "sterilizer summary stale or unknown",
			req.SterilizerSummary)
	}
	// The requested inoculation line must match the sterilizer run's bound
	// inoculation line (接种线绑定).
	if run.InoculationLine != req.InoculationLine {
		return newCatalogError(domain.CodeSubstrateMismatch,
			"inoculation line does not match sterilizer run",
			req.InoculationLine, run.InoculationLine)
	}
	// The line must exist and be allowed to inoculate the requested strain.
	line, ok := r.InoculationLine(req.InoculationLine)
	if !ok {
		return newCatalogError(domain.CodeStrainMismatch, "unknown inoculation line",
			req.InoculationLine)
	}
	if !line.AllowsStrain(req.StrainRevision) {
		return newCatalogError(domain.CodeStrainMismatch,
			"inoculation line cannot handle strain",
			req.InoculationLine, req.StrainRevision)
	}

	// Bag positions must be present, unique and non-empty.
	if err := validateBagPositions(req.BagPositions); err != nil {
		return err
	}

	// Schedule and thresholds must be valid.
	if err := ValidateSchedule(req.Schedule); err != nil {
		return err
	}
	if err := ValidateThresholds(req.Thresholds); err != nil {
		return err
	}

	// Probe window must be well-formed.
	if req.ProbeWindow.End <= req.ProbeWindow.Start {
		return newCatalogError(domain.CodeResourceWindowConflict, "probe window end before start")
	}

	// Reviewers must be at least two distinct qualified persons.
	if err := validateReviewers(r, req.Reviewers); err != nil {
		return err
	}

	return nil
}

func strainAllowsSubstrate(s Strain, revision string) bool {
	for _, r := range s.AllowedSubstrates {
		if r == revision {
			return true
		}
	}
	return false
}

func validateBagPositions(positions []domain.BagPosition) error {
	if len(positions) == 0 {
		return newCatalogError(domain.CodeDuplicateBagPosition, "empty bag position set")
	}
	seen := make(map[domain.BagPosition]struct{}, len(positions))
	var dups []string
	for _, p := range positions {
		if p == "" {
			return newCatalogError(domain.CodeDuplicateBagPosition, "empty bag position")
		}
		if _, ok := seen[p]; ok {
			dups = append(dups, string(p))
		}
		seen[p] = struct{}{}
	}
	if len(dups) > 0 {
		return newCatalogError(domain.CodeDuplicateBagPosition, "duplicate bag position", dups...)
	}
	return nil
}

func validateReviewers(r *Repo, reviewers []domain.PersonID) error {
	if len(reviewers) < 2 {
		return newCatalogError(domain.CodeRoleOverlap, "need at least two reviewers")
	}
	seen := make(map[domain.PersonID]struct{}, len(reviewers))
	var problems []string
	for _, p := range reviewers {
		if _, ok := seen[p]; ok {
			problems = append(problems, fmt.Sprintf("duplicate reviewer %s", p))
		}
		seen[p] = struct{}{}
		v, ok := r.Reviewer(p)
		if !ok || !v.IsQualified() {
			problems = append(problems, fmt.Sprintf("unqualified reviewer %s", p))
		}
	}
	if len(problems) > 0 {
		return newCatalogError(domain.CodeRoleOverlap, "invalid reviewer set", problems...)
	}
	return nil
}
