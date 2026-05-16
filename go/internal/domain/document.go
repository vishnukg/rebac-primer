// Package domain contains the core business types and errors for the
// collaborative documents workspace. It has no dependency on HTTP, the database,
// or the authorization infrastructure — only on the authz.Authorizer interface.
package domain

import (
	"fmt"

	"rebac-primer/internal/authz"
)

// CollaborativeDocument is the aggregate root for the documents domain.
type CollaborativeDocument struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	Body      string       `json:"body"`
	Workspace authz.Object `json:"workspace"`
	UpdatedBy authz.Object `json:"updatedBy"`
}

// CreateDocumentInput carries the data needed to create a new document.
type CreateDocumentInput struct {
	ID        string
	Title     string
	Body      string
	Workspace authz.Object
	Actor     authz.Object
}

// UpdateDocumentInput carries the data needed to update an existing document.
type UpdateDocumentInput struct {
	ID    string
	Body  string
	Actor authz.Object
}

// DocumentNotFoundError is returned when a document ID does not match any stored document.
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

func (e *ForbiddenError) Error() string {
	return e.Message
}
