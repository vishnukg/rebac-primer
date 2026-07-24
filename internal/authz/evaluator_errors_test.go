package authz_test

import (
	"context"
	"errors"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/rebac"
)

// erroringStore is a RelationshipReader whose reads always fail. It proves the
// evaluator surfaces a backend failure as an error instead of silently denying
// access — a silent deny would look identical to "no permission", hiding outages.
type erroringStore struct{ err error }

func (e erroringStore) Has(context.Context, rebac.Subject, rebac.Relation, rebac.Resource) (bool, error) {
	return false, e.err
}
func (e erroringStore) FindByResourceRelation(context.Context, rebac.Resource, rebac.Relation) ([]rebac.Relationship, error) {
	return nil, e.err
}
func TestGraphEvaluator_PropagatesStoreError(t *testing.T) {
	// Arrange
	sentinel := errors.New("relationship store unavailable")
	ev := authz.NewGraphEvaluator(erroringStore{err: sentinel})

	// Act
	_, err := ev.Evaluate(t.Context(), rebac.CheckRequest{
		Subject:    fixtures.Alice,
		Permission: rebac.PermissionDocumentEdit,
		Resource:   fixtures.RoadmapDocument,
	})

	// Assert
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected store error to propagate, got %v", err)
	}
}

func TestGraphEvaluator_CancelledContextReturnsError(t *testing.T) {
	// Arrange
	ev := authz.NewGraphEvaluator(authz.NewInMemoryStore(fixtures.SeedRelationships()...))

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // already cancelled before the check starts

	// Act
	_, err := ev.Evaluate(ctx, rebac.CheckRequest{
		Subject:    fixtures.Alice,
		Permission: rebac.PermissionDocumentEdit,
		Resource:   fixtures.RoadmapDocument,
	})

	// Assert
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
