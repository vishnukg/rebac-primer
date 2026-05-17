package authz_test

import (
	"context"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/fixtures"
)

func TestAllPermissions_ReturnsFullSummaryForEditor(t *testing.T) {
	// Arrange: alice can edit (and therefore read and comment) the
	// roadmap document. Only can_delete should be denied because editor ≠ owner.
	store := seedStore()
	auth := authz.NewGraphAuthorizer(store)

	// Act
	summary, err := authz.AllPermissions(context.Background(), auth, fixtures.Alice, fixtures.RoadmapDocument)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[authz.Relation]bool{
		authz.RelationDocumentCanRead:    true,
		authz.RelationDocumentCanComment: true,
		authz.RelationDocumentCanEdit:    true,
		authz.RelationDocumentCanDelete:  false,
	}
	for rel, expected := range want {
		if got := summary[rel]; got != expected {
			t.Errorf("summary[%s] = %v, want %v", rel, got, expected)
		}
	}
}

func TestAllPermissions_ViewerCanReadButNotEdit(t *testing.T) {
	// Arrange: bob has viewer access only.
	store := seedStore()
	auth := authz.NewGraphAuthorizer(store)

	// Act
	summary, err := authz.AllPermissions(context.Background(), auth, fixtures.Bob, fixtures.RoadmapDocument)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !summary[authz.RelationDocumentCanRead] {
		t.Error("expected viewer can_read=true")
	}
	if !summary[authz.RelationDocumentCanComment] {
		t.Error("expected viewer can_comment=true")
	}
	if summary[authz.RelationDocumentCanEdit] {
		t.Error("expected viewer can_edit=false")
	}
	if summary[authz.RelationDocumentCanDelete] {
		t.Error("expected viewer can_delete=false")
	}
}

func TestAllPermissions_NonDocumentObjectReturnsEmptySummary(t *testing.T) {
	// Arrange: AllPermissions only knows how to enumerate permissions for
	// documents. A workspace object has no computed permissions defined.
	store := seedStore()
	auth := authz.NewGraphAuthorizer(store)

	// Act
	summary, err := authz.AllPermissions(context.Background(), auth, fixtures.Alice, fixtures.ProductWorkspace)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summary) != 0 {
		t.Errorf("expected empty summary for workspace object, got %d entries", len(summary))
	}
}

func TestBulkCheck_ReturnsResultsInInputOrder(t *testing.T) {
	// Arrange: three requests in a specific order. BulkCheck must preserve order
	// even though goroutines finish in an unpredictable sequence.
	store := seedStore()
	auth := authz.NewGraphAuthorizer(store)
	reqs := []authz.CheckRequest{
		{User: fixtures.Alice, Relation: authz.RelationDocumentCanEdit, Object: fixtures.RoadmapDocument},
		{User: fixtures.Bob, Relation: authz.RelationDocumentCanEdit, Object: fixtures.RoadmapDocument},
		{User: fixtures.Bob, Relation: authz.RelationDocumentCanRead, Object: fixtures.RoadmapDocument},
	}

	// Act
	results := authz.BulkCheck(context.Background(), auth, reqs)

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
			t.Errorf("results[%d].Allowed = %v, want %v (req: %+v)", i, results[i].Result.Allowed, want, reqs[i])
		}
		// The Request field must match the input so callers can correlate results.
		if results[i].Request != reqs[i] {
			t.Errorf("results[%d].Request = %+v, want %+v", i, results[i].Request, reqs[i])
		}
	}
}

func TestBulkCheck_EmptyInputReturnsEmptySlice(t *testing.T) {
	// Arrange
	store := seedStore()
	auth := authz.NewGraphAuthorizer(store)

	// Act
	results := authz.BulkCheck(context.Background(), auth, nil)

	// Assert
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}
