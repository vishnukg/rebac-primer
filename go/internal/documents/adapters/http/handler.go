package http

import (
	"errors"
	"net/http"

	"rebac-primer/internal/documents"
	"rebac-primer/internal/shared"
)

// handler holds the domain operations and authenticator.
type handler struct {
	authenticator documents.Authenticator
	docs          documents.Service
}

// handleHealth responds with {"ok": true}.
func (h *handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleWhoami returns the verified identity for the bearer token.
func (h *handler) handleWhoami(w http.ResponseWriter, r *http.Request) {
	user, err := h.authenticator.VerifyAccessToken(r.Header.Get("Authorization"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":   user.Subject,
		"scopes": user.Scopes,
	})
}

// handleCreateDocument handles POST /documents.
//
// Authorization header: Bearer <token>
// Request body (JSON): { "id": "...", "title": "...", "body": "...", "workspaceId": "..." }
// Response: 201 with { "document": {...} }
func (h *handler) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
	user, err := h.authenticator.VerifyAccessToken(r.Header.Get("Authorization"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	var body struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Body        string `json:"body"`
		WorkspaceID string `json:"workspaceId"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody("invalid JSON: "+err.Error()))
		return
	}
	if body.ID == "" || body.Title == "" || body.Body == "" || body.WorkspaceID == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("id, title, body, and workspaceId are required"))
		return
	}

	doc, err := h.docs.Create(r.Context(), documents.CreateDocumentInput{
		ID:        body.ID,
		Title:     body.Title,
		Body:      body.Body,
		Workspace: shared.Workspace(body.WorkspaceID),
		Actor:     user.Subject,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"document": doc})
}

// handleGetDocument handles GET /documents/{id}.
func (h *handler) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	user, err := h.authenticator.VerifyAccessToken(r.Header.Get("Authorization"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	id := r.PathValue("id")
	doc, err := h.docs.Read(r.Context(), id, user.Subject)
	if err != nil {
		h.writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"document": doc})
}

// handleUpdateDocument handles PATCH /documents/{id}.
func (h *handler) handleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	user, err := h.authenticator.VerifyAccessToken(r.Header.Get("Authorization"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	id := r.PathValue("id")
	var body struct {
		Body string `json:"body"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody("invalid JSON: "+err.Error()))
		return
	}
	if body.Body == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("body is required"))
		return
	}

	doc, err := h.docs.Update(r.Context(), documents.UpdateDocumentInput{
		ID:    id,
		Body:  body.Body,
		Actor: user.Subject,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"document": doc})
}

// writeError maps domain errors to HTTP status codes.
func (h *handler) writeError(w http.ResponseWriter, err error) {
	if documents.IsAuthenticationError(err) {
		writeJSON(w, http.StatusUnauthorized, errorBody(err.Error()))
		return
	}

	var notFound *documents.DocumentNotFoundError
	if errors.As(err, &notFound) {
		writeJSON(w, http.StatusNotFound, errorBody(err.Error()))
		return
	}

	var forbidden *documents.ForbiddenError
	if errors.As(err, &forbidden) {
		writeJSON(w, http.StatusForbidden, errorBody(err.Error()))
		return
	}

	writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
}
