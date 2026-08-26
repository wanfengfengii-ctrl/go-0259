package service

import (
	"testing"

	"mycocycle-growbag-transfer-gate/internal/arbiter"
	"mycocycle-growbag-transfer-gate/internal/domain"
)

func TestModel_ReviewDuplicateReviewerAndTransferQuorum(t *testing.T) {
	type wantReview struct {
		person     domain.PersonID
		conclusion string
		operation  domain.OperationID
	}

	cases := []struct {
		name              string
		reviews           []ReviewRequest
		wantReviewErrs    []domain.ErrorCode
		wantReviews       []wantReview
		finalConclusion   string
		wantFinalType     string
		wantFinalizeError domain.ErrorCode
	}{
		{
			name: "duplicate cancel cannot replace prior approve",
			reviews: []ReviewRequest{
				{Operation: "carol-approve", Generation: 1, Person: "carol", Conclusion: "approve"},
				{Operation: "carol-cancel", Generation: 1, Person: "carol", Conclusion: "cancel"},
				{Operation: "dave-approve", Generation: 1, Person: "dave", Conclusion: "approve"},
			},
			wantReviewErrs: []domain.ErrorCode{"", domain.CodeRoleOverlap, ""},
			wantReviews: []wantReview{
				{person: "carol", conclusion: "approve", operation: "carol-approve"},
				{person: "dave", conclusion: "approve", operation: "dave-approve"},
			},
			finalConclusion: "transfer",
			wantFinalType:   string(arbiter.FinalTransferable),
		},
		{
			name: "duplicate approve cannot be added under a new operation",
			reviews: []ReviewRequest{
				{Operation: "carol-approve", Generation: 1, Person: "carol", Conclusion: "approve"},
				{Operation: "carol-approve-again", Generation: 1, Person: "carol", Conclusion: "approve"},
			},
			wantReviewErrs: []domain.ErrorCode{"", domain.CodeRoleOverlap},
			wantReviews: []wantReview{
				{person: "carol", conclusion: "approve", operation: "carol-approve"},
			},
		},
		{
			name: "non approve review is retained but excluded from transfer quorum",
			reviews: []ReviewRequest{
				{Operation: "carol-isolate", Generation: 1, Person: "carol", Conclusion: "isolate"},
				{Operation: "dave-approve", Generation: 1, Person: "dave", Conclusion: "approve"},
			},
			wantReviewErrs: []domain.ErrorCode{"", ""},
			wantReviews: []wantReview{
				{person: "carol", conclusion: "isolate", operation: "carol-isolate"},
				{person: "dave", conclusion: "approve", operation: "dave-approve"},
			},
			finalConclusion:   "transfer",
			wantFinalizeError: domain.CodeRoleOverlap,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestService(t)
			id := toVerifying(t, s)
			submitPhysChem(t, s, id)

			for i, req := range tc.reviews {
				_, err := s.Review(id, req)
				if want := tc.wantReviewErrs[i]; want != "" {
					assertErrorCode(t, err, want)
					continue
				}
				if err != nil {
					t.Fatalf("review %s by %s: %v", req.Operation, req.Person, err)
				}
			}

			gotReviews, err := s.store.LoadReviews(ctx(), id)
			if err != nil {
				t.Fatalf("load reviews: %v", err)
			}
			if len(gotReviews) != len(tc.wantReviews) {
				t.Fatalf("reviews = %d, want %d: %+v", len(gotReviews), len(tc.wantReviews), gotReviews)
			}
			for i, want := range tc.wantReviews {
				got := gotReviews[i]
				if got.Person != want.person || got.Conclusion != want.conclusion || got.Operation != want.operation {
					t.Fatalf("review[%d] = person %s conclusion %s operation %s, want person %s conclusion %s operation %s",
						i, got.Person, got.Conclusion, got.Operation, want.person, want.conclusion, want.operation)
				}
			}

			if tc.finalConclusion == "" {
				return
			}
			gotFinal, err := s.Finalize(id, FinalizeRequest{
				Operation:  "final-" + domain.OperationID(tc.finalConclusion),
				Generation: 1,
				Conclusion: tc.finalConclusion,
			})
			if tc.wantFinalizeError != "" {
				assertErrorCode(t, err, tc.wantFinalizeError)
				return
			}
			if err != nil {
				t.Fatalf("finalize %s: %v", tc.finalConclusion, err)
			}
			if gotFinal.FinalType != tc.wantFinalType {
				t.Fatalf("final type = %s, want %s", gotFinal.FinalType, tc.wantFinalType)
			}
		})
	}
}
