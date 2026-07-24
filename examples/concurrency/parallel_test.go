package concurrency_test

import (
	"context"
	"errors"
	"testing"

	"rebac-primer/examples/concurrency"
	"rebac-primer/internal/authz"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/rebac"
)

// blockingEvaluator is a fake Checker whose Evaluate does no work until the
// context is cancelled, then reports the context error. It lets us exercise
// AllPermissions' cancellation path deterministically.
type blockingEvaluator struct{}

func (blockingEvaluator) Evaluate(ctx context.Context, _ rebac.CheckRequest) (rebac.CheckResult, error) {
	<-ctx.Done()
	return rebac.CheckResult{}, ctx.Err()
}

func TestAllPermissions_CancelledContextReturnsError(t *testing.T) {
	// Arrange: a context that is already cancelled, and an evaluator that only
	// unblocks once the context is done.
	ctx, cancel := context.WithCancel(t.Context())
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
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	ev := authz.NewGraphEvaluator(store)

	// Act
	summary, err := concurrency.AllPermissions(t.Context(), ev, fixtures.Alice, fixtures.RoadmapDocument)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[rebac.Permission]bool{
		rebac.PermissionDocumentRead:    true,
		rebac.PermissionDocumentComment: true,
		rebac.PermissionDocumentEdit:    true,
		rebac.PermissionDocumentDelete:  false,
	}
	for permission, expected := range want {
		if got := summary[permission]; got != expected {
			t.Errorf("summary[%s] = %v, want %v", permission, got, expected)
		}
	}
}

func TestAllPermissions_ViewerCanReadButNotEdit(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	ev := authz.NewGraphEvaluator(store)

	// Act
	summary, err := concurrency.AllPermissions(t.Context(), ev, fixtures.Bob, fixtures.RoadmapDocument)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !summary[rebac.PermissionDocumentRead] {
		t.Error("expected viewer can_read=true")
	}
	if !summary[rebac.PermissionDocumentComment] {
		t.Error("expected viewer can_comment=true")
	}
	if summary[rebac.PermissionDocumentEdit] {
		t.Error("expected viewer can_edit=false")
	}
	if summary[rebac.PermissionDocumentDelete] {
		t.Error("expected viewer can_delete=false")
	}
}

func TestAllPermissions_NonDocumentObjectReturnsEmptySummary(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	ev := authz.NewGraphEvaluator(store)

	// Act
	summary, err := concurrency.AllPermissions(t.Context(), ev, fixtures.Alice, fixtures.ProductWorkspace)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summary) != 0 {
		t.Errorf("expected empty summary for workspace resource, got %d entries", len(summary))
	}
}

func TestBulkCheck_ReturnsResultsInInputOrder(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	ev := authz.NewGraphEvaluator(store)
	reqs := []rebac.CheckRequest{
		{Subject: fixtures.Alice, Permission: rebac.PermissionDocumentEdit, Resource: fixtures.RoadmapDocument},
		{Subject: fixtures.Bob, Permission: rebac.PermissionDocumentEdit, Resource: fixtures.RoadmapDocument},
		{Subject: fixtures.Bob, Permission: rebac.PermissionDocumentRead, Resource: fixtures.RoadmapDocument},
	}

	// Act
	results := concurrency.BulkCheck(t.Context(), ev, reqs)

	// Assert
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
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	ev := authz.NewGraphEvaluator(store)

	// Act
	results := concurrency.BulkCheck(t.Context(), ev, nil)

	// Assert
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}
