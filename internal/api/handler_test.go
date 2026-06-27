package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rebac-primer/internal/api"
	"rebac-primer/internal/authz"
	"rebac-primer/internal/documents"
	"rebac-primer/internal/fixtures"
)

// newTestHandler builds a fully-wired http.Handler.
// These are integration-level tests: they exercise the full stack
// (authn → authz → domain → HTTP) without starting a real server.
func newTestHandler(t *testing.T) http.Handler {
	return newTestHandlerWithTokens(t, fixtures.DemoTokens())
}

func newTestHandlerWithTokens(t *testing.T, tokens map[string]documents.TokenClaims) http.Handler {
	t.Helper()

	tupleStore := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	evaluator := authz.NewGraphEvaluator(tupleStore)
	authzSvc := authz.New(tupleStore, evaluator)

	docRepo := documents.NewInMemoryRepository()
	tokenVerifier := documents.NewDemoTokenVerifier(tokens)
	docsSvc := documents.New(docRepo, authzSvc)

	_, err := docsSvc.Create(context.Background(), documents.CreateDocumentInput{
		ID:        "roadmapDocument",
		Title:     "Roadmap",
		Body:      "Initial roadmap document",
		Workspace: fixtures.ProductWorkspace,
		Actor:     fixtures.Alice,
	})
	if err != nil {
		t.Fatalf("seed demo document: %v", err)
	}

	return api.NewServer(tokenVerifier, docsSvc)
}

func TestHandler_Health(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

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

func TestHandler_Whoami_Returns200WithValidToken(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer demo-token-alice")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["user"] != "user:alice" {
		t.Errorf("expected user=user:alice, got %v", body["user"])
	}
}

func TestHandler_Whoami_Returns401WithMissingToken(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandler_CreateDocument_Returns201ForEditor(t *testing.T) {
	handler := newTestHandler(t)
	payload := map[string]string{
		"id":          "testDoc",
		"title":       "Test Document",
		"body":        "Hello, world",
		"workspaceId": "productWorkspace",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer demo-token-alice")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

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

func TestHandler_CreateDocument_Returns409ForExistingID(t *testing.T) {
	handler := newTestHandler(t)
	payload := map[string]string{
		"id":          "roadmapDocument", // seeded by newTestHandler
		"title":       "Duplicate",
		"body":        "should not overwrite",
		"workspaceId": "productWorkspace",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer demo-token-alice")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_CreateDocument_Returns400ForBlankFields(t *testing.T) {
	cases := map[string]map[string]string{
		"id":          {"id": " ", "title": "Test", "body": "Body", "workspaceId": "productWorkspace"},
		"title":       {"id": "testDoc", "title": "\t", "body": "Body", "workspaceId": "productWorkspace"},
		"body":        {"id": "testDoc", "title": "Test", "body": "\n", "workspaceId": "productWorkspace"},
		"workspaceId": {"id": "testDoc", "title": "Test", "body": "Body", "workspaceId": " "},
	}

	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			handler := newTestHandler(t)
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer demo-token-alice")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d — body: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandler_CreateDocument_Returns401WhenTokenMissing(t *testing.T) {
	handler := newTestHandler(t)
	payload := map[string]string{
		"id": "testDoc", "title": "Test", "body": "Body", "workspaceId": "productWorkspace",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandler_CreateDocument_Returns400ForUnknownJSONField(t *testing.T) {
	handler := newTestHandler(t)
	payload := []byte(`{"id":"testDoc","title":"Test","body":"Body","workspaceId":"productWorkspace","extra":true}`)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer demo-token-alice")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_CreateDocument_Returns415ForUnsupportedMediaType(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Authorization", "Bearer demo-token-alice")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_CreateDocument_Returns413ForOversizedBody(t *testing.T) {
	handler := newTestHandler(t)
	payload := append([]byte(`{"id":"`), bytes.Repeat([]byte("x"), (1<<20)+1)...)
	payload = append(payload, []byte(`","title":"Test","body":"Body","workspaceId":"productWorkspace"}`)...)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer demo-token-alice")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_PatchDocument_Returns400ForMultipleJSONValues(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPatch, "/documents/roadmapDocument", bytes.NewReader([]byte(`{"body":"updated"} {}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer demo-token-alice")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_PatchDocument_Returns400ForBlankBody(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPatch, "/documents/roadmapDocument", bytes.NewReader([]byte(`{"body":" "}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer demo-token-alice")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_GetDocument_Returns200ForViewer(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/documents/roadmapDocument", nil)
	req.Header.Set("Authorization", "Bearer demo-token-bob")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

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

func TestHandler_GetDocument_Returns401WhenTokenMissing(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/documents/roadmapDocument", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandler_GetDocument_Returns403ForOutsider(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/documents/roadmapDocument", nil)
	req.Header.Set("Authorization", "Bearer demo-token-casey")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_PatchDocument_Returns403WhenWriteScopeMissing(t *testing.T) {
	handler := newTestHandler(t)
	payload := map[string]string{"body": "should not save"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/documents/roadmapDocument", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer demo-token-bob")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("WWW-Authenticate"); got != `Bearer error="insufficient_scope", scope="documents:write"` {
		t.Errorf("WWW-Authenticate = %q, want insufficient_scope challenge", got)
	}
}

func TestHandler_PatchDocument_Returns403WhenReBACDeniesViewerWithWriteScope(t *testing.T) {
	tokens := fixtures.DemoTokens()
	tokens["demo-token-bob"] = documents.TokenClaims{
		Sub:    "bob",
		Scopes: []string{"documents:read", "documents:write"},
	}
	handler := newTestHandlerWithTokens(t, tokens)
	payload := map[string]string{"body": "should not save"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/documents/roadmapDocument", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer demo-token-bob")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("WWW-Authenticate"); got != "" {
		t.Errorf("WWW-Authenticate = %q, want empty for a ReBAC denial", got)
	}
}

func TestHandler_GetDocument_Returns403WhenReadScopeMissing(t *testing.T) {
	tupleStore := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	authzSvc := authz.New(tupleStore, authz.NewGraphEvaluator(tupleStore))
	docsSvc := documents.New(documents.NewInMemoryRepository(), authzSvc)
	_, err := docsSvc.Create(context.Background(), documents.CreateDocumentInput{
		ID: "roadmapDocument", Title: "Roadmap", Body: "body",
		Workspace: fixtures.ProductWorkspace, Actor: fixtures.Alice,
	})
	if err != nil {
		t.Fatalf("seed document: %v", err)
	}
	verifier := documents.NewDemoTokenVerifier(map[string]documents.TokenClaims{
		"write-only": {Sub: "alice", Scopes: []string{"documents:write"}},
	})
	handler := api.NewServer(verifier, docsSvc)

	req := httptest.NewRequest(http.MethodGet, "/documents/roadmapDocument", nil)
	req.Header.Set("Authorization", "Bearer write-only")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_PatchDocument_Returns200ForEditor(t *testing.T) {
	handler := newTestHandler(t)
	payload := map[string]string{"body": "updated by editor"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/documents/roadmapDocument", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer demo-token-alice")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

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
