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
	deleted []rebac.TupleKey
}

func (f *failingWriteAuthz) Check(context.Context, rebac.CheckRequest) (rebac.CheckResult, error) {
	return rebac.CheckResult{Allowed: true}, nil
}

func (f *failingWriteAuthz) WriteTuples(context.Context, []rebac.TupleKey) error {
	return errors.New("tuple write failed")
}

func (f *failingWriteAuthz) DeleteTuples(_ context.Context, tuples []rebac.TupleKey) error {
	f.deleted = append(f.deleted, tuples...)
	return nil
}

// newSeededService wires together a *documents.Service backed by the standard
// fixture store and pre-creates the roadmap document so tests can read/update it.
func newSeededService(t *testing.T) *documents.Service {
	t.Helper()

	// Authz service wired in-process (no HTTP hop in tests)
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	evaluator := authz.NewGraphEvaluator(store)
	authzSvc := authz.New(store, evaluator)

	// Documents service
	repo := documents.NewInMemoryRepository()
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

func TestDocumentService_Create_RejectsExistingID(t *testing.T) {
	svc := newSeededService(t)

	// alice is a workspace editor, so the authorization check passes — the create
	// must still fail because the ID is taken. Allowing it would overwrite the
	// document and grant alice a fresh owner tuple on it.
	_, err := svc.Create(context.Background(), documents.CreateDocumentInput{
		ID:        "roadmapDocument",
		Title:     "Hijack",
		Body:      "overwritten",
		Workspace: fixtures.ProductWorkspace,
		Actor:     fixtures.Alice,
	})

	var alreadyExists *documents.DocumentAlreadyExistsError
	if !errors.As(err, &alreadyExists) {
		t.Fatalf("expected *DocumentAlreadyExistsError, got %T: %v", err, err)
	}

	// The stored document is untouched.
	doc, err := svc.Read(context.Background(), "roadmapDocument", fixtures.Alice)
	if err != nil {
		t.Fatalf("read after rejected create: %v", err)
	}
	if doc.Body != "Initial roadmap document" {
		t.Errorf("expected original body to survive, got %q", doc.Body)
	}
}

func TestDocumentService_Create_MakesCreatorOwner(t *testing.T) {
	// Arrange: wire authz + documents over a shared tuple store so we can inspect
	// the tuples Create writes.
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	authzSvc := authz.New(store, authz.NewGraphEvaluator(store))
	svc := documents.New(documents.NewInMemoryRepository(), authzSvc)

	// Act: alice (a workspace editor) creates a document.
	if _, err := svc.Create(context.Background(), documents.CreateDocumentInput{
		ID: "d1", Title: "Roadmap", Body: "v1",
		Workspace: fixtures.ProductWorkspace, Actor: fixtures.Alice,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Assert: alice can_delete d1. can_delete requires document owner, and a
	// workspace editor only inherits document editor (can_edit) — never owner. So
	// this passes only because Create wrote a direct (d1, owner, alice) tuple.
	ownerCheck, err := authzSvc.Check(context.Background(), rebac.CheckRequest{
		User: fixtures.Alice, Relation: rebac.RelationDocumentCanDelete, Object: rebac.Document("d1"),
	})
	if err != nil {
		t.Fatalf("check alice: %v", err)
	}
	if !ownerCheck.Allowed {
		t.Error("expected creator alice to have can_delete on the document she created")
	}

	// And bob (a workspace viewer) is not an owner — cannot delete.
	viewerCheck, err := authzSvc.Check(context.Background(), rebac.CheckRequest{
		User: fixtures.Bob, Relation: rebac.RelationDocumentCanDelete, Object: rebac.Document("d1"),
	})
	if err != nil {
		t.Fatalf("check bob: %v", err)
	}
	if viewerCheck.Allowed {
		t.Error("expected workspace viewer bob to NOT have can_delete")
	}
}

func TestDocumentService_Create_WhenTupleWriteFails_RollsBackDocument(t *testing.T) {
	repo := documents.NewInMemoryRepository()
	authzClient := &failingWriteAuthz{}
	svc := documents.New(repo, authzClient)

	_, err := svc.Create(context.Background(), documents.CreateDocumentInput{
		ID: "d1", Title: "Roadmap", Body: "v1",
		Workspace: fixtures.ProductWorkspace, Actor: fixtures.Alice,
	})
	if err == nil {
		t.Fatal("expected tuple write error")
	}

	doc, findErr := repo.FindByID(context.Background(), "d1")
	if findErr != nil {
		t.Fatalf("FindByID returned error: %v", findErr)
	}
	if doc != nil {
		t.Errorf("document still exists after rollback: %+v", doc)
	}
	if len(authzClient.deleted) != 2 {
		t.Errorf("deleted %d tuples during rollback, want 2", len(authzClient.deleted))
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
