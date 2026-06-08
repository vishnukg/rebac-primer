package authzhttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/rebac"
)

// handler holds the authz service.
type handler struct {
	authz authz.Service
}

// handleHealth responds with {"ok": true}.
func (h *handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleCheck handles POST /check.
//
// Request body: { "user": "user:alice", "relation": "can_edit", "object": "document:roadmapDocument" }
// Response:     { "allowed": true, "trace": ["Check whether ...", "Result: allowed"] }
func (h *handler) handleCheck(w http.ResponseWriter, r *http.Request) {
	var body struct {
		User     string `json:"user"`
		Relation string `json:"relation"`
		Object   string `json:"object"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody("invalid JSON: "+err.Error()))
		return
	}
	if body.User == "" || body.Relation == "" || body.Object == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("user, relation, and object are required"))
		return
	}

	result, err := h.authz.Check(r.Context(), rebac.CheckRequest{
		User:     rebac.Object(body.User),
		Relation: rebac.Relation(body.Relation),
		Object:   rebac.Object(body.Object),
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

// handleWriteTuples handles POST /tuples.
//
// Request body: { "tuples": [{ "object": "...", "relation": "...", "user": "..." }] }
// Response:     { "written": 1 }
func (h *handler) handleWriteTuples(w http.ResponseWriter, r *http.Request) {
	tuples, err := parseTupleBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
		return
	}
	if err := h.authz.WriteTuples(r.Context(), tuples); err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"written": len(tuples)})
}

// handleDeleteTuples handles DELETE /tuples.
//
// Request body: { "tuples": [{ "object": "...", "relation": "...", "user": "..." }] }
// Response:     { "deleted": 1 }
func (h *handler) handleDeleteTuples(w http.ResponseWriter, r *http.Request) {
	tuples, err := parseTupleBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
		return
	}
	if err := h.authz.DeleteTuples(r.Context(), tuples); err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"deleted": len(tuples)})
}

// handleListTuples handles GET /tuples.
//
// Optional query params: ?object=workspace:productWorkspace&relation=editor
// Response: { "tuples": [...] }
func (h *handler) handleListTuples(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	// Empty strings in TupleFilter are treated as "match any" by FindAll.
	filter := authz.TupleFilter{
		Object:   rebac.Object(q.Get("object")),
		Relation: rebac.Relation(q.Get("relation")),
	}

	tuples, err := h.authz.ListTuples(r.Context(), filter)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tuples": tuples})
}

// writeError maps a service error to an HTTP status code.
//
//   - [authz.TupleValidationError] → 422 Unprocessable Entity. The caller sent a
//     well-formed request whose tuple is semantically invalid; the message names
//     the problem and is safe to return.
//   - Anything else is an unexpected internal failure (store outage, cancelled
//     context, bug): log it server-side and return a generic 500.
func (h *handler) writeError(w http.ResponseWriter, err error) {
	var tupleValidation *authz.TupleValidationError
	if errors.As(err, &tupleValidation) {
		writeJSON(w, http.StatusUnprocessableEntity, errorBody(err.Error()))
		return
	}
	log.Printf("authz: unhandled internal error: %v", err)
	writeJSON(w, http.StatusInternalServerError, errorBody("internal server error"))
}

// parseTupleBody reads a JSON body of shape { "tuples": [{object,relation,user}] }.
func parseTupleBody(r *http.Request) ([]rebac.TupleKey, error) {
	var body struct {
		Tuples []struct {
			Object   string `json:"object"`
			Relation string `json:"relation"`
			User     string `json:"user"`
		} `json:"tuples"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	out := make([]rebac.TupleKey, 0, len(body.Tuples))
	for i, t := range body.Tuples {
		if t.Object == "" || t.Relation == "" || t.User == "" {
			return nil, fmt.Errorf("tuples[%d]: object, relation, and user are required", i)
		}
		out = append(out, rebac.TupleKey{
			Object:   rebac.Object(t.Object),
			Relation: rebac.Relation(t.Relation),
			User:     rebac.Subject(t.User),
		})
	}
	return out, nil
}
