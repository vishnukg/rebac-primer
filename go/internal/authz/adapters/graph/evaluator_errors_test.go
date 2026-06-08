package graph_test

import (
	"context"
	"errors"
	"testing"

	"rebac-primer/internal/authz"
	authzdb "rebac-primer/internal/authz/adapters/db"
	"rebac-primer/internal/authz/adapters/graph"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/shared"
)

// erroringStore is a TupleRepository whose reads always fail. It proves the
// evaluator surfaces a backend failure as an error instead of silently denying
// access — a silent deny would look identical to "no permission", hiding outages.
type erroringStore struct{ err error }

func (e erroringStore) Has(context.Context, shared.Object, shared.Relation, shared.Subject) (bool, error) {
	return false, e.err
}
func (e erroringStore) FindByObjectRelation(context.Context, shared.Object, shared.Relation) ([]shared.TupleKey, error) {
	return nil, e.err
}
func (e erroringStore) FindAll(context.Context, ...authz.TupleFilter) ([]shared.TupleKey, error) {
	return nil, e.err
}
func (e erroringStore) Write(context.Context, shared.TupleKey) error  { return e.err }
func (e erroringStore) Delete(context.Context, shared.TupleKey) error { return e.err }

func TestGraphEvaluator_PropagatesStoreError(t *testing.T) {
	sentinel := errors.New("tuple store unavailable")
	ev := graph.NewGraphEvaluator(erroringStore{err: sentinel})

	_, err := ev.Evaluate(context.Background(), shared.CheckRequest{
		User:     fixtures.Alice,
		Relation: shared.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected store error to propagate, got %v", err)
	}
}

func TestGraphEvaluator_CancelledContextReturnsError(t *testing.T) {
	ev := graph.NewGraphEvaluator(authzdb.New(fixtures.SeedRelationshipTuples()...))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before the check starts

	_, err := ev.Evaluate(ctx, shared.CheckRequest{
		User:     fixtures.Alice,
		Relation: shared.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
