package authz

import (
	"fmt"
	"sync"
)

// TupleReader is the read-only side of the tuple store.
// GraphAuthorizer depends only on this interface — it never needs to write.
type TupleReader interface {
	Has(object Object, relation Relation, user Subject) bool
	FindByObjectRelation(object Object, relation Relation) []TupleKey
}

// TupleWriter is the write side of the tuple store.
type TupleWriter interface {
	Write(key TupleKey)
	Delete(key TupleKey)
}

// TupleStore combines reader and writer and exposes the full set of tuples.
type TupleStore interface {
	TupleReader
	TupleWriter
	All() []TupleKey
}

// InMemoryTupleStore is a thread-safe, map-backed TupleStore.
// Keys are stored as "object|relation|user" strings — the same format as the TS implementation.
type InMemoryTupleStore struct {
	mu     sync.RWMutex
	tuples map[string]TupleKey
}

// NewInMemoryTupleStore creates a store pre-seeded with the given tuples.
func NewInMemoryTupleStore(seed ...TupleKey) *InMemoryTupleStore {
	s := &InMemoryTupleStore{
		tuples: make(map[string]TupleKey, len(seed)),
	}
	for _, k := range seed {
		s.Write(k)
	}
	return s
}

// Write adds a tuple to the store (idempotent).
func (s *InMemoryTupleStore) Write(key TupleKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tuples[keyFor(key)] = key
}

// Delete removes a tuple from the store. No-op if the tuple does not exist.
func (s *InMemoryTupleStore) Delete(key TupleKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tuples, keyFor(key))
}

// Has reports whether the exact tuple (object, relation, user) exists.
func (s *InMemoryTupleStore) Has(object Object, relation Relation, user Subject) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.tuples[keyFor(TupleKey{Object: object, Relation: relation, User: user})]
	return ok
}

// FindByObjectRelation returns all tuples whose object and relation match.
func (s *InMemoryTupleStore) FindByObjectRelation(object Object, relation Relation) []TupleKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []TupleKey
	for _, k := range s.tuples {
		if k.Object == object && k.Relation == relation {
			out = append(out, k)
		}
	}
	return out
}

// All returns a snapshot of every tuple in the store.
func (s *InMemoryTupleStore) All() []TupleKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]TupleKey, 0, len(s.tuples))
	for _, k := range s.tuples {
		out = append(out, k)
	}
	return out
}

func keyFor(k TupleKey) string {
	return fmt.Sprintf("%s|%s|%s", k.Object, k.Relation, k.User)
}
