// Package documents is the documents service's core.
//
// It defines the [Service] driving port and the driven ports
// ([DocumentRepository], [AuthzClient], [Authenticator]) that adapters must
// satisfy.  The concrete implementation lives in [New] — everything else in
// this package is an interface or a domain type.
//
// Hexagonal architecture in one diagram:
//
//	                    ┌──────────────────────────────────┐
//	   driving adapters │            documents             │  driven adapters
//	   (HTTP handler)   │                                  │  (db, authn, authz)
//	        ───────────►│  Service                         │
//	                    │    Create() ─────────────────────│──►  DocumentRepository
//	                    │    Read()   ─────────────────────│──►  AuthzClient
//	                    │    Update() ─────────────────────│──►  Authenticator (HTTP layer)
//	                    └──────────────────────────────────┘
//
// Mirrors typescript/src/documents-service/core/domain/types.ts
// and typescript/src/documents-service/core/ports/.
package documents

import (
	"context"
	"fmt"

	"rebac-primer/internal/shared"
)

// ── Driving port ──────────────────────────────────────────────────────────────

// Service is the driving port — what the HTTP handler and tests depend on.
// The concrete implementation is returned by [New].
//
// Mirrors typescript/src/documents-service/core/domain/types.ts (Documents).
type Service interface {
	Create(ctx context.Context, input CreateDocumentInput) (*CollaborativeDocument, error)
	Read(ctx context.Context, id string, actor shared.Object) (*CollaborativeDocument, error)
	Update(ctx context.Context, input UpdateDocumentInput) (*CollaborativeDocument, error)
}

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

// ── Domain errors ─────────────────────────────────────────────────────────────

// DocumentNotFoundError is returned when a document ID does not match any
// stored document.
//
// Mirrors typescript/src/documents-service/core/domain/types.ts (DocumentNotFoundError).
type DocumentNotFoundError struct {
	ID string
}

func (e *DocumentNotFoundError) Error() string {
	return fmt.Sprintf("document not found: %s", e.ID)
}

// ForbiddenError is returned when an actor lacks the required permission.
//
// Mirrors typescript/src/documents-service/core/domain/types.ts (ForbiddenError).
type ForbiddenError struct {
	Message string
}

func (e *ForbiddenError) Error() string { return e.Message }
