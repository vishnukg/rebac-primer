package authzhttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rebac-primer/examples/authzhttp"
	"rebac-primer/internal/authz"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/rebac"
)

func TestAuthzHandler_Health(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	svc := authz.New(store, authz.NewGraphEvaluator(store))
	handler := authzhttp.NewServer(svc)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
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
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	svc := authz.New(store, authz.NewGraphEvaluator(store))
	handler := authzhttp.NewServer(svc)

	payload, err := json.Marshal(map[string]string{
		"user":     string(fixtures.Alice),
		"relation": string(rebac.RelationDocumentCanEdit),
		"object":   string(fixtures.RoadmapDocument),
	})
	if err != nil {
		t.Fatalf("marshal check payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
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
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	svc := authz.New(store, authz.NewGraphEvaluator(store))
	handler := authzhttp.NewServer(svc)

	payload, err := json.Marshal(map[string]string{
		"user":     string(fixtures.Bob),
		"relation": string(rebac.RelationDocumentCanEdit),
		"object":   string(fixtures.RoadmapDocument),
	})
	if err != nil {
		t.Fatalf("marshal check payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
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
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	svc := authz.New(store, authz.NewGraphEvaluator(store))
	handler := authzhttp.NewServer(svc)

	payload, err := json.Marshal(map[string]string{
		"user":     string(fixtures.Casey),
		"relation": string(rebac.RelationDocumentCanRead),
		"object":   string(fixtures.RoadmapDocument),
	})
	if err != nil {
		t.Fatalf("marshal check payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
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
	last, ok := trace[len(trace)-1].(string)
	if !ok {
		t.Fatalf("last trace entry = %T, want string", trace[len(trace)-1])
	}
	if last != "Result: denied" {
		t.Errorf("expected last trace line %q, got %q", "Result: denied", last)
	}
}

func TestAuthzHandler_WriteTuples_ThenCheck(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	svc := authz.New(store, authz.NewGraphEvaluator(store))
	handler := authzhttp.NewServer(svc)

	writePayload, err := json.Marshal(map[string]any{
		"tuples": []map[string]string{{
			"object":   string(fixtures.RoadmapDocument),
			"relation": string(rebac.RelationDocumentWorkspace),
			"user":     string(fixtures.ProductWorkspace),
		}},
	})
	if err != nil {
		t.Fatalf("marshal write payload: %v", err)
	}
	writeReq := httptest.NewRequest(http.MethodPost, "/tuples", bytes.NewReader(writePayload))
	writeReq.Header.Set("Content-Type", "application/json")
	writeRec := httptest.NewRecorder()
	// Act
	handler.ServeHTTP(writeRec, writeReq)

	// Assert
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

	checkPayload, err := json.Marshal(map[string]string{
		"user": string(fixtures.Alice), "relation": string(rebac.RelationDocumentCanRead), "object": string(fixtures.RoadmapDocument),
	})
	if err != nil {
		t.Fatalf("marshal check payload: %v", err)
	}
	checkReq := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader(checkPayload))
	checkReq.Header.Set("Content-Type", "application/json")
	checkRec := httptest.NewRecorder()
	// Act
	handler.ServeHTTP(checkRec, checkReq)
	var checkResp map[string]any
	if err := json.NewDecoder(checkRec.Body).Decode(&checkResp); err != nil {
		t.Fatalf("decode /check response: %v", err)
	}
	allowed, ok := checkResp["allowed"].(bool)
	if !ok {
		t.Fatalf("allowed = %T, want bool", checkResp["allowed"])
	}
	if !allowed {
		t.Error("expected alice can_read=true after writing workspace tuple")
	}
}

func TestAuthzHandler_DeleteTuples_RevokesPermission(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	svc := authz.New(store, authz.NewGraphEvaluator(store))
	handler := authzhttp.NewServer(svc)

	beforePayload, err := json.Marshal(map[string]string{
		"user": string(fixtures.Bob), "relation": string(rebac.RelationDocumentCanRead), "object": string(fixtures.RoadmapDocument),
	})
	if err != nil {
		t.Fatalf("marshal before-check payload: %v", err)
	}
	beforeReq := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader(beforePayload))
	beforeReq.Header.Set("Content-Type", "application/json")
	beforeRec := httptest.NewRecorder()
	// Act
	handler.ServeHTTP(beforeRec, beforeReq)
	var beforeResp map[string]any
	if err := json.NewDecoder(beforeRec.Body).Decode(&beforeResp); err != nil {
		t.Fatalf("decode before-check response: %v", err)
	}
	beforeAllowed, ok := beforeResp["allowed"].(bool)
	if !ok {
		t.Fatalf("before allowed = %T, want bool", beforeResp["allowed"])
	}
	if !beforeAllowed {
		t.Fatal("expected bob can_read=true before delete")
	}

	deletePayload, err := json.Marshal(map[string]any{
		"tuples": []map[string]string{{
			"object":   string(fixtures.RoadmapDocument),
			"relation": string(rebac.RelationDocumentWorkspace),
			"user":     string(fixtures.ProductWorkspace),
		}},
	})
	if err != nil {
		t.Fatalf("marshal delete payload: %v", err)
	}
	deleteReq := httptest.NewRequest(http.MethodDelete, "/tuples", bytes.NewReader(deletePayload))
	deleteReq.Header.Set("Content-Type", "application/json")
	deleteRec := httptest.NewRecorder()
	// Act
	handler.ServeHTTP(deleteRec, deleteReq)

	// Assert
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete tuples: expected 200, got %d", deleteRec.Code)
	}

	afterPayload, err := json.Marshal(map[string]string{
		"user": string(fixtures.Bob), "relation": string(rebac.RelationDocumentCanRead), "object": string(fixtures.RoadmapDocument),
	})
	if err != nil {
		t.Fatalf("marshal after-check payload: %v", err)
	}
	afterReq := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader(afterPayload))
	afterReq.Header.Set("Content-Type", "application/json")
	afterRec := httptest.NewRecorder()
	// Act
	handler.ServeHTTP(afterRec, afterReq)
	var afterResp map[string]any
	if err := json.NewDecoder(afterRec.Body).Decode(&afterResp); err != nil {
		t.Fatalf("decode after-check response: %v", err)
	}
	afterAllowed, ok := afterResp["allowed"].(bool)
	if !ok {
		t.Fatalf("after allowed = %T, want bool", afterResp["allowed"])
	}
	if afterAllowed {
		t.Error("expected bob can_read=false after deleting workspace tuple")
	}
}

func TestAuthzHandler_WriteTuples_InvalidTupleReturns422(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	svc := authz.New(store, authz.NewGraphEvaluator(store))
	handler := authzhttp.NewServer(svc)

	// "roadmap" is non-empty (so it passes the handler's required-field check) but
	// is not a valid "type:id" object, so domain validation rejects it. That maps
	// to 422 Unprocessable Entity — the request was understood but is invalid.
	payload, err := json.Marshal(map[string]any{
		"tuples": []map[string]string{{
			"object":   "roadmap",
			"relation": string(rebac.RelationDocumentOwner),
			"user":     string(fixtures.Alice),
		}},
	})
	if err != nil {
		t.Fatalf("marshal write payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/tuples", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthzHandler_ListTuples_ReturnsAllTuples(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	svc := authz.New(store, authz.NewGraphEvaluator(store))
	handler := authzhttp.NewServer(svc) // 4 seed tuples from fixtures
	req := httptest.NewRequest(http.MethodGet, "/tuples", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
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
