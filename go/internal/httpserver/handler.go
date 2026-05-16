package httpserver

import (
	"errors"
	"net/http"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/domain"
)

// handler holds the domain operations and implements each route as a method.
// Keeping fields unexported forces callers to go through NewServer.
type handler struct {
	docs domain.DocumentOperations
}

// handleHealth responds with {"ok": true}.
func (h *handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleCreateDocument handles POST /documents.
//
// Request body (JSON):
//
//	{ "id": "...", "title": "...", "body": "...", "workspaceId": "...", "actorId": "..." }
//
// Response: 201 with { "document": {...} }
func (h *handler) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Body        string `json:"body"`
		WorkspaceID string `json:"workspaceId"`
		ActorID     string `json:"actorId"`
	}

	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody("invalid JSON: "+err.Error()))
		return
	}

	if body.ID == "" || body.Title == "" || body.Body == "" || body.WorkspaceID == "" || body.ActorID == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("id, title, body, workspaceId, and actorId are required"))
		return
	}

	doc, err := h.docs.Create(r.Context(), domain.CreateDocumentInput{
		ID:        body.ID,
		Title:     body.Title,
		Body:      body.Body,
		Workspace: authz.Workspace(body.WorkspaceID),
		Actor:     authz.User(body.ActorID),
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"document": doc})
}

// handleGetDocument handles GET /documents/{id}?actorId=...
//
// Response: 200 with { "document": {...} }
func (h *handler) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	actorID := r.URL.Query().Get("actorId")
	if actorID == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("Missing query parameter: actorId"))
		return
	}

	doc, err := h.docs.Read(r.Context(), id, authz.User(actorID))
	if err != nil {
		h.writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"document": doc})
}

// handleUpdateDocument handles PATCH /documents/{id}.
//
// Request body (JSON):
//
//	{ "body": "...", "actorId": "..." }
//
// Response: 200 with { "document": {...} }
func (h *handler) handleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Body    string `json:"body"`
		ActorID string `json:"actorId"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody("invalid JSON: "+err.Error()))
		return
	}

	if body.Body == "" || body.ActorID == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("body and actorId are required"))
		return
	}

	doc, err := h.docs.Update(r.Context(), domain.UpdateDocumentInput{
		ID:    id,
		Body:  body.Body,
		Actor: authz.User(body.ActorID),
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"document": doc})
}

// writeError maps domain errors to HTTP status codes.
// This is the Go equivalent of the TS errorResponse function.
func (h *handler) writeError(w http.ResponseWriter, err error) {
	var notFound *domain.DocumentNotFoundError
	if errors.As(err, &notFound) {
		writeJSON(w, http.StatusNotFound, errorBody(err.Error()))
		return
	}

	var forbidden *domain.ForbiddenError
	if errors.As(err, &forbidden) {
		writeJSON(w, http.StatusForbidden, errorBody(err.Error()))
		return
	}

	writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
}

func errorBody(msg string) map[string]string {
	return map[string]string{"error": msg}
}
