package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rebac-primer/internal/app"
)

// newTestHandler builds a fully-wired http.Handler via app.New.
// It uses the same fixture store and demo document as production, which makes
// these integration-level tests: they exercise the full stack (authz → domain → HTTP)
// without starting a real network listener.
func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	a, err := app.New(context.Background())
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	return a.Handler
}

func TestHandler_Health(t *testing.T) {
	// Arrange
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	var body map[string]bool
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body["ok"] {
		t.Errorf("expected {\"ok\":true}, got %v", body)
	}
}

func TestHandler_CreateDocument_Returns201ForEditor(t *testing.T) {
	// Arrange
	handler := newTestHandler(t)
	payload := map[string]string{
		"id":          "testDoc",
		"title":       "Test Document",
		"body":        "Hello, world",
		"workspaceId": "productWorkspace",
		"actorId":     "alice",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["document"]; !ok {
		t.Errorf("expected response to contain 'document' key, got %v", resp)
	}
}

func TestHandler_CreateDocument_Returns400WhenFieldsMissing(t *testing.T) {
	// Arrange: title and body are absent from the request.
	handler := newTestHandler(t)
	payload := map[string]string{
		"id":          "oops",
		"workspaceId": "productWorkspace",
		"actorId":     "alice",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_GetDocument_Returns200ForViewer(t *testing.T) {
	// Arrange: roadmapDocument is pre-seeded; bob has can_read via the graph.
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/documents/roadmapDocument?actorId=bob", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	doc, ok := resp["document"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'document' to be an object, got %T", resp["document"])
	}
	if doc["id"] != "roadmapDocument" {
		t.Errorf("expected id=%q, got %v", "roadmapDocument", doc["id"])
	}
}

func TestHandler_GetDocument_Returns403ForOutsider(t *testing.T) {
	// Arrange: casey has no tuples — every check must deny.
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/documents/roadmapDocument?actorId=casey", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_GetDocument_Returns400WhenActorMissing(t *testing.T) {
	// Arrange: actorId query parameter is omitted.
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/documents/roadmapDocument", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_PatchDocument_Returns403ForViewer(t *testing.T) {
	// Arrange: bob has viewer, not editor — PATCH must be denied.
	handler := newTestHandler(t)
	payload := map[string]string{
		"body":    "should not save",
		"actorId": "bob",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/documents/roadmapDocument", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_PatchDocument_Returns200ForEditor(t *testing.T) {
	// Arrange
	handler := newTestHandler(t)
	payload := map[string]string{
		"body":    "updated by editor",
		"actorId": "alice",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/documents/roadmapDocument", bytes.NewReader(body))
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
		t.Fatalf("decode response: %v", err)
	}
	doc, ok := resp["document"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'document' to be an object, got %T", resp["document"])
	}
	if doc["body"] != "updated by editor" {
		t.Errorf("expected body=%q, got %v", "updated by editor", doc["body"])
	}
}
