// Command server starts the Go ReBAC HTTP server.
//
// Usage:
//
//	PORT=4001 go run ./cmd/server
//
// The server listens on PORT (default 4001) and exposes:
//
//	GET   /health
//	GET   /whoami
//	POST  /documents
//	GET   /documents/{id}
//	PATCH /documents/{id}
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"rebac-primer/internal/authz"
	authzdb "rebac-primer/internal/authz/adapters/db"
	"rebac-primer/internal/authz/adapters/graph"
	"rebac-primer/internal/documents"
	docsauthn "rebac-primer/internal/documents/adapters/authn"
	docsdb "rebac-primer/internal/documents/adapters/db"
	docshttp "rebac-primer/internal/documents/adapters/http"
	"rebac-primer/internal/fixtures"
)

func main() {
	ctx := context.Background()

	port, err := readPort()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	handler, err := buildHandler(ctx)
	if err != nil {
		log.Fatalf("init: %v", err)
	}

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Go ReBAC server listening on http://127.0.0.1%s", addr)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// readPort reads PORT from the environment (default 4001).
func readPort() (int, error) {
	v := os.Getenv("PORT")
	if v == "" {
		return 4001, nil
	}
	p, err := strconv.Atoi(v)
	if err != nil || p < 1 || p > 65535 {
		return 0, fmt.Errorf("invalid PORT %q", v)
	}
	return p, nil
}

// buildHandler wires together all adapters and returns the HTTP handler.
//
// Wiring order:
//  1. Authz:     tuple store → graph evaluator → authz.Service
//  2. Documents: repo + token verifier + authz service → documents.Service
//  3. HTTP:      documents server wraps the domain
func buildHandler(ctx context.Context) (http.Handler, error) {
	// ── Authz service ──────────────────────────────────────────────────────────
	// In-memory tuple store pre-seeded with fixture relationships.
	// To use a real OpenFGA server instead, swap the evaluator for
	// openfga.New(cfg) — domain code is unchanged.
	tupleStore := authzdb.New(fixtures.SeedRelationshipTuples()...)
	evaluator := graph.NewGraphEvaluator(tupleStore)
	authzSvc := authz.New(tupleStore, evaluator)

	// ── Documents service ──────────────────────────────────────────────────────
	docRepo := docsdb.New()
	tokenVerifier := docsauthn.New(fixtures.DemoTokens())
	docsSvc := documents.New(docRepo, authzSvc)

	// Seed demo document so GET /documents/roadmapDocument works out of the box.
	_, err := docsSvc.Create(ctx, documents.CreateDocumentInput{
		ID:        "roadmapDocument",
		Title:     "Roadmap",
		Body:      "Initial roadmap document",
		Workspace: fixtures.ProductWorkspace,
		Actor:     fixtures.Alice,
	})
	if err != nil {
		return nil, fmt.Errorf("seed demo document: %w", err)
	}

	// ── HTTP layer ─────────────────────────────────────────────────────────────
	return docshttp.NewServer(tokenVerifier, docsSvc), nil
}
