// Package db provides the in-memory TupleRepository adapter for the authz service.
//
// Mirrors typescript/src/authz-service/adapters/db/makeInMemoryTupleRepository.ts.
package db

import (
	"fmt"
	"sync"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/shared"
)

// InMemoryTupleStore is a thread-safe, map-backed [authz.TupleRepository].
// Keys are stored as "object|relation|user" strings — same format as the
// TypeScript implementation.
type InMemoryTupleStore struct {
	mu     sync.RWMutex
	tuples map[string]shared.TupleKey
}

// New creates a store pre-seeded with the given tuples.
func New(seed ...shared.TupleKey) *InMemoryTupleStore {
	s := &InMemoryTupleStore{
		tuples: make(map[string]shared.TupleKey, len(seed)),
	}
	for _, k := range seed {
		s.Write(k)
	}
	return s
}

// Compile-time assertion: *InMemoryTupleStore must satisfy TupleRepository.
var _ authz.TupleRepository = (*InMemoryTupleStore)(nil)

// Write adds a tuple to the store (idempotent).
func (s *InMemoryTupleStore) Write(key shared.TupleKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tuples[keyFor(key)] = key
}

// Delete removes a tuple from the store.  No-op if the tuple does not exist.
func (s *InMemoryTupleStore) Delete(key shared.TupleKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tuples, keyFor(key))
}

// Has reports whether the exact tuple (object, relation, user) exists.
func (s *InMemoryTupleStore) Has(object shared.Object, relation shared.Relation, user shared.Subject) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.tuples[keyFor(shared.TupleKey{Object: object, Relation: relation, User: user})]
	return ok
}

// FindByObjectRelation returns all tuples whose object and relation match.
func (s *InMemoryTupleStore) FindByObjectRelation(object shared.Object, relation shared.Relation) []shared.TupleKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []shared.TupleKey
	for _, k := range s.tuples {
		if k.Object == object && k.Relation == relation {
			out = append(out, k)
		}
	}
	return out
}

// FindAll returns a snapshot of tuples, optionally filtered.
func (s *InMemoryTupleStore) FindAll(filter ...authz.TupleFilter) []shared.TupleKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]shared.TupleKey, 0, len(s.tuples))
	for _, k := range s.tuples {
		if len(filter) == 0 || matchesFilter(k, filter[0]) {
			out = append(out, k)
		}
	}
	return out
}

// ── Private helpers ───────────────────────────────────────────────────────────

func keyFor(k shared.TupleKey) string {
	return fmt.Sprintf("%s|%s|%s", k.Object, k.Relation, k.User)
}

func matchesFilter(k shared.TupleKey, f authz.TupleFilter) bool {
	if f.Object != "" && k.Object != f.Object {
		return false
	}
	if f.Relation != "" && k.Relation != f.Relation {
		return false
	}
	return true
}
