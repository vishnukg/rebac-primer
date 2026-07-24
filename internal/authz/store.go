package authz

import (
	"cmp"
	"context"
	"slices"
	"sync"

	"rebac-primer/internal/rebac"
)

// InMemoryStore is a thread-safe, map-backed relationship store. Relationships
// are keyed by their subject-relation-resource triple, so writing the same fact
// twice is a harmless overwrite.
type InMemoryStore struct {
	mu            sync.RWMutex
	relationships map[rebac.Relationship]struct{}
}

// NewInMemoryStore creates a store pre-seeded with the given relationships.
func NewInMemoryStore(seed ...rebac.Relationship) *InMemoryStore {
	s := &InMemoryStore{
		relationships: make(map[rebac.Relationship]struct{}, len(seed)),
	}
	// Populate the map directly: during construction the store is not yet shared,
	// so we need neither a lock nor a context.
	for _, k := range seed {
		s.relationships[k] = struct{}{}
	}
	return s
}

// Compile-time assertion: *InMemoryStore must satisfy RelationshipRepository.
var (
	_ RelationshipReader     = (*InMemoryStore)(nil)
	_ RelationshipWriter     = (*InMemoryStore)(nil)
	_ RelationshipLister     = (*InMemoryStore)(nil)
	_ RelationshipRepository = (*InMemoryStore)(nil)
)

// The context argument is unused here — an in-memory map never blocks — but it is
// part of the port so a real backend can honour cancellation and deadlines. The
// error return is always nil for the same reason.

// Write adds a relationship to the store (idempotent).
func (s *InMemoryStore) Write(_ context.Context, relationship rebac.Relationship) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.relationships[relationship] = struct{}{}
	return nil
}

// Delete removes a relationship from the store. No-op if it does not exist.
func (s *InMemoryStore) Delete(_ context.Context, relationship rebac.Relationship) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.relationships, relationship)
	return nil
}

// Has reports whether the exact (subject, relation, resource) relationship exists.
func (s *InMemoryStore) Has(_ context.Context, subject rebac.Subject, relation rebac.Relation, resource rebac.Resource) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.relationships[rebac.Relationship{Subject: subject, Relation: relation, Resource: resource}]
	return ok, nil
}

// FindByResourceRelation returns all relationships whose resource and relation match.
func (s *InMemoryStore) FindByResourceRelation(_ context.Context, resource rebac.Resource, relation rebac.Relation) ([]rebac.Relationship, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []rebac.Relationship
	for relationship := range s.relationships {
		if relationship.Resource == resource && relationship.Relation == relation {
			out = append(out, relationship)
		}
	}
	sortRelationships(out)
	return out, nil
}

// FindAll returns a snapshot of relationships, optionally filtered.
func (s *InMemoryStore) FindAll(_ context.Context, filter ...RelationshipFilter) ([]rebac.Relationship, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]rebac.Relationship, 0, len(s.relationships))
	for relationship := range s.relationships {
		if len(filter) == 0 || matchesFilter(relationship, filter[0]) {
			out = append(out, relationship)
		}
	}
	sortRelationships(out)
	return out, nil
}

// ── Private helpers ───────────────────────────────────────────────────────────

func matchesFilter(relationship rebac.Relationship, filter RelationshipFilter) bool {
	if filter.Resource != "" && relationship.Resource != filter.Resource {
		return false
	}
	if filter.Relation != "" && relationship.Relation != filter.Relation {
		return false
	}
	return true
}

func sortRelationships(relationships []rebac.Relationship) {
	slices.SortFunc(relationships, func(a, b rebac.Relationship) int {
		if n := cmp.Compare(a.Resource, b.Resource); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Relation, b.Relation); n != 0 {
			return n
		}
		return cmp.Compare(a.Subject, b.Subject)
	})
}
