// Package domain contains the documents service's core business logic.
// It depends only on ports — no knowledge of HTTP, databases, or authz implementation.
//
// Mirrors typescript/src/documents-service/core/domain/.
package domain

import (
	"fmt"

	"rebac-primer/internal/documentsservice/core/ports"
	"rebac-primer/internal/shared"
)

// CollaborativeDocument is a type alias for ports.CollaborativeDocument.
//
// Using an alias (=) rather than a new type means domain code can use
// CollaborativeDocument everywhere without any conversion — it is the same type
// as what the repository stores and what the HTTP layer serialises.
type CollaborativeDocument = ports.CollaborativeDocument

// ── Input types ───────────────────────────────────────────────────────────────

// CreateDocumentInput carries the data needed to create a new document.
type CreateDocumentInput struct {
	ID        string
	Title     string
	Body      string
	Workspace shared.Object
	Actor     shared.Object
}

// UpdateDocumentInput carries the data needed to update an existing document.
type UpdateDocumentInput struct {
	ID    string
	Body  string
	Actor shared.Object
}

// ── Error types ───────────────────────────────────────────────────────────────

// DocumentNotFoundError is returned when a document ID does not match any
// stored document.
type DocumentNotFoundError struct {
	ID string
}

func (e *DocumentNotFoundError) Error() string {
	return fmt.Sprintf("document not found: %s", e.ID)
}

// ForbiddenError is returned when an actor lacks the required permission.
type ForbiddenError struct {
	Message string
}

func (e *ForbiddenError) Error() string { return e.Message }
