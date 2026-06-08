package authz_test

import (
	"context"
	"errors"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/rebac"
)

// erroringStore is a TupleRepository whose reads always fail. It proves the
// evaluator surfaces a backend failure as an error instead of silently denying
// access — a silent deny would look identical to "no permission", hiding outages.
type erroringStore struct{ err error }

func (e erroringStore) Has(context.Context, rebac.Object, rebac.Relation, rebac.Subject) (bool, error) {
	return false, e.err
}
func (e erroringStore) FindByObjectRelation(context.Context, rebac.Object, rebac.Relation) ([]rebac.TupleKey, error) {
	return nil, e.err
}
func (e erroringStore) FindAll(context.Context, ...authz.TupleFilter) ([]rebac.TupleKey, error) {
	return nil, e.err
}
func (e erroringStore) Write(context.Context, rebac.TupleKey) error  { return e.err }
func (e erroringStore) Delete(context.Context, rebac.TupleKey) error { return e.err }

func TestGraphEvaluator_PropagatesStoreError(t *testing.T) {
	sentinel := errors.New("tuple store unavailable")
	ev := authz.NewGraphEvaluator(erroringStore{err: sentinel})

	_, err := ev.Evaluate(context.Background(), rebac.CheckRequest{
		User:     fixtures.Alice,
		Relation: rebac.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected store error to propagate, got %v", err)
	}
}

func TestGraphEvaluator_CancelledContextReturnsError(t *testing.T) {
	ev := authz.NewGraphEvaluator(authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before the check starts

	_, err := ev.Evaluate(ctx, rebac.CheckRequest{
		User:     fixtures.Alice,
		Relation: rebac.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
