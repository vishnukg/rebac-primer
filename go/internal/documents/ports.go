package documents

import (
	"context"

	"rebac-primer/internal/shared"
)

// ── Aggregate root ────────────────────────────────────────────────────────────

// CollaborativeDocument is the aggregate root shared by the domain and its
// persistence port.  Defining it here (rather than in a separate ports package)
// keeps the import graph flat: adapters import documents; domain uses the type
// directly without a conversion or alias.
//
// JSON tags use camelCase to match the TypeScript wire format.
//
// Mirrors typescript/src/documents-service/core/domain/types.ts (CollaborativeDocument).
type CollaborativeDocument struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Body      string        `json:"body"`
	Workspace shared.Object `json:"workspace"`
	UpdatedBy shared.Object `json:"updatedBy"`
}

// ── Driven ports ──────────────────────────────────────────────────────────────
//
// A driven port is an interface the domain calls out to.  Adapters in adapters/
// supply concrete implementations; the domain never imports adapters.

// DocumentRepository is the persistence port for [CollaborativeDocument].
//
// Mirrors typescript/src/documents-service/core/ports/documentRepository.ts.
type DocumentRepository interface {
	Save(ctx context.Context, doc CollaborativeDocument) error
	FindByID(ctx context.Context, id string) (*CollaborativeDocument, error)
	List(ctx context.Context) ([]CollaborativeDocument, error)
}

// AuthzClient is what the documents domain needs from the authz service.
//
// In tests a fake in-memory implementation is used.  In production this is
// satisfied by the in-process authz [authz.Service] (Go structural typing:
// authz.Service is a superset of AuthzClient) or by an HTTP client that calls
// the standalone authz server.
//
// Mirrors typescript/src/documents-service/core/ports/authzClient.ts.
type AuthzClient interface {
	Check(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error)
	WriteTuples(ctx context.Context, tuples []shared.TupleKey) error
}

// ── Authentication port ───────────────────────────────────────────────────────

// AuthenticatedUser is the verified identity returned after a successful token check.
type AuthenticatedUser struct {
	Subject shared.Object // e.g. "user:alice"
	Scopes  []string      // OAuth scopes granted to this token
}

// AuthenticationError is returned when a token is missing or invalid.
// The HTTP adapter maps this to 401 Unauthorized.
//
// Mirrors typescript/src/documents-service/core/ports/authenticator.ts (AuthenticationError).
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string { return e.Message }

// IsAuthenticationError reports whether err is (or wraps) an [AuthenticationError].
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
