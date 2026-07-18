package documents_test

import (
	"errors"
	"testing"

	"rebac-primer/internal/documents"
	"rebac-primer/internal/rebac"
)

// These tests cover the in-memory DocumentRepository adapter. It is a
// self-contained stateful unit, so no test doubles are needed.

func TestRepository_GivenSavedDocument_WhenFoundByID_ThenReturnsIt(t *testing.T) {
	// Arrange
	repo := documents.NewInMemoryRepository()
	doc := documents.CollaborativeDocument{ID: "roadmapDocument", Title: "Roadmap", Body: "v1", Workspace: rebac.Workspace("productWorkspace"), UpdatedBy: rebac.User("alice")}
	if err := repo.Save(t.Context(), doc); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	// Act
	got, err := repo.FindByID(t.Context(), doc.ID)

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
	got, err := repo.FindByID(t.Context(), "doesNotExist")

	// Assert: a miss is (nil, nil), not an error.
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("FindByID = %+v, want nil", got)
	}
}

func TestRepository_GivenExistingID_WhenCreatedAgain_ThenRejectsWithoutOverwrite(t *testing.T) {
	// Arrange
	repo := documents.NewInMemoryRepository()
	original := documents.CollaborativeDocument{ID: "roadmapDocument", Title: "Roadmap", Body: "v1", Workspace: rebac.Workspace("productWorkspace"), UpdatedBy: rebac.User("alice")}
	if err := repo.Create(t.Context(), original); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	replacement := original
	replacement.Body = "replacement"

	// Act
	err := repo.Create(t.Context(), replacement)

	// Assert
	var alreadyExists *documents.DocumentAlreadyExistsError
	if !errors.As(err, &alreadyExists) {
		t.Fatalf("expected *DocumentAlreadyExistsError, got %v", err)
	}
	got, err := repo.FindByID(t.Context(), original.ID)
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID = nil, want original document")
	}
	if got.Body != original.Body {
		t.Errorf("stored body = %q, want original %q", got.Body, original.Body)
	}
}

func TestRepository_GivenSameIDSavedTwice_WhenFoundByID_ThenReturnsLatest(t *testing.T) {
	// Arrange
	repo := documents.NewInMemoryRepository()
	doc := documents.CollaborativeDocument{ID: "roadmapDocument", Title: "Roadmap", Body: "v1", Workspace: rebac.Workspace("productWorkspace"), UpdatedBy: rebac.User("alice")}
	if err := repo.Save(t.Context(), doc); err != nil {
		t.Fatalf("Save (first) returned unexpected error: %v", err)
	}

	// Act: save again under the same ID with new content.
	updated := doc
	updated.Body = "v2"
	if err := repo.Save(t.Context(), updated); err != nil {
		t.Fatalf("Save (second) returned unexpected error: %v", err)
	}

	// Assert
	got, err := repo.FindByID(t.Context(), doc.ID)
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if got == nil || got.Body != "v2" {
		t.Errorf("FindByID body = %v, want v2", got)
	}
}

func TestRepository_GivenCallerMutatesInputAfterSave_WhenFoundByID_ThenStoredCopyUnchanged(t *testing.T) {
	// Arrange
	repo := documents.NewInMemoryRepository()
	doc := documents.CollaborativeDocument{ID: "roadmapDocument", Title: "Roadmap", Body: "v1", Workspace: rebac.Workspace("productWorkspace"), UpdatedBy: rebac.User("alice")}
	if err := repo.Save(t.Context(), doc); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	// Act: mutate the caller's value after saving — snapshot semantics mean the
	// store keeps its own copy.
	doc.Body = "mutated by caller"

	// Assert
	got, err := repo.FindByID(t.Context(), doc.ID)
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if got == nil || got.Body != "v1" {
		t.Errorf("stored body = %v, want v1 (Save must snapshot its input)", got)
	}
}

func TestRepository_GivenCallerMutatesReturnedValue_WhenFoundByIDAgain_ThenStoredCopyUnchanged(t *testing.T) {
	// Arrange
	repo := documents.NewInMemoryRepository()
	doc := documents.CollaborativeDocument{ID: "roadmapDocument", Title: "Roadmap", Body: "v1", Workspace: rebac.Workspace("productWorkspace"), UpdatedBy: rebac.User("alice")}
	if err := repo.Save(t.Context(), doc); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	// Act: mutate the value handed back by FindByID.
	first, err := repo.FindByID(t.Context(), doc.ID)
	if err != nil {
		t.Fatalf("first FindByID returned unexpected error: %v", err)
	}
	if first == nil {
		t.Fatal("first FindByID = nil, want saved document")
	}
	first.Body = "mutated via returned pointer"

	// Assert: a fresh read is unaffected.
	second, err := repo.FindByID(t.Context(), doc.ID)
	if err != nil {
		t.Fatalf("second FindByID returned unexpected error: %v", err)
	}
	if second == nil || second.Body != "v1" {
		t.Errorf("stored body = %v, want v1 (FindByID must return a copy)", second)
	}
}
