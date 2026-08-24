package service

import (
	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/store"
)

// Review records an independent review. Reviewers must be qualified, distinct
// from the sampling persons and from each other.
func (s *Service) Review(id domain.TaskID, req ReviewRequest) (ReviewResult, error) {
	t, err := s.loadTask(id)
	if err != nil {
		return ReviewResult{}, err
	}
	if err := s.guard(t, req.Generation); err != nil {
		return ReviewResult{}, err
	}
	dg := digest(req)
	if stored, replay, err := s.resolveIdempotency(id, req.Generation, req.Operation, dg); err != nil {
		return ReviewResult{}, err
	} else if replay {
		var out ReviewResult
		if err := replayResult(stored, &out); err != nil {
			return ReviewResult{}, err
		}
		return out, nil
	}
	if !isVerifying(t.State) {
		return ReviewResult{}, domain.NewError(domain.CodeFinalStateRejected,
			"task not ready for review", string(t.State))
	}
	if req.Conclusion != "approve" && req.Conclusion != "isolate" && req.Conclusion != "cancel" {
		return ReviewResult{}, domain.NewError(domain.CodeReadingOutOfRange,
			"unknown review conclusion", req.Conclusion)
	}

	// Role separation: reviewer must be qualified, eligible and not a sampler.
	rv, ok := s.catalog.Reviewer(req.Person)
	if !ok || !rv.IsQualified() {
		return ReviewResult{}, domain.NewError(domain.CodeRoleOverlap,
			"unqualified reviewer", string(req.Person))
	}
	if !personIn(req.Person, t.Snapshot.Reviewers) {
		return ReviewResult{}, domain.NewError(domain.CodeRoleOverlap,
			"reviewer not eligible for task", string(req.Person))
	}
	confs, err := s.store.LoadConfirmations(ctx(), id)
	if err != nil {
		return ReviewResult{}, err
	}
	for _, c := range confs {
		if c.Person == req.Person {
			return ReviewResult{}, domain.NewError(domain.CodeRoleOverlap,
				"reviewer overlaps with sampling person", string(req.Person))
		}
	}

	review := &arbiter.Review{
		TaskID: id, Generation: req.Generation, Person: req.Person,
		Qualification: rv.Qualification, Conclusion: req.Conclusion,
		Digest: dg, Operation: req.Operation, At: s.tick(),
	}
	if err := s.store.SaveReview(ctx(), review); err != nil {
		if err == store.ErrConflict {
			return ReviewResult{}, domain.NewError(domain.CodeRoleOverlap,
				"duplicate reviewer", string(req.Person))
		}
		return ReviewResult{}, err
	}
	s.audit(id, req.Generation, req.Operation, domain.AuditReviewed, string(req.Person))
	return ReviewResult{State: t.State}, nil
}

func personIn(p domain.PersonID, set []domain.PersonID) bool {
	for _, x := range set {
		if x == p {
			return true
		}
	}
	return false
}
