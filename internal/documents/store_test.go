package documents_test

import (
	"context"
	"errors"
	"testing"

	"rebac-primer/internal/documents"
	"rebac-primer/internal/rebac"
)

// These tests cover the in-memory DocumentRepository adapter. It is a
// self-contained stateful unit, so no test doubles are needed.

func sampleDoc() documents.CollaborativeDocument {
	return documents.CollaborativeDocument{
		ID:        "roadmapDocument",
		Title:     "Roadmap",
		Body:      "v1",
		Workspace: rebac.Workspace("productWorkspace"),
		UpdatedBy: rebac.User("alice"),
	}
}

func TestRepository_GivenSavedDocument_WhenFoundByID_ThenReturnsIt(t *testing.T) {
	// Arrange
	repo := documents.NewInMemoryRepository()
	doc := sampleDoc()
	if err := repo.Save(context.Background(), doc); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	// Act
	got, err := repo.FindByID(context.Background(), doc.ID)

	// Assert
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if got == nil {
		t.Fatalf("FindByID = nil, want the saved document")
	}
	if *got != doc {
		t.Errorf("FindByID = %+v, want %+v", *got, doc)
	}
}

func TestRepository_GivenUnknownID_WhenFoundByID_ThenReturnsNilWithoutError(t *testing.T) {
	// Arrange
	repo := documents.NewInMemoryRepository()

	// Act
	got, err := repo.FindByID(context.Background(), "doesNotExist")

	// Assert: a miss is (nil, nil), not an error.
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("FindByID = %+v, want nil", got)
	}
}

func TestRepository_GivenExistingID_WhenCreatedAgain_ThenRejectsWithoutOverwrite(t *testing.T) {
	repo := documents.NewInMemoryRepository()
	original := sampleDoc()
	if err := repo.Create(context.Background(), original); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	replacement := original
	replacement.Body = "replacement"

	err := repo.Create(context.Background(), replacement)

	var alreadyExists *documents.DocumentAlreadyExistsError
	if !errors.As(err, &alreadyExists) {
		t.Fatalf("expected *DocumentAlreadyExistsError, got %v", err)
	}
	got, _ := repo.FindByID(context.Background(), original.ID)
	if got.Body != original.Body {
		t.Errorf("stored body = %q, want original %q", got.Body, original.Body)
	}
}

func TestRepository_GivenSameIDSavedTwice_WhenFoundByID_ThenReturnsLatest(t *testing.T) {
	// Arrange
	repo := documents.NewInMemoryRepository()
	doc := sampleDoc()
	if err := repo.Save(context.Background(), doc); err != nil {
		t.Fatalf("Save (first) returned unexpected error: %v", err)
	}

	// Act: save again under the same ID with new content.
	updated := doc
	updated.Body = "v2"
	if err := repo.Save(context.Background(), updated); err != nil {
		t.Fatalf("Save (second) returned unexpected error: %v", err)
	}

	// Assert
	got, _ := repo.FindByID(context.Background(), doc.ID)
	if got == nil || got.Body != "v2" {
		t.Errorf("FindByID body = %v, want v2", got)
	}
}

func TestRepository_GivenCallerMutatesInputAfterSave_WhenFoundByID_ThenStoredCopyUnchanged(t *testing.T) {
	// Arrange
	repo := documents.NewInMemoryRepository()
	doc := sampleDoc()
	if err := repo.Save(context.Background(), doc); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	// Act: mutate the caller's value after saving — snapshot semantics mean the
	// store keeps its own copy.
	doc.Body = "mutated by caller"

	// Assert
	got, _ := repo.FindByID(context.Background(), doc.ID)
	if got == nil || got.Body != "v1" {
		t.Errorf("stored body = %v, want v1 (Save must snapshot its input)", got)
	}
}

func TestRepository_GivenCallerMutatesReturnedValue_WhenFoundByIDAgain_ThenStoredCopyUnchanged(t *testing.T) {
	// Arrange
	repo := documents.NewInMemoryRepository()
	doc := sampleDoc()
	if err := repo.Save(context.Background(), doc); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	// Act: mutate the value handed back by FindByID.
	first, _ := repo.FindByID(context.Background(), doc.ID)
	first.Body = "mutated via returned pointer"

	// Assert: a fresh read is unaffected.
	second, _ := repo.FindByID(context.Background(), doc.ID)
	if second == nil || second.Body != "v1" {
		t.Errorf("stored body = %v, want v1 (FindByID must return a copy)", second)
	}
}
