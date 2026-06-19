// Package api serves the documents service over HTTP.
//
// No external router framework is used — Go 1.22+ ServeMux handles
// method+path patterns like "GET /documents/{id}" natively.
package api

import (
	"context"
	"net/http"

	"rebac-primer/internal/documents"
	"rebac-primer/internal/rebac"
)

// Authenticator is the identity capability required by the HTTP API.
// Implementations may verify local demo tokens or call an external identity
// provider.
type Authenticator interface {
	VerifyAccessToken(authorizationHeader string) (documents.AuthenticatedUser, error)
}

// DocumentService is the set of document operations exposed by this HTTP API.
// It is declared here, at the point of use, so the documents package can return
// a concrete implementation without owning its consumers' abstractions.
type DocumentService interface {
	Create(ctx context.Context, input documents.CreateDocumentInput) (*documents.CollaborativeDocument, error)
	Read(ctx context.Context, id string, actor rebac.Object) (*documents.CollaborativeDocument, error)
	Update(ctx context.Context, input documents.UpdateDocumentInput) (*documents.CollaborativeDocument, error)
}

// NewServer returns an http.Handler with all document routes registered.
func NewServer(authenticator Authenticator, svc DocumentService) http.Handler {
	h := &handler{authenticator: authenticator, docs: svc}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("GET /whoami", h.handleWhoami)
	mux.HandleFunc("POST /documents", h.handleCreateDocument)
	mux.HandleFunc("GET /documents/{id}", h.handleGetDocument)
	mux.HandleFunc("PATCH /documents/{id}", h.handleUpdateDocument)

	return mux
}
