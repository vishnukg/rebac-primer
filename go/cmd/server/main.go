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

	"rebac-primer/internal/api"
	"rebac-primer/internal/authz"
	"rebac-primer/internal/documents"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/openfga"
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
	// The backend is chosen by AUTHZ_BACKEND. Either implementation is an
	// authz.Service, so the documents domain below is identical for both.
	authzSvc, err := buildAuthzService()
	if err != nil {
		return nil, err
	}

	// ── Documents service ──────────────────────────────────────────────────────
	docRepo := documents.NewInMemoryRepository()
	tokenVerifier := documents.NewDemoTokenVerifier(fixtures.DemoTokens())
	docsSvc := documents.New(docRepo, authzSvc)

	// Seed demo document so GET /documents/roadmapDocument works out of the box.
	// In OpenFGA mode this writes the document's tuples to the OpenFGA store and
	// requires the policy tuples to be seeded first (deployments/openfga/seed.sh).
	if _, err = docsSvc.Create(ctx, documents.CreateDocumentInput{
		ID:        "roadmapDocument",
		Title:     "Roadmap",
		Body:      "Initial roadmap document",
		Workspace: fixtures.ProductWorkspace,
		Actor:     fixtures.Alice,
	}); err != nil {
		return nil, fmt.Errorf("seed demo document: %w", err)
	}

	// ── HTTP layer ─────────────────────────────────────────────────────────────
	return api.NewServer(tokenVerifier, docsSvc), nil
}

// buildAuthzService selects the authorization backend from the environment:
//
//	AUTHZ_BACKEND=openfga  → talk to a real OpenFGA server (OPENFGA_API_URL,
//	                         OPENFGA_STORE_ID, OPENFGA_MODEL_ID)
//	otherwise (default)    → the in-process graph evaluator over an in-memory
//	                         tuple store seeded with the fixture policy tuples
//
// Both return an authz.Service, so nothing downstream changes between backends.
func buildAuthzService() (authz.Service, error) {
	if os.Getenv("AUTHZ_BACKEND") == "openfga" {
		cfg := openfga.Config{
			APIURL:  envOr("OPENFGA_API_URL", "http://127.0.0.1:8080"),
			StoreID: os.Getenv("OPENFGA_STORE_ID"),
			ModelID: os.Getenv("OPENFGA_MODEL_ID"),
		}
		if cfg.StoreID == "" || cfg.ModelID == "" {
			return nil, fmt.Errorf("AUTHZ_BACKEND=openfga requires OPENFGA_STORE_ID and OPENFGA_MODEL_ID (run deployments/openfga/seed.sh)")
		}
		svc, err := openfga.New(cfg)
		if err != nil {
			return nil, fmt.Errorf("openfga backend: %w", err)
		}
		log.Printf("authz backend: openfga (%s, store=%s)", cfg.APIURL, cfg.StoreID)
		return svc, nil
	}

	tupleStore := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	authzSvc := authz.New(tupleStore, authz.NewGraphEvaluator(tupleStore))
	log.Printf("authz backend: in-process graph evaluator")
	return authzSvc, nil
}

// envOr returns the value of the environment variable, or fallback when unset/empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
