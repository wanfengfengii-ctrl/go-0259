package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

func TestModel_ReviewRejectsSamplingConfirmers(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "sampler approve is rejected and cannot produce a transfer credential",
			run: func(t *testing.T) {
				s := newTestService(t)
				id := toVerifying(t, s)
				submitPhysChem(t, s, id)

				_, err := s.Review(id, ReviewRequest{
					Operation: "review-sampler", Generation: 1, Person: "alice", Conclusion: "approve",
				})
				assertErrorCode(t, err, domain.CodeRoleOverlap)

				reviewApprove(t, s, id, "review-independent", "carol")
				res, err := s.Finalize(id, FinalizeRequest{
					Operation: "finalize-transfer", Generation: 1, Conclusion: "transfer",
				})
				if err == nil {
					t.Fatalf("finalize with one independent reviewer succeeded: final_type=%s credential=%q", res.FinalType, res.Credential)
				}
				assertErrorCode(t, err, domain.CodeRoleOverlap)
			},
		},
		{
			name: "two non sampler qualified reviewers can transfer",
			run: func(t *testing.T) {
				s := newTestService(t)
				id := toReviewReady(t, s)

				res, err := s.Finalize(id, FinalizeRequest{
					Operation: "finalize-transfer", Generation: 1, Conclusion: "transfer",
				})
				if err != nil {
					t.Fatalf("finalize: %v", err)
				}
				if res.FinalType != string(arbiter.FinalTransferable) {
					t.Fatalf("final type = %s, want %s", res.FinalType, arbiter.FinalTransferable)
				}
				if res.State != inspection.StateTransferable {
					t.Fatalf("state = %s, want %s", res.State, inspection.StateTransferable)
				}
				if res.Credential == "" {
					t.Fatal("empty credential")
				}
			},
		},
		{
			name: "unqualified reviewer remains rejected",
			run: func(t *testing.T) {
				s := newTestService(t)
				id := toVerifying(t, s)

				_, err := s.Review(id, ReviewRequest{
					Operation: "review-unqualified", Generation: 1, Person: "eve", Conclusion: "approve",
				})
				assertErrorCode(t, err, domain.CodeRoleOverlap)
			},
		},
		{
			name: "qualified reviewer outside locked set remains rejected",
			run: func(t *testing.T) {
				s := newTestService(t)
				id := toVerifying(t, s)

				_, err := s.Review(id, ReviewRequest{
					Operation: "review-not-locked", Generation: 1, Person: "frank", Conclusion: "approve",
				})
				assertErrorCode(t, err, domain.CodeRoleOverlap)
			},
		},
		{
			name: "duplicate reviewer remains rejected",
			run: func(t *testing.T) {
				s := newTestService(t)
				id := toVerifying(t, s)

				reviewApprove(t, s, id, "review-carol", "carol")
				_, err := s.Review(id, ReviewRequest{
					Operation: "review-carol-again", Generation: 1, Person: "carol", Conclusion: "approve",
				})
				assertErrorCode(t, err, domain.CodeRoleOverlap)
			},
		},
		{
			name: "non approve review remains accepted but does not satisfy transfer",
			run: func(t *testing.T) {
				s := newTestService(t)
				id := toVerifying(t, s)
				submitPhysChem(t, s, id)

				if _, err := s.Review(id, ReviewRequest{
					Operation: "review-isolate", Generation: 1, Person: "carol", Conclusion: "isolate",
				}); err != nil {
					t.Fatalf("isolate review: %v", err)
				}
				reviewApprove(t, s, id, "review-approve", "dave")
				res, err := s.Finalize(id, FinalizeRequest{
					Operation: "finalize-transfer", Generation: 1, Conclusion: "transfer",
				})
				if err == nil {
					t.Fatalf("finalize with one approve and one isolate review succeeded: final_type=%s credential=%q", res.FinalType, res.Credential)
				}
				assertErrorCode(t, err, domain.CodeRoleOverlap)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
