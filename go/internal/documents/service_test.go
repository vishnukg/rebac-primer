package documents_test

import (
	"context"
	"errors"
	"testing"

	authzdb "rebac-primer/internal/authz/adapters/db"
	"rebac-primer/internal/authz/adapters/graph"
	"rebac-primer/internal/authz"
	docsdb "rebac-primer/internal/documents/adapters/db"
	"rebac-primer/internal/documents"
	"rebac-primer/internal/fixtures"
)

// newSeededService wires together a documents.Service backed by the standard
// fixture store and pre-creates the roadmap document so tests can read/update it.
func newSeededService(t *testing.T) documents.Service {
	t.Helper()

	// Authz service wired in-process (no HTTP hop in tests)
	store := authzdb.New(fixtures.SeedRelationshipTuples()...)
	evaluator := graph.NewGraphEvaluator(store)
	authzSvc := authz.New(store, evaluator)

	// Documents service
	repo := docsdb.New()
	svc := documents.New(repo, authzSvc)

	_, err := svc.Create(context.Background(), documents.CreateDocumentInput{
		ID:        "roadmapDocument",
		Title:     "Roadmap",
		Body:      "Initial roadmap document",
		Workspace: fixtures.ProductWorkspace,
		Actor:     fixtures.Alice,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return svc
}

func TestDocumentService_Create_ForbiddenForViewer(t *testing.T) {
	svc := newSeededService(t)
	input := documents.CreateDocumentInput{
		ID:        "newDoc",
		Title:     "New",
		Body:      "body",
		Workspace: fixtures.ProductWorkspace,
		Actor:     fixtures.Bob,
	}

	_, err := svc.Create(context.Background(), input)

	if err == nil {
		t.Fatal("expected ForbiddenError but got nil")
	}
	var forbidden *documents.ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Errorf("expected *ForbiddenError, got %T: %v", err, err)
	}
}

func TestDocumentService_Create_SucceedsForEditor(t *testing.T) {
	svc := newSeededService(t)
	input := documents.CreateDocumentInput{
		ID:        "anotherDoc",
		Title:     "Another",
		Body:      "content",
		Workspace: fixtures.ProductWorkspace,
		Actor:     fixtures.Alice,
	}

	doc, err := svc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.ID != "anotherDoc" {
		t.Errorf("expected id=%q, got %q", "anotherDoc", doc.ID)
	}
	if doc.UpdatedBy != fixtures.Alice {
		t.Errorf("expected updatedBy=%q, got %q", fixtures.Alice, doc.UpdatedBy)
	}
}

func TestDocumentService_Read_ForbiddenForOutsider(t *testing.T) {
	svc := newSeededService(t)

	_, err := svc.Read(context.Background(), "roadmapDocument", fixtures.Casey)
	if err == nil {
		t.Fatal("expected ForbiddenError but got nil")
	}
	var forbidden *documents.ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Errorf("expected *ForbiddenError, got %T: %v", err, err)
	}
}

func TestDocumentService_Read_NotFoundBeforeAuthCheck(t *testing.T) {
	svc := newSeededService(t)

	_, err := svc.Read(context.Background(), "nonexistent", fixtures.Alice)
	if err == nil {
		t.Fatal("expected DocumentNotFoundError but got nil")
	}
	var notFound *documents.DocumentNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected *DocumentNotFoundError, got %T: %v", err, err)
	}
}

func TestDocumentService_Read_SucceedsForViewer(t *testing.T) {
	svc := newSeededService(t)

	doc, err := svc.Read(context.Background(), "roadmapDocument", fixtures.Bob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.ID != "roadmapDocument" {
		t.Errorf("expected id=%q, got %q", "roadmapDocument", doc.ID)
	}
}

func TestDocumentService_Update_ForbiddenForViewer(t *testing.T) {
	svc := newSeededService(t)
	input := documents.UpdateDocumentInput{
		ID:    "roadmapDocument",
		Body:  "should not save",
		Actor: fixtures.Bob,
	}

	_, err := svc.Update(context.Background(), input)
	if err == nil {
		t.Fatal("expected ForbiddenError but got nil")
	}
	var forbidden *documents.ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Errorf("expected *ForbiddenError, got %T: %v", err, err)
	}
}

func TestDocumentService_Update_SucceedsForEditor(t *testing.T) {
	svc := newSeededService(t)
	input := documents.UpdateDocumentInput{
		ID:    "roadmapDocument",
		Body:  "updated content",
		Actor: fixtures.Alice,
	}

	updated, err := svc.Update(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Body != "updated content" {
		t.Errorf("expected body=%q, got %q", "updated content", updated.Body)
	}
	if updated.UpdatedBy != fixtures.Alice {
		t.Errorf("expected updatedBy=%q, got %q", fixtures.Alice, updated.UpdatedBy)
	}
}
