// Package ports defines the driven ports for the documents service.
//
// A "driven port" is an interface the domain calls out to.  Adapters supply
// the concrete implementation; the domain never sees the concrete type.
//
// Mirrors typescript/src/documents-service/core/ports/.
package ports

import (
	"context"

	"rebac-primer/internal/shared"
)

// ── CollaborativeDocument ─────────────────────────────────────────────────────

// CollaborativeDocument is the aggregate root shared by the domain and
// repository port.  Defining it here avoids an import cycle: the domain
// imports ports; the repository adapter imports ports; neither needs to
// import the other.
//
// JSON tags are included here because both the domain and HTTP adapter need
// consistent serialisation — they do not make the type HTTP-specific.
type CollaborativeDocument struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Body      string        `json:"body"`
	Workspace shared.Object `json:"workspace"`
	UpdatedBy shared.Object `json:"updatedBy"`
}

// ── DocumentRepository ────────────────────────────────────────────────────────

// DocumentRepository is the persistence interface for CollaborativeDocument.
// The domain service depends only on this interface — it never knows whether
// documents are stored in memory, Postgres, or somewhere else.
//
// Mirrors typescript/src/documents-service/core/ports/documentRepository.ts.
type DocumentRepository interface {
	Save(ctx context.Context, doc CollaborativeDocument) error
	FindByID(ctx context.Context, id string) (*CollaborativeDocument, error)
	List(ctx context.Context) ([]CollaborativeDocument, error)
}

// ── AuthzClient ───────────────────────────────────────────────────────────────

// AuthzClient is what the documents domain needs from the authz service.
//
// In tests this is satisfied by a fake in-memory implementation.
// In production it can be satisfied by the in-process authz domain (structural
// typing: authzdomain.AuthzService is a superset of AuthzClient) or by an HTTP
// client that calls the authz service over the wire.
//
// Mirrors typescript/src/documents-service/core/ports/authzClient.ts.
type AuthzClient interface {
	Check(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error)
	WriteTuples(ctx context.Context, tuples []shared.TupleKey) error
}

// ── Authenticator ─────────────────────────────────────────────────────────────

// AuthenticatedUser is the verified identity returned after a successful token check.
type AuthenticatedUser struct {
	Subject shared.Object // e.g. "user:alice"
	Scopes  []string      // OAuth scopes granted to this token
}

// AuthenticationError is returned when a token is missing or invalid.
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string { return e.Message }

// IsAuthenticationError reports whether err wraps an AuthenticationError.
func IsAuthenticationError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*AuthenticationError)
	return ok
}

// Authenticator is the port the HTTP handler calls to establish caller identity.
//
// Mirrors typescript/src/documents-service/core/ports/authenticator.ts.
type Authenticator interface {
	VerifyAccessToken(authorizationHeader string) (AuthenticatedUser, error)
}
