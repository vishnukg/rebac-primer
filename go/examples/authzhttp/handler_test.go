package authzhttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rebac-primer/examples/authzhttp"
	"rebac-primer/internal/authz"
	authzdb "rebac-primer/internal/authz/adapters/db"
	"rebac-primer/internal/authz/adapters/graph"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/shared"
)

// newTestHandler builds a fully-wired authz HTTP handler.
// Optional extra tuples are appended to the standard fixture store.
func newTestHandler(extra ...shared.TupleKey) http.Handler {
	all := append(fixtures.SeedRelationshipTuples(), extra...)
	store := authzdb.New(all...)
	evaluator := graph.NewGraphEvaluator(store)
	svc := authz.New(store, evaluator)
	return authzhttp.NewServer(svc)
}

// roadmapWorkspaceTuple links roadmapDocument to productWorkspace.
// Tests that need document-level permission inheritance must include this.
var roadmapWorkspaceTuple = shared.Tuple(
	fixtures.RoadmapDocument,
	shared.RelationDocumentWorkspace,
	shared.Subject(fixtures.ProductWorkspace),
)

// checkPermission is a test helper that calls POST /check and returns allowed.
func checkPermission(t *testing.T, handler http.Handler, user shared.Object, relation shared.Relation, object shared.Object) bool {
	t.Helper()
	payload, _ := json.Marshal(map[string]string{
		"user":     string(user),
		"relation": string(relation),
		"object":   string(object),
	})
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode /check response: %v", err)
	}
	allowed, _ := resp["allowed"].(bool)
	return allowed
}

func TestAuthzHandler_Health(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var body map[string]bool
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body["ok"] {
		t.Errorf("expected {\"ok\":true}, got %v", body)
	}
}

func TestAuthzHandler_Check_AllowedForEditor(t *testing.T) {
	handler := newTestHandler(roadmapWorkspaceTuple)

	payload, _ := json.Marshal(map[string]string{
		"user":     string(fixtures.Alice),
		"relation": string(shared.RelationDocumentCanEdit),
		"object":   string(fixtures.RoadmapDocument),
	})
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["allowed"] != true {
		t.Errorf("expected allowed=true, got %v", resp["allowed"])
	}
}

func TestAuthzHandler_Check_DeniedForViewer(t *testing.T) {
	handler := newTestHandler(roadmapWorkspaceTuple)

	payload, _ := json.Marshal(map[string]string{
		"user":     string(fixtures.Bob),
		"relation": string(shared.RelationDocumentCanEdit),
		"object":   string(fixtures.RoadmapDocument),
	})
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["allowed"] != false {
		t.Errorf("expected allowed=false, got %v", resp["allowed"])
	}
}

func TestAuthzHandler_Check_IncludesTrace(t *testing.T) {
	handler := newTestHandler(roadmapWorkspaceTuple)

	payload, _ := json.Marshal(map[string]string{
		"user":     string(fixtures.Casey),
		"relation": string(shared.RelationDocumentCanRead),
		"object":   string(fixtures.RoadmapDocument),
	})
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	trace, ok := resp["trace"].([]any)
	if !ok || len(trace) == 0 {
		t.Fatalf("expected non-empty trace array, got %T: %v", resp["trace"], resp["trace"])
	}
	last, _ := trace[len(trace)-1].(string)
	if last != "Result: denied" {
		t.Errorf("expected last trace line %q, got %q", "Result: denied", last)
	}
}

func TestAuthzHandler_WriteTuples_ThenCheck(t *testing.T) {
	handler := newTestHandler()

	writePayload, _ := json.Marshal(map[string]any{
		"tuples": []map[string]string{{
			"object":   string(fixtures.RoadmapDocument),
			"relation": string(shared.RelationDocumentWorkspace),
			"user":     string(fixtures.ProductWorkspace),
		}},
	})
	writeReq := httptest.NewRequest(http.MethodPost, "/tuples", bytes.NewReader(writePayload))
	writeReq.Header.Set("Content-Type", "application/json")
	writeRec := httptest.NewRecorder()
	handler.ServeHTTP(writeRec, writeReq)

	if writeRec.Code != http.StatusOK {
		t.Fatalf("write tuples: expected 200, got %d — body: %s", writeRec.Code, writeRec.Body.String())
	}
	var writeResp map[string]any
	if err := json.NewDecoder(writeRec.Body).Decode(&writeResp); err != nil {
		t.Fatalf("decode write response: %v", err)
	}
	if writeResp["written"] != float64(1) {
		t.Errorf("expected written=1, got %v", writeResp["written"])
	}

	if !checkPermission(t, handler, fixtures.Alice, shared.RelationDocumentCanRead, fixtures.RoadmapDocument) {
		t.Error("expected alice can_read=true after writing workspace tuple")
	}
}

func TestAuthzHandler_DeleteTuples_RevokesPermission(t *testing.T) {
	handler := newTestHandler(roadmapWorkspaceTuple)

	if !checkPermission(t, handler, fixtures.Bob, shared.RelationDocumentCanRead, fixtures.RoadmapDocument) {
		t.Fatal("expected bob can_read=true before delete")
	}

	deletePayload, _ := json.Marshal(map[string]any{
		"tuples": []map[string]string{{
			"object":   string(fixtures.RoadmapDocument),
			"relation": string(shared.RelationDocumentWorkspace),
			"user":     string(fixtures.ProductWorkspace),
		}},
	})
	deleteReq := httptest.NewRequest(http.MethodDelete, "/tuples", bytes.NewReader(deletePayload))
	deleteReq.Header.Set("Content-Type", "application/json")
	deleteRec := httptest.NewRecorder()
	handler.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete tuples: expected 200, got %d", deleteRec.Code)
	}

	if checkPermission(t, handler, fixtures.Bob, shared.RelationDocumentCanRead, fixtures.RoadmapDocument) {
		t.Error("expected bob can_read=false after deleting workspace tuple")
	}
}

func TestAuthzHandler_WriteTuples_InvalidTupleReturns422(t *testing.T) {
	handler := newTestHandler()

	// "roadmap" is non-empty (so it passes the handler's required-field check) but
	// is not a valid "type:id" object, so domain validation rejects it. That maps
	// to 422 Unprocessable Entity — the request was understood but is invalid.
	payload, _ := json.Marshal(map[string]any{
		"tuples": []map[string]string{{
			"object":   "roadmap",
			"relation": string(shared.RelationDocumentOwner),
			"user":     string(fixtures.Alice),
		}},
	})
	req := httptest.NewRequest(http.MethodPost, "/tuples", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthzHandler_ListTuples_ReturnsAllTuples(t *testing.T) {
	handler := newTestHandler() // 4 seed tuples from fixtures
	req := httptest.NewRequest(http.MethodGet, "/tuples", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	tuples, ok := resp["tuples"].([]any)
	if !ok {
		t.Fatalf("expected tuples to be an array, got %T", resp["tuples"])
	}
	if len(tuples) != 4 {
		t.Errorf("expected 4 seed tuples, got %d", len(tuples))
	}
}
