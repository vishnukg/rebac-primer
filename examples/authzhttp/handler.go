package authzhttp

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/rebac"
)

// handler holds the authz service.
type handler struct {
	authz AuthorizationService
}

// handleHealth responds with {"ok": true}.
func (h *handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleCheck handles POST /check.
//
// Request body: { "subject": "user:alice", "permission": "can_edit", "resource": "document:roadmapDocument" }
// Response:     { "allowed": true, "trace": ["Check whether ...", "Result: allowed"] }
func (h *handler) handleCheck(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Subject    string `json:"subject"`
		Permission string `json:"permission"`
		Resource   string `json:"resource"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody("invalid JSON: "+err.Error()))
		return
	}
	if body.Subject == "" || body.Permission == "" || body.Resource == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("subject, permission, and resource are required"))
		return
	}

	result, err := h.authz.Check(r.Context(), rebac.CheckRequest{
		Subject:    rebac.Resource(body.Subject),
		Permission: rebac.Permission(body.Permission),
		Resource:   rebac.Resource(body.Resource),
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"allowed": result.Allowed,
		"trace":   result.Trace,
	})
}

// handleWriteRelationships handles POST /relationships.
//
// Request body: { "relationships": [{ "subject": "...", "relation": "...", "resource": "..." }] }
// Response:     { "written": 1 }
func (h *handler) handleWriteRelationships(w http.ResponseWriter, r *http.Request) {
	relationships, err := parseRelationshipBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
		return
	}
	if err := h.authz.WriteRelationships(r.Context(), relationships); err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"written": len(relationships)})
}

// handleDeleteRelationships handles DELETE /relationships.
//
// Request body: { "relationships": [{ "subject": "...", "relation": "...", "resource": "..." }] }
// Response:     { "deleted": 1 }
func (h *handler) handleDeleteRelationships(w http.ResponseWriter, r *http.Request) {
	relationships, err := parseRelationshipBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
		return
	}
	if err := h.authz.DeleteRelationships(r.Context(), relationships); err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"deleted": len(relationships)})
}

// handleListRelationships handles GET /relationships.
//
// Optional query params: ?resource=workspace:productWorkspace&relation=editor
// Response: { "relationships": [...] }
func (h *handler) handleListRelationships(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	// Empty strings in RelationshipFilter are treated as "match any" by FindAll.
	filter := authz.RelationshipFilter{
		Resource: rebac.Resource(q.Get("resource")),
		Relation: rebac.Relation(q.Get("relation")),
	}

	relationships, err := h.authz.ListRelationships(r.Context(), filter)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"relationships": relationships})
}

// writeError maps a service error to an HTTP status code.
//
//   - [authz.RelationshipValidationError] or [authz.CheckValidationError] →
//     422 Unprocessable Entity. The caller sent a well-formed request whose
//     authorization terms are semantically invalid; the message names the
//     problem and is safe to return.
//   - Anything else is an unexpected internal failure (store outage, cancelled
//     context, bug): log it server-side and return a generic 500.
func (h *handler) writeError(w http.ResponseWriter, err error) {
	var relationshipValidation *authz.RelationshipValidationError
	if errors.As(err, &relationshipValidation) {
		writeJSON(w, http.StatusUnprocessableEntity, errorBody(err.Error()))
		return
	}
	var checkValidation *authz.CheckValidationError
	if errors.As(err, &checkValidation) {
		writeJSON(w, http.StatusUnprocessableEntity, errorBody(err.Error()))
		return
	}
	log.Printf("authz: unhandled internal error: %v", err)
	writeJSON(w, http.StatusInternalServerError, errorBody("internal server error"))
}

// parseRelationshipBody reads
// {"relationships":[{"subject":...,"relation":...,"resource":...}]}.
func parseRelationshipBody(r *http.Request) ([]rebac.Relationship, error) {
	var body struct {
		Relationships []struct {
			Subject  string `json:"subject"`
			Relation string `json:"relation"`
			Resource string `json:"resource"`
		} `json:"relationships"`
	}
	if err := readJSON(r, &body); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	out := make([]rebac.Relationship, 0, len(body.Relationships))
	for i, relationship := range body.Relationships {
		if relationship.Subject == "" || relationship.Relation == "" || relationship.Resource == "" {
			return nil, fmt.Errorf(
				"relationships[%d]: subject, relation, and resource are required", i,
			)
		}
		out = append(out, rebac.Relationship{
			Subject:  rebac.Subject(relationship.Subject),
			Relation: rebac.Relation(relationship.Relation),
			Resource: rebac.Resource(relationship.Resource),
		})
	}
	return out, nil
}
