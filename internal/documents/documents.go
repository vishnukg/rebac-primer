// Package documents is the document service: it creates, reads, and updates
// collaborative documents, gating every operation on an authorization check.
//
// New builds a Service from the things it needs — a DocumentRepository for
// persistence and an AuthzClient for permission checks. This package ships an
// in-memory DocumentRepository (NewInMemoryRepository) and a demo Authenticator
// (NewDemoTokenVerifier) for the HTTP layer; production swaps either for a real
// implementation. Callers depend on the Service interface, never the concrete
// type.
package documents

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"rebac-primer/internal/rebac"
)

// Service creates, reads, and updates documents. It is what the HTTP handler and
// tests call into; New returns the concrete implementation.
type Service interface {
	Create(ctx context.Context, input CreateDocumentInput) (*CollaborativeDocument, error)
	Read(ctx context.Context, id string, actor rebac.Object) (*CollaborativeDocument, error)
	Update(ctx context.Context, input UpdateDocumentInput) (*CollaborativeDocument, error)
}

// CollaborativeDocument is a stored document. It is defined here, alongside the
// Service, so the repository and the service share one type with no conversion
// or alias. The JSON tags are the wire format the HTTP layer emits.
type CollaborativeDocument struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	Body      string       `json:"body"`
	Workspace rebac.Object `json:"workspace"`
	UpdatedBy rebac.Object `json:"updatedBy"`
}

// CreateDocumentInput carries the data needed to create a new document.
type CreateDocumentInput struct {
	ID        string
	Title     string
	Body      string
	Workspace rebac.Object
	Actor     rebac.Object
}

// UpdateDocumentInput carries the data needed to update an existing document.
type UpdateDocumentInput struct {
	ID    string
	Body  string
	Actor rebac.Object
}

// DocumentRepository stores documents. NewInMemoryRepository is the default
// implementation.
type DocumentRepository interface {
	Create(ctx context.Context, doc CollaborativeDocument) error
	Save(ctx context.Context, doc CollaborativeDocument) error
	FindByID(ctx context.Context, id string) (*CollaborativeDocument, error)
	Delete(ctx context.Context, id string) error
}

// AuthzClient is what the service needs from authorization: check a permission
// and write the relationship tuples a new document implies.
//
// The in-process authz.Service satisfies this directly (its method set is a
// superset), and so would an HTTP client to a standalone authz server — the
// service never knows which.
type AuthzClient interface {
	Check(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error)
	WriteTuples(ctx context.Context, tuples []rebac.TupleKey) error
	DeleteTuples(ctx context.Context, tuples []rebac.TupleKey) error
}

// AuthenticatedUser is the verified identity returned after a successful token check.
type AuthenticatedUser struct {
	Subject rebac.Object // e.g. "user:alice"
	Scopes  []string     // OAuth scopes granted to this token
}

// Authenticator establishes caller identity from an Authorization header. The
// HTTP handler calls it before every request; NewDemoTokenVerifier is the
// development implementation.
type Authenticator interface {
	VerifyAccessToken(authorizationHeader string) (AuthenticatedUser, error)
}

// AuthenticationError is returned when a token is missing or invalid. The HTTP
// layer maps it to 401 Unauthorized.
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string { return e.Message }

// IsAuthenticationError reports whether err is, or wraps, an AuthenticationError.
// It uses errors.As so it still matches through a fmt.Errorf("...: %w", err)
// wrapper — the same unwrapping the HTTP layer relies on for the other errors.
func IsAuthenticationError(err error) bool {
	var authErr *AuthenticationError
	return errors.As(err, &authErr)
}

// InsufficientScopeError is returned when a valid access token does not grant
// the coarse API scope required by an endpoint. ReBAC still performs the
// separate object-level decision after this check passes.
type InsufficientScopeError struct {
	Required string
}

func (e *InsufficientScopeError) Error() string {
	return fmt.Sprintf("access token requires scope %q", e.Required)
}

// HasScope reports whether an authenticated token includes scope.
func (u AuthenticatedUser) HasScope(scope string) bool {
	return slices.Contains(u.Scopes, scope)
}

// DocumentAlreadyExistsError is returned when Create is given an ID that is
// already taken. The HTTP layer maps it to 409 Conflict.
type DocumentAlreadyExistsError struct {
	ID string
}

func (e *DocumentAlreadyExistsError) Error() string {
	return fmt.Sprintf("document already exists: %s", e.ID)
}

// DocumentNotFoundError is returned when an ID matches no stored document. The
// HTTP layer maps it to 404 Not Found.
type DocumentNotFoundError struct {
	ID string
}

func (e *DocumentNotFoundError) Error() string {
	return fmt.Sprintf("document not found: %s", e.ID)
}

// ForbiddenError is returned when an actor lacks the required permission. The
// HTTP layer maps it to 403 Forbidden.
type ForbiddenError struct {
	Message string
}

func (e *ForbiddenError) Error() string { return e.Message }
