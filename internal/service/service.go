// Package service orchestrates the MycoCycle grow-bag transfer gate business
// flows across the catalog, task aggregate, occupancy ledger, maturity and
// physico-chemical ledgers, contamination evidence chain and final arbiter.
// It is the single entry point used by the HTTP API and by public tests.
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/store"
)

// Service is the orchestration facade.
type Service struct {
	store   store.Store
	catalog catalog.Catalog
	clock   *domain.Clock
	scripts *ScriptRegistry
}

// New constructs a Service bound to a store, catalog and logical clock.
func New(st store.Store, cat catalog.Catalog, clock *domain.Clock) *Service {
	return &Service{
		store:   st,
		catalog: cat,
		clock:   clock,
		scripts: NewScriptRegistry(),
	}
}

// Store exposes the underlying store for recovery and audit introspection.
func (s *Service) Store() store.Store { return s.store }

// Clock exposes the controllable logical clock.
func (s *Service) Clock() *domain.Clock { return s.clock }

// Scripts exposes the device script registry for tests and the API.
func (s *Service) Scripts() *ScriptRegistry { return s.scripts }

// now returns the current logical time.
func (s *Service) now() domain.LogicalTime { return s.clock.Now() }

// tick advances the logical clock and returns the new time.
func (s *Service) tick() domain.LogicalTime { return s.clock.Tick() }

func newTaskID() domain.TaskID {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return domain.TaskID("t-" + hex.EncodeToString(b))
}

// digest computes a stable content digest for idempotency comparison.
func digest(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func ctx() context.Context { return context.Background() }
