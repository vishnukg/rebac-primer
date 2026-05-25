// Package authz provides an HTTP client adapter for the authz service.
//
// Client adapts HTTP calls to the authz service to satisfy the
// [documents.AuthzClient] interface.  In a distributed deployment the documents
// service uses this instead of the in-process authz service — domain code is
// unchanged because both satisfy the same interface.
//
// In production:
//
//	documents.New(repo, authzclient.NewClient("http://authz-service:4100"))
//
// In the monolith (app.go):
//
//	documents.New(repo, authzSvc) // in-process, no network hop
//
// Mirrors typescript/src/documents-service/adapters/authz/makeAuthzServiceClient.ts.
package authz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"rebac-primer/internal/documents"
	"rebac-primer/internal/shared"
)

// Client calls the authz service over HTTP and satisfies [documents.AuthzClient].
// Construct with [NewClient]; do not use the zero value directly.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a Client that calls the authz service at baseURL.
// Example: authz.NewClient("http://127.0.0.1:4100")
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// Compile-time assertion: *Client must satisfy the documents service AuthzClient port.
var _ documents.AuthzClient = (*Client)(nil)

// Check calls POST /check on the authz service.
func (c *Client) Check(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error) {
	body := map[string]string{
		"user":     string(req.User),
		"relation": string(req.Relation),
		"object":   string(req.Object),
	}

	var result struct {
		Allowed bool     `json:"allowed"`
		Trace   []string `json:"trace"`
	}
	if err := c.post(ctx, "/check", body, &result); err != nil {
		return shared.CheckResult{}, err
	}
	return shared.CheckResult{Allowed: result.Allowed, Trace: result.Trace}, nil
}

// WriteTuples calls POST /tuples on the authz service.
func (c *Client) WriteTuples(ctx context.Context, tuples []shared.TupleKey) error {
	type tupleJSON struct {
		Object   string `json:"object"`
		Relation string `json:"relation"`
		User     string `json:"user"`
	}
	items := make([]tupleJSON, 0, len(tuples))
	for _, t := range tuples {
		items = append(items, tupleJSON{
			Object:   string(t.Object),
			Relation: string(t.Relation),
			User:     string(t.User),
		})
	}
	return c.post(ctx, "/tuples", map[string]any{"tuples": items}, nil)
}

// ── Private helpers ───────────────────────────────────────────────────────────

// post marshals body as JSON, POSTs to baseURL+path, and decodes the response
// into result (if non-nil).  Returns an error on non-2xx status.
func (c *Client) post(ctx context.Context, path string, body, result any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("authz client: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("authz client: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("authz client %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		if errBody.Error != "" {
			return fmt.Errorf("authz service error: %s", errBody.Error)
		}
		return fmt.Errorf("authz service %s: unexpected status %d", path, resp.StatusCode)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("authz client: decode response: %w", err)
		}
	}
	return nil
}
