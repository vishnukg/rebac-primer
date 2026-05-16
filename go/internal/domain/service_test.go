package domain_test

import (
	"context"
	"errors"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/domain"
	"rebac-primer/internal/fixtures"
)

// newSeededService wires together a DocumentService backed by the standard
// fixture store and pre-creates the roadmap document so tests can read/update it.
func newSeededService(t *testing.T) domain.DocumentOperations {
	t.Helper()

	store := authz.NewInMemoryTupleStore(fixtures.SeedRelationshipTuples()...)
	auth := authz.NewGraphAuthorizer(store)
	repo := domain.NewInMemoryDocumentRepository()
	svc := domain.NewDocumentService(repo, auth)

	_, err := svc.Create(context.Background(), domain.CreateDocumentInput{
		ID:        "roadmapDocument",
		Title:     "Roadmap",
		Body:      "Initial roadmap document",
		Workspace: fixtures.ProductWorkspace,
		Actor:     fixtures.WorkspaceEditor,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return svc
}

func TestDocumentService_Create_ForbiddenForViewer(t *testing.T) {
	// Arrange: workspaceViewer only has viewer on productWorkspace, not editor.
	svc := newSeededService(t)
	input := domain.CreateDocumentInput{
		ID:        "newDoc",
		Title:     "New",
		Body:      "body",
		Workspace: fixtures.ProductWorkspace,
		Actor:     fixtures.WorkspaceViewer,
	}

	// Act
	_, err := svc.Create(context.Background(), input)

	// Assert
	if err == nil {
		t.Fatal("expected ForbiddenError but got nil")
	}
	var forbidden *domain.ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Errorf("expected *ForbiddenError, got %T: %v", err, err)
	}
}

func TestDocumentService_Create_SucceedsForEditor(t *testing.T) {
	// Arrange
	svc := newSeededService(t)
	input := domain.CreateDocumentInput{
		ID:        "anotherDoc",
		Title:     "Another",
		Body:      "content",
		Workspace: fixtures.ProductWorkspace,
		Actor:     fixtures.WorkspaceEditor,
	}

	// Act
	doc, err := svc.Create(context.Background(), input)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.ID != "anotherDoc" {
		t.Errorf("expected id=%q, got %q", "anotherDoc", doc.ID)
	}
	if doc.UpdatedBy != fixtures.WorkspaceEditor {
		t.Errorf("expected updatedBy=%q, got %q", fixtures.WorkspaceEditor, doc.UpdatedBy)
	}
}

func TestDocumentService_Read_ForbiddenForOutsider(t *testing.T) {
	// Arrange: outsideCollaborator has no tuples in the graph.
	svc := newSeededService(t)

	// Act
	_, err := svc.Read(context.Background(), "roadmapDocument", fixtures.OutsideCollaborator)

	// Assert
	if err == nil {
		t.Fatal("expected ForbiddenError but got nil")
	}
	var forbidden *domain.ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Errorf("expected *ForbiddenError, got %T: %v", err, err)
	}
}

func TestDocumentService_Read_NotFoundBeforeAuthCheck(t *testing.T) {
	// Arrange: workspaceEditor has permission but the document does not exist.
	// The service must check existence first and return not-found, not forbidden.
	svc := newSeededService(t)

	// Act
	_, err := svc.Read(context.Background(), "nonexistent", fixtures.WorkspaceEditor)

	// Assert
	if err == nil {
		t.Fatal("expected DocumentNotFoundError but got nil")
	}
	var notFound *domain.DocumentNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected *DocumentNotFoundError, got %T: %v", err, err)
	}
}

func TestDocumentService_Read_SucceedsForViewer(t *testing.T) {
	// Arrange: workspaceViewer has viewer on productWorkspace → can_read on roadmapDocument.
	svc := newSeededService(t)

	// Act
	doc, err := svc.Read(context.Background(), "roadmapDocument", fixtures.WorkspaceViewer)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.ID != "roadmapDocument" {
		t.Errorf("expected id=%q, got %q", "roadmapDocument", doc.ID)
	}
}

func TestDocumentService_Update_ForbiddenForViewer(t *testing.T) {
	// Arrange: workspaceViewer has viewer, not editor — update must be denied.
	svc := newSeededService(t)
	input := domain.UpdateDocumentInput{
		ID:    "roadmapDocument",
		Body:  "should not save",
		Actor: fixtures.WorkspaceViewer,
	}

	// Act
	_, err := svc.Update(context.Background(), input)

	// Assert
	if err == nil {
		t.Fatal("expected ForbiddenError but got nil")
	}
	var forbidden *domain.ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Errorf("expected *ForbiddenError, got %T: %v", err, err)
	}
}

func TestDocumentService_Update_SucceedsForEditor(t *testing.T) {
	// Arrange
	svc := newSeededService(t)
	input := domain.UpdateDocumentInput{
		ID:    "roadmapDocument",
		Body:  "updated content",
		Actor: fixtures.WorkspaceEditor,
	}

	// Act
	updated, err := svc.Update(context.Background(), input)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Body != "updated content" {
		t.Errorf("expected body=%q, got %q", "updated content", updated.Body)
	}
	if updated.UpdatedBy != fixtures.WorkspaceEditor {
		t.Errorf("expected updatedBy=%q, got %q", fixtures.WorkspaceEditor, updated.UpdatedBy)
	}
}
