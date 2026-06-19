package authz

import (
	"context"
	"sort"
	"sync"

	"rebac-primer/internal/rebac"
)

// InMemoryStore is a thread-safe, map-backed [TupleRepository]. Tuples are keyed
// by their "object|relation|user" triple, so writing the same tuple twice is a
// harmless overwrite.
type InMemoryStore struct {
	mu     sync.RWMutex
	tuples map[rebac.TupleKey]struct{}
}

// NewInMemoryStore creates a store pre-seeded with the given tuples.
func NewInMemoryStore(seed ...rebac.TupleKey) *InMemoryStore {
	s := &InMemoryStore{
		tuples: make(map[rebac.TupleKey]struct{}, len(seed)),
	}
	// Populate the map directly: during construction the store is not yet shared,
	// so we need neither a lock nor a context.
	for _, k := range seed {
		s.tuples[k] = struct{}{}
	}
	return s
}

// Compile-time assertion: *InMemoryStore must satisfy TupleRepository.
var _ TupleRepository = (*InMemoryStore)(nil)

// The context argument is unused here — an in-memory map never blocks — but it is
// part of the port so a real backend can honour cancellation and deadlines. The
// error return is always nil for the same reason.

// Write adds a tuple to the store (idempotent).
func (s *InMemoryStore) Write(_ context.Context, key rebac.TupleKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tuples[key] = struct{}{}
	return nil
}

// Delete removes a tuple from the store. No-op if the tuple does not exist.
func (s *InMemoryStore) Delete(_ context.Context, key rebac.TupleKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tuples, key)
	return nil
}

// Has reports whether the exact tuple (object, relation, user) exists.
func (s *InMemoryStore) Has(_ context.Context, object rebac.Object, relation rebac.Relation, user rebac.Subject) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.tuples[rebac.TupleKey{Object: object, Relation: relation, User: user}]
	return ok, nil
}

// FindByObjectRelation returns all tuples whose object and relation match.
func (s *InMemoryStore) FindByObjectRelation(_ context.Context, object rebac.Object, relation rebac.Relation) ([]rebac.TupleKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []rebac.TupleKey
	for k := range s.tuples {
		if k.Object == object && k.Relation == relation {
			out = append(out, k)
		}
	}
	sortTuples(out)
	return out, nil
}

// FindAll returns a snapshot of tuples, optionally filtered.
func (s *InMemoryStore) FindAll(_ context.Context, filter ...TupleFilter) ([]rebac.TupleKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]rebac.TupleKey, 0, len(s.tuples))
	for k := range s.tuples {
		if len(filter) == 0 || matchesFilter(k, filter[0]) {
			out = append(out, k)
		}
	}
	sortTuples(out)
	return out, nil
}

// ── Private helpers ───────────────────────────────────────────────────────────

func matchesFilter(k rebac.TupleKey, f TupleFilter) bool {
	if f.Object != "" && k.Object != f.Object {
		return false
	}
	if f.Relation != "" && k.Relation != f.Relation {
		return false
	}
	return true
}

func sortTuples(tuples []rebac.TupleKey) {
	sort.Slice(tuples, func(i, j int) bool {
		if tuples[i].Object != tuples[j].Object {
			return tuples[i].Object < tuples[j].Object
		}
		if tuples[i].Relation != tuples[j].Relation {
			return tuples[i].Relation < tuples[j].Relation
		}
		return tuples[i].User < tuples[j].User
	})
}
