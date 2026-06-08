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

	// Choose the authorization backend from the environment. Both branches yield
	// an authz.Service, so everything downstream is identical either way.
	var authzService authz.Service
	if os.Getenv("AUTHZ_BACKEND") == "openfga" {
		cfg := openfga.Config{
			APIURL:  envOr("OPENFGA_API_URL", "http://127.0.0.1:8080"),
			StoreID: os.Getenv("OPENFGA_STORE_ID"),
			ModelID: os.Getenv("OPENFGA_MODEL_ID"),
		}
		if cfg.StoreID == "" || cfg.ModelID == "" {
			log.Fatalf("AUTHZ_BACKEND=openfga requires OPENFGA_STORE_ID and OPENFGA_MODEL_ID (run deployments/openfga/seed.sh)")
		}
		authzService, err = openfga.New(cfg)
		if err != nil {
			log.Fatalf("openfga backend: %v", err)
		}
		log.Printf("authz backend: openfga (%s, store=%s)", cfg.APIURL, cfg.StoreID)
	} else {
		store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
		authzService = authz.New(store, authz.NewGraphEvaluator(store))
		log.Printf("authz backend: in-process graph evaluator")
	}

	// Wire the documents service over the chosen authz backend.
	documentsService := documents.New(documents.NewInMemoryRepository(), authzService)
	verifier := documents.NewDemoTokenVerifier(fixtures.DemoTokens())

	// Seed the demo document so GET /documents/roadmapDocument works out of the box.
	// In OpenFGA mode this also writes the document's tuples to the store, which
	// requires the policy tuples to be seeded first (deployments/openfga/seed.sh).
	if _, err := documentsService.Create(ctx, documents.CreateDocumentInput{
		ID:        "roadmapDocument",
		Title:     "Roadmap",
		Body:      "Initial roadmap document",
		Workspace: fixtures.ProductWorkspace,
		Actor:     fixtures.Alice,
	}); err != nil {
		log.Fatalf("seed demo document: %v", err)
	}

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Go ReBAC server listening on http://127.0.0.1%s", addr)
	if err := http.ListenAndServe(addr, api.NewServer(verifier, documentsService)); err != nil {
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

// envOr returns the value of the environment variable, or fallback when unset/empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
