package concurrency_test

import (
	"context"
	"errors"
	"testing"

	"rebac-primer/examples/concurrency"
	authzdb "rebac-primer/internal/authz/adapters/db"
	"rebac-primer/internal/authz/adapters/graph"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/shared"
)

// newEvaluator builds a graph evaluator over the standard fixture tuples.
// The concurrency helpers under test work with any authz.Evaluator; here we
// drive them with the real graph evaluator.
func newEvaluator(extra ...shared.TupleKey) *graph.GraphEvaluator {
	all := append(fixtures.SeedRelationshipTuples(), extra...)
	return graph.NewGraphEvaluator(authzdb.New(all...))
}

// blockingEvaluator is a fake Checker whose Evaluate does no work until the
// context is cancelled, then reports the context error. It lets us exercise
// AllPermissions' cancellation path deterministically.
type blockingEvaluator struct{}

func (blockingEvaluator) Evaluate(ctx context.Context, _ shared.CheckRequest) (shared.CheckResult, error) {
	<-ctx.Done()
	return shared.CheckResult{}, ctx.Err()
}

func TestAllPermissions_CancelledContextReturnsError(t *testing.T) {
	// Arrange: a context that is already cancelled, and an evaluator that only
	// unblocks once the context is done.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	summary, err := concurrency.AllPermissions(ctx, blockingEvaluator{}, fixtures.Alice, fixtures.RoadmapDocument)

	// Assert: AllPermissions must surface the cancellation, not block or return a
	// partial summary. (-race confirms no goroutine writes after we return.)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if summary != nil {
		t.Errorf("expected nil summary on cancellation, got %v", summary)
	}
}

func TestAllPermissions_ReturnsFullSummaryForEditor(t *testing.T) {
	ev := newEvaluator()

	summary, err := concurrency.AllPermissions(context.Background(), ev, fixtures.Alice, fixtures.RoadmapDocument)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[shared.Relation]bool{
		shared.RelationDocumentCanRead:    true,
		shared.RelationDocumentCanComment: true,
		shared.RelationDocumentCanEdit:    true,
		shared.RelationDocumentCanDelete:  false,
	}
	for rel, expected := range want {
		if got := summary[rel]; got != expected {
			t.Errorf("summary[%s] = %v, want %v", rel, got, expected)
		}
	}
}

func TestAllPermissions_ViewerCanReadButNotEdit(t *testing.T) {
	ev := newEvaluator()

	summary, err := concurrency.AllPermissions(context.Background(), ev, fixtures.Bob, fixtures.RoadmapDocument)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !summary[shared.RelationDocumentCanRead] {
		t.Error("expected viewer can_read=true")
	}
	if !summary[shared.RelationDocumentCanComment] {
		t.Error("expected viewer can_comment=true")
	}
	if summary[shared.RelationDocumentCanEdit] {
		t.Error("expected viewer can_edit=false")
	}
	if summary[shared.RelationDocumentCanDelete] {
		t.Error("expected viewer can_delete=false")
	}
}

func TestAllPermissions_NonDocumentObjectReturnsEmptySummary(t *testing.T) {
	ev := newEvaluator()

	summary, err := concurrency.AllPermissions(context.Background(), ev, fixtures.Alice, fixtures.ProductWorkspace)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summary) != 0 {
		t.Errorf("expected empty summary for workspace object, got %d entries", len(summary))
	}
}

func TestBulkCheck_ReturnsResultsInInputOrder(t *testing.T) {
	ev := newEvaluator()
	reqs := []shared.CheckRequest{
		{User: fixtures.Alice, Relation: shared.RelationDocumentCanEdit, Object: fixtures.RoadmapDocument},
		{User: fixtures.Bob, Relation: shared.RelationDocumentCanEdit, Object: fixtures.RoadmapDocument},
		{User: fixtures.Bob, Relation: shared.RelationDocumentCanRead, Object: fixtures.RoadmapDocument},
	}

	results := concurrency.BulkCheck(context.Background(), ev, reqs)

	if len(results) != len(reqs) {
		t.Fatalf("expected %d results, got %d", len(reqs), len(results))
	}
	wantAllowed := []bool{true, false, true}
	for i, want := range wantAllowed {
		if results[i].Err != nil {
			t.Errorf("results[%d].Err = %v, want nil", i, results[i].Err)
		}
		if results[i].Result.Allowed != want {
			t.Errorf("results[%d].Allowed = %v, want %v", i, results[i].Result.Allowed, want)
		}
		if results[i].Request != reqs[i] {
			t.Errorf("results[%d].Request = %+v, want %+v", i, results[i].Request, reqs[i])
		}
	}
}

func TestBulkCheck_EmptyInputReturnsEmptySlice(t *testing.T) {
	ev := newEvaluator()
	results := concurrency.BulkCheck(context.Background(), ev, nil)
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}
