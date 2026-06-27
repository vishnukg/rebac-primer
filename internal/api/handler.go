package api

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"rebac-primer/internal/documents"
	"rebac-primer/internal/rebac"
)

// handler holds the domain operations and authenticator.
type handler struct {
	authenticator Authenticator
	docs          DocumentService
}

// handleHealth responds with {"ok": true}.
func (h *handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleWhoami returns the verified identity for the bearer token.
func (h *handler) handleWhoami(w http.ResponseWriter, r *http.Request) {
	user, err := h.authenticate(r)
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
	user, err := h.authenticate(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if err = requireScope(user, "documents:write"); err != nil {
		h.writeError(w, err)
		return
	}

	var body struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Body        string `json:"body"`
		WorkspaceID string `json:"workspaceId"`
	}
	if err = readJSON(w, r, &body); err != nil {
		writeJSONReadError(w, err)
		return
	}
	if isBlank(body.ID) || isBlank(body.Title) || isBlank(body.Body) || isBlank(body.WorkspaceID) {
		writeJSON(w, http.StatusBadRequest, errorBody("id, title, body, and workspaceId are required"))
		return
	}

	doc, err := h.docs.Create(r.Context(), documents.CreateDocumentInput{
		ID:        body.ID,
		Title:     body.Title,
		Body:      body.Body,
		Workspace: rebac.Workspace(body.WorkspaceID),
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
	user, err := h.authenticate(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if err = requireScope(user, "documents:read"); err != nil {
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
	user, err := h.authenticate(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if err = requireScope(user, "documents:write"); err != nil {
		h.writeError(w, err)
		return
	}

	id := r.PathValue("id")
	var body struct {
		Body string `json:"body"`
	}
	if err = readJSON(w, r, &body); err != nil {
		writeJSONReadError(w, err)
		return
	}
	if isBlank(body.Body) {
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

// authenticate verifies the request's bearer token and returns the caller.
func (h *handler) authenticate(r *http.Request) (documents.AuthenticatedUser, error) {
	return h.authenticator.VerifyAccessToken(r.Header.Get("Authorization"))
}

func requireScope(user documents.AuthenticatedUser, scope string) error {
	if user.HasScope(scope) {
		return nil
	}
	return &documents.InsufficientScopeError{Required: scope}
}

func isBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// writeError maps a domain error to an HTTP status code.
//
// Each known domain error maps to a specific 4xx. Anything else is an
// unexpected internal failure (a bug, a store outage, a cancelled context): we
// log the real error server-side and return a generic 500, rather than leaking
// internal details to the caller or mislabelling a server fault as a 400.
func (h *handler) writeError(w http.ResponseWriter, err error) {
	if documents.IsAuthenticationError(err) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="rebac-primer"`)
		writeJSON(w, http.StatusUnauthorized, errorBody(err.Error()))
		return
	}

	var insufficientScope *documents.InsufficientScopeError
	if errors.As(err, &insufficientScope) {
		w.Header().Set("WWW-Authenticate", `Bearer error="insufficient_scope", scope="`+insufficientScope.Required+`"`)
		writeJSON(w, http.StatusForbidden, errorBody(err.Error()))
		return
	}

	var notFound *documents.DocumentNotFoundError
	if errors.As(err, &notFound) {
		writeJSON(w, http.StatusNotFound, errorBody(err.Error()))
		return
	}

	var alreadyExists *documents.DocumentAlreadyExistsError
	if errors.As(err, &alreadyExists) {
		writeJSON(w, http.StatusConflict, errorBody(err.Error()))
		return
	}

	var forbidden *documents.ForbiddenError
	if errors.As(err, &forbidden) {
		writeJSON(w, http.StatusForbidden, errorBody(err.Error()))
		return
	}

	log.Printf("documents: unhandled internal error: %v", err)
	writeJSON(w, http.StatusInternalServerError, errorBody("internal server error"))
}
