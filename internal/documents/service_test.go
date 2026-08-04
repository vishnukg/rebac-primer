package documents_test

import (
	"context"
	"errors"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/documents"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/rebac"
)

type failingWriteAuthz struct {
	deleted []rebac.Relationship
}

func (f *failingWriteAuthz) Check(context.Context, rebac.CheckRequest) (rebac.CheckResult, error) {
	return rebac.CheckResult{Allowed: true}, nil
}

func (f *failingWriteAuthz) WriteRelationships(context.Context, []rebac.Relationship) error {
	return errors.New("relationship write failed")
}

func (f *failingWriteAuthz) DeleteRelationships(_ context.Context, relationships []rebac.Relationship) error {
	f.deleted = append(f.deleted, relationships...)
	return nil
}

func TestDocumentService_Create_ForbiddenForViewer(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	authzSvc := authz.New(store, authz.NewGraphEvaluator(store))
	svc := documents.New(documents.NewInMemoryRepository(), authzSvc)
	input := documents.CreateDocumentInput{
		ID:        "newDoc",
		Title:     "New",
		Body:      "body",
		Workspace: fixtures.ProductWorkspace,
		Subject:   fixtures.Bob,
	}

	// Act
	_, err := svc.Create(t.Context(), input)

	// Assert
	if err == nil {
		t.Fatal("expected ForbiddenError but got nil")
	}
	var forbidden *documents.ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Errorf("expected *ForbiddenError, got %T: %v", err, err)
	}
}

func TestDocumentService_Create_SucceedsForEditor(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	authzSvc := authz.New(store, authz.NewGraphEvaluator(store))
	svc := documents.New(documents.NewInMemoryRepository(), authzSvc)
	input := documents.CreateDocumentInput{
		ID:        "anotherDoc",
		Title:     "Another",
		Body:      "content",
		Workspace: fixtures.ProductWorkspace,
		Subject:   fixtures.Alice,
	}

	// Act
	doc, err := svc.Create(t.Context(), input)

	// Assert
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

func TestDocumentService_Create_RejectsExistingID(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	authzSvc := authz.New(store, authz.NewGraphEvaluator(store))
	repo := documents.NewInMemoryRepository()
	if err := repo.Create(t.Context(), documents.CollaborativeDocument{
		ID: "roadmapDocument", Title: "Roadmap", Body: "Initial roadmap document",
		Workspace: fixtures.ProductWorkspace, UpdatedBy: fixtures.Alice,
	}); err != nil {
		t.Fatalf("seed repository: %v", err)
	}
	svc := documents.New(repo, authzSvc)

	// alice is a workspace editor, so the authorization check passes — the create
	// must still fail because the ID is taken. Allowing it would overwrite the
	// document and grant alice a fresh owner relationship on it.
	// Act
	_, err := svc.Create(t.Context(), documents.CreateDocumentInput{
		ID:        "roadmapDocument",
		Title:     "Hijack",
		Body:      "overwritten",
		Workspace: fixtures.ProductWorkspace,
		Subject:   fixtures.Alice,
	})

	// Assert
	var alreadyExists *documents.DocumentAlreadyExistsError
	if !errors.As(err, &alreadyExists) {
		t.Fatalf("expected *DocumentAlreadyExistsError, got %T: %v", err, err)
	}

	// The stored document is untouched.
	doc, err := svc.Read(t.Context(), "roadmapDocument", fixtures.Alice)
	if err != nil {
		t.Fatalf("read after rejected create: %v", err)
	}
	if doc.Body != "Initial roadmap document" {
		t.Errorf("expected original body to survive, got %q", doc.Body)
	}
}

func TestDocumentService_Create_MakesCreatorOwner(t *testing.T) {
	// Arrange: wire authz + documents over a shared relationship store so we can inspect
	// the relationships Create writes.
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	authzSvc := authz.New(store, authz.NewGraphEvaluator(store))
	svc := documents.New(documents.NewInMemoryRepository(), authzSvc)

	// Act: alice (a workspace editor) creates a document.
	if _, err := svc.Create(t.Context(), documents.CreateDocumentInput{
		ID: "d1", Title: "Roadmap", Body: "v1",
		Workspace: fixtures.ProductWorkspace, Subject: fixtures.Alice,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Assert: alice can_delete d1. can_delete requires document owner, and a
	// workspace editor only inherits document editor (can_edit) — never owner. So
	// this passes only because Create wrote a direct (alice, owner, d1) relationship.
	ownerCheck, err := authzSvc.Check(t.Context(), rebac.CheckRequest{
		Subject: fixtures.Alice, Action: rebac.ActionDocumentDelete, Resource: rebac.Document("d1"),
	})
	if err != nil {
		t.Fatalf("check alice: %v", err)
	}
	if !ownerCheck.Allowed {
		t.Error("expected creator alice to have can_delete on the document she created")
	}

	// And bob (a workspace viewer) is not an owner — cannot delete.
	viewerCheck, err := authzSvc.Check(t.Context(), rebac.CheckRequest{
		Subject: fixtures.Bob, Action: rebac.ActionDocumentDelete, Resource: rebac.Document("d1"),
	})
	if err != nil {
		t.Fatalf("check bob: %v", err)
	}
	if viewerCheck.Allowed {
		t.Error("expected workspace viewer bob to NOT have can_delete")
	}
}

func TestDocumentService_Create_WhenRelationshipWriteFails_RollsBackDocument(t *testing.T) {
	// Arrange
	repo := documents.NewInMemoryRepository()
	authzClient := &failingWriteAuthz{}
	svc := documents.New(repo, authzClient)

	// Act
	_, err := svc.Create(t.Context(), documents.CreateDocumentInput{
		ID: "d1", Title: "Roadmap", Body: "v1",
		Workspace: fixtures.ProductWorkspace, Subject: fixtures.Alice,
	})
	// Assert
	if err == nil {
		t.Fatal("expected relationship write error")
	}

	doc, findErr := repo.FindByID(t.Context(), "d1")
	if findErr != nil {
		t.Fatalf("FindByID returned error: %v", findErr)
	}
	if doc != nil {
		t.Errorf("document still exists after rollback: %+v", doc)
	}
	if len(authzClient.deleted) != 2 {
		t.Errorf("deleted %d relationships during rollback, want 2", len(authzClient.deleted))
	}
}

func TestDocumentService_Read_ForbiddenForOutsider(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	authzSvc := authz.New(store, authz.NewGraphEvaluator(store))
	repo := documents.NewInMemoryRepository()
	if err := repo.Create(t.Context(), documents.CollaborativeDocument{ID: "roadmapDocument", Title: "Roadmap", Body: "Initial roadmap document", Workspace: fixtures.ProductWorkspace, UpdatedBy: fixtures.Alice}); err != nil {
		t.Fatalf("seed repository: %v", err)
	}
	svc := documents.New(repo, authzSvc)

	// Act
	_, err := svc.Read(t.Context(), "roadmapDocument", fixtures.Casey)

	// Assert
	if err == nil {
		t.Fatal("expected ForbiddenError but got nil")
	}
	var forbidden *documents.ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Errorf("expected *ForbiddenError, got %T: %v", err, err)
	}
}

func TestDocumentService_Read_NotFoundBeforeAuthCheck(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	authzSvc := authz.New(store, authz.NewGraphEvaluator(store))
	svc := documents.New(documents.NewInMemoryRepository(), authzSvc)

	// Act
	_, err := svc.Read(t.Context(), "nonexistent", fixtures.Alice)

	// Assert
	if err == nil {
		t.Fatal("expected DocumentNotFoundError but got nil")
	}
	var notFound *documents.DocumentNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected *DocumentNotFoundError, got %T: %v", err, err)
	}
}

func TestDocumentService_Read_SucceedsForViewer(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	authzSvc := authz.New(store, authz.NewGraphEvaluator(store))
	repo := documents.NewInMemoryRepository()
	if err := repo.Create(t.Context(), documents.CollaborativeDocument{ID: "roadmapDocument", Title: "Roadmap", Body: "Initial roadmap document", Workspace: fixtures.ProductWorkspace, UpdatedBy: fixtures.Alice}); err != nil {
		t.Fatalf("seed repository: %v", err)
	}
	svc := documents.New(repo, authzSvc)

	// Act
	doc, err := svc.Read(t.Context(), "roadmapDocument", fixtures.Bob)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.ID != "roadmapDocument" {
		t.Errorf("expected id=%q, got %q", "roadmapDocument", doc.ID)
	}
}

func TestDocumentService_Update_ForbiddenForViewer(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	authzSvc := authz.New(store, authz.NewGraphEvaluator(store))
	repo := documents.NewInMemoryRepository()
	if err := repo.Create(t.Context(), documents.CollaborativeDocument{ID: "roadmapDocument", Title: "Roadmap", Body: "Initial roadmap document", Workspace: fixtures.ProductWorkspace, UpdatedBy: fixtures.Alice}); err != nil {
		t.Fatalf("seed repository: %v", err)
	}
	svc := documents.New(repo, authzSvc)
	input := documents.UpdateDocumentInput{
		ID:      "roadmapDocument",
		Body:    "should not save",
		Subject: fixtures.Bob,
	}

	// Act
	_, err := svc.Update(t.Context(), input)

	// Assert
	if err == nil {
		t.Fatal("expected ForbiddenError but got nil")
	}
	var forbidden *documents.ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Errorf("expected *ForbiddenError, got %T: %v", err, err)
	}
}

func TestDocumentService_Update_SucceedsForEditor(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	authzSvc := authz.New(store, authz.NewGraphEvaluator(store))
	repo := documents.NewInMemoryRepository()
	if err := repo.Create(t.Context(), documents.CollaborativeDocument{ID: "roadmapDocument", Title: "Roadmap", Body: "Initial roadmap document", Workspace: fixtures.ProductWorkspace, UpdatedBy: fixtures.Alice}); err != nil {
		t.Fatalf("seed repository: %v", err)
	}
	svc := documents.New(repo, authzSvc)
	input := documents.UpdateDocumentInput{
		ID:      "roadmapDocument",
		Body:    "updated content",
		Subject: fixtures.Alice,
	}

	// Act
	updated, err := svc.Update(t.Context(), input)

	// Assert
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
