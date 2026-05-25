// Package http provides the HTTP adapter for the documents service.
//
// No external router framework is used — Go 1.22+ ServeMux handles
// method+path patterns like "GET /documents/{id}" natively.
//
// Mirrors typescript/src/documents-service/adapters/http/makeDocumentsHttpServer.ts.
package http

import (
	"net/http"

	"rebac-primer/internal/documents"
)

// NewServer returns an http.Handler with all document routes registered.
// It accepts an [documents.Authenticator] (for authn) and [documents.Service]
// (for domain operations).
func NewServer(authenticator documents.Authenticator, svc documents.Service) http.Handler {
	h := &handler{authenticator: authenticator, docs: svc}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("GET /whoami", h.handleWhoami)
	mux.HandleFunc("POST /documents", h.handleCreateDocument)
	mux.HandleFunc("GET /documents/{id}", h.handleGetDocument)
	mux.HandleFunc("PATCH /documents/{id}", h.handleUpdateDocument)

	return mux
}
