package service

import (
	"reflect"
	"testing"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/inspection"
)

func TestModel_SamplingConfirmationsValidateBeforeQuorum(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "failed unregistered sampler does not count toward quorum",
			run: func(t *testing.T) {
				s := newTestService(t)
				id := lockTask(t, s)

				_, err := s.SamplingConfirm(id, SamplingConfirmRequest{
					Operation: "mallory-op", Generation: 1, Person: "mallory", Batch: "B-001",
					Positions: []domain.BagPosition{"P1", "P2"},
				})
				assertErrorCode(t, err, domain.CodeRoleOverlap)

				alice, err := s.SamplingConfirm(id, SamplingConfirmRequest{
					Operation: "alice-op", Generation: 1, Person: "alice", Batch: "B-001",
					Positions: []domain.BagPosition{"P1", "P2"},
				})
				if err != nil {
					t.Fatalf("alice confirm after failed mallory: %v", err)
				}
				if alice.State != inspection.StatePendingSampling {
					t.Fatalf("state after one qualified confirmation = %s, want %s", alice.State, inspection.StatePendingSampling)
				}
				if alice.Confirmations != 1 {
					t.Fatalf("confirmations after one qualified confirmation = %d, want 1", alice.Confirmations)
				}
				if want := []domain.PersonID{"alice"}; !reflect.DeepEqual(alice.ConfirmedBy, want) {
					t.Fatalf("confirmed_by after one qualified confirmation = %v, want %v", alice.ConfirmedBy, want)
				}

				view, err := s.GetTask(id)
				if err != nil {
					t.Fatalf("get task after one qualified confirmation: %v", err)
				}
				if view.State != inspection.StatePendingSampling {
					t.Fatalf("stored state after one qualified confirmation = %s, want %s", view.State, inspection.StatePendingSampling)
				}

				_, err = s.SampleSeal(id, SampleSealRequest{
					Operation: "seal-too-early", Generation: 1, Person: "alice",
					Positions: []domain.BagPosition{"P1", "P2"},
				})
				assertErrorCode(t, err, domain.CodeFinalStateRejected)

				bob, err := s.SamplingConfirm(id, SamplingConfirmRequest{
					Operation: "bob-op", Generation: 1, Person: "bob", Batch: "B-001",
					Positions: []domain.BagPosition{"P1", "P2"},
				})
				if err != nil {
					t.Fatalf("bob confirm after alice: %v", err)
				}
				if bob.State != inspection.StateSealingSamples {
					t.Fatalf("state after two qualified confirmations = %s, want %s", bob.State, inspection.StateSealingSamples)
				}
				if bob.Confirmations != 2 {
					t.Fatalf("confirmations after two qualified confirmations = %d, want 2", bob.Confirmations)
				}
				if want := []domain.PersonID{"alice", "bob"}; !reflect.DeepEqual(bob.ConfirmedBy, want) {
					t.Fatalf("confirmed_by after two qualified confirmations = %v, want %v", bob.ConfirmedBy, want)
				}
			},
		},
		{
			name: "batch and bag position mismatches do not count toward quorum",
			run: func(t *testing.T) {
				s := newTestService(t)
				id := lockTask(t, s)

				_, err := s.SamplingConfirm(id, SamplingConfirmRequest{
					Operation: "bad-batch", Generation: 1, Person: "alice", Batch: "B-999",
					Positions: []domain.BagPosition{"P1", "P2"},
				})
				assertErrorCode(t, err, domain.CodeSubstrateMismatch)

				_, err = s.SamplingConfirm(id, SamplingConfirmRequest{
					Operation: "bad-positions", Generation: 1, Person: "bob", Batch: "B-001",
					Positions: []domain.BagPosition{"P1", "P3"},
				})
				assertErrorCode(t, err, domain.CodeSubstrateMismatch)

				alice, err := s.SamplingConfirm(id, SamplingConfirmRequest{
					Operation: "alice-op", Generation: 1, Person: "alice", Batch: "B-001",
					Positions: []domain.BagPosition{"P1", "P2"},
				})
				if err != nil {
					t.Fatalf("alice confirm after substrate errors: %v", err)
				}
				if alice.State != inspection.StatePendingSampling || alice.Confirmations != 1 {
					t.Fatalf("after substrate errors, first valid result = %+v, want pending_sampling with one confirmation", alice)
				}
				if want := []domain.PersonID{"alice"}; !reflect.DeepEqual(alice.ConfirmedBy, want) {
					t.Fatalf("confirmed_by after substrate errors = %v, want %v", alice.ConfirmedBy, want)
				}

				bob, err := s.SamplingConfirm(id, SamplingConfirmRequest{
					Operation: "bob-op", Generation: 1, Person: "bob", Batch: "B-001",
					Positions: []domain.BagPosition{"P1", "P2"},
				})
				if err != nil {
					t.Fatalf("bob confirm after substrate errors: %v", err)
				}
				if bob.State != inspection.StateSealingSamples || bob.Confirmations != 2 {
					t.Fatalf("after substrate errors, second valid result = %+v, want sealing_samples with two confirmations", bob)
				}
				if want := []domain.PersonID{"alice", "bob"}; !reflect.DeepEqual(bob.ConfirmedBy, want) {
					t.Fatalf("confirmed_by after second valid confirmation = %v, want %v", bob.ConfirmedBy, want)
				}
			},
		},
		{
			name: "duplicate replay and generation checks remain stable",
			run: func(t *testing.T) {
				s := newTestService(t)
				id := lockTask(t, s)
				aliceReq := SamplingConfirmRequest{
					Operation: "alice-op", Generation: 1, Person: "alice", Batch: "B-001",
					Positions: []domain.BagPosition{"P1", "P2"},
				}

				first, err := s.SamplingConfirm(id, aliceReq)
				if err != nil {
					t.Fatalf("first alice confirm: %v", err)
				}
				replay, err := s.SamplingConfirm(id, aliceReq)
				if err != nil {
					t.Fatalf("idempotent alice replay: %v", err)
				}
				if !reflect.DeepEqual(replay, first) {
					t.Fatalf("idempotent replay = %+v, want %+v", replay, first)
				}

				_, err = s.SamplingConfirm(id, SamplingConfirmRequest{
					Operation: "duplicate-alice", Generation: 1, Person: "alice", Batch: "B-001",
					Positions: []domain.BagPosition{"P1", "P2"},
				})
				assertErrorCode(t, err, domain.CodeRoleOverlap)

				_, err = s.SamplingConfirm(id, SamplingConfirmRequest{
					Operation: "wrong-generation", Generation: 2, Person: "carol", Batch: "B-001",
					Positions: []domain.BagPosition{"P1", "P2"},
				})
				assertErrorCode(t, err, domain.CodeGenerationConflict)

				bob, err := s.SamplingConfirm(id, SamplingConfirmRequest{
					Operation: "bob-op", Generation: 1, Person: "bob", Batch: "B-001",
					Positions: []domain.BagPosition{"P1", "P2"},
				})
				if err != nil {
					t.Fatalf("bob confirm after replay and rejected attempts: %v", err)
				}
				if bob.State != inspection.StateSealingSamples || bob.Confirmations != 2 {
					t.Fatalf("after replay and rejected attempts, second valid result = %+v, want sealing_samples with two confirmations", bob)
				}
				if want := []domain.PersonID{"alice", "bob"}; !reflect.DeepEqual(bob.ConfirmedBy, want) {
					t.Fatalf("confirmed_by after replay and rejected attempts = %v, want %v", bob.ConfirmedBy, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
