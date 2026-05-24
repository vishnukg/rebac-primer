// Package http wires the documents service to an HTTP server using the
// Go 1.22+ ServeMux method+path pattern syntax.
//
// No external router framework is used. The stdlib ServeMux added support for
// patterns like "GET /documents/{id}" in Go 1.22, which is all we need.
//
// Mirrors typescript/src/documents-service/adapters/http/makeDocumentsHttpServer.ts.
package http

import (
	"net/http"

	"rebac-primer/internal/documentsservice/core/domain"
	"rebac-primer/internal/documentsservice/core/ports"
)

// NewServer returns an http.Handler with all routes registered.
// It accepts an Authenticator (for authn) and Documents (for domain operations).
func NewServer(authenticator ports.Authenticator, docs domain.Documents) http.Handler {
	h := &handler{authenticator: authenticator, docs: docs}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("GET /whoami", h.handleWhoami)
	mux.HandleFunc("POST /documents", h.handleCreateDocument)
	mux.HandleFunc("GET /documents/{id}", h.handleGetDocument)
	mux.HandleFunc("PATCH /documents/{id}", h.handleUpdateDocument)

	return mux
}
