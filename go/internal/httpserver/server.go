// Package httpserver wires the domain services to an HTTP server using the
// Go 1.22+ ServeMux method+path pattern syntax.
//
// No external router framework is used. The stdlib ServeMux added support for
// patterns like "GET /documents/{id}" in Go 1.22, which is all we need here.
package httpserver

import (
	"encoding/json"
	"net/http"

	"rebac-primer/internal/domain"
)

// NewServer returns an http.Handler with all routes registered.
// It accepts domain.DocumentOperations so the HTTP layer can be tested without
// a real server — the same pattern as TS createHttpServer.
func NewServer(docs domain.DocumentOperations) http.Handler {
	h := &handler{docs: docs}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("POST /documents", h.handleCreateDocument)
	mux.HandleFunc("GET /documents/{id}", h.handleGetDocument)
	mux.HandleFunc("PATCH /documents/{id}", h.handleUpdateDocument)

	return mux
}

// --- JSON helpers ---

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func readJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}
