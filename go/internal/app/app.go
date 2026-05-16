// Package app is the composition root. It wires together the authz, domain,
// and httpserver layers and seeds the store with demo data.
//
// This is the Go equivalent of typescript/src/app/create-services.ts and
// create-server.ts: one place that knows every concrete type so the rest of
// the codebase can depend only on interfaces.
package app

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/domain"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/httpserver"
)

// App holds the fully-wired HTTP handler and the port to listen on.
type App struct {
	Handler http.Handler
	Port    int
}

// New creates and seeds the application. It:
//  1. Builds the in-memory tuple store and seeds it with fixture tuples.
//  2. Creates the graph authorizer and domain service.
//  3. Creates a demo "Roadmap" document.
//  4. Returns an http.Handler ready to serve requests.
func New(ctx context.Context) (*App, error) {
	// --- authz layer ---
	tupleStore := authz.NewInMemoryTupleStore(fixtures.SeedRelationshipTuples()...)
	authorizer := authz.NewGraphAuthorizer(tupleStore)

	// --- domain layer ---
	repo := domain.NewInMemoryDocumentRepository()
	docs := domain.NewDocumentService(repo, authorizer)

	// --- seed demo document ---
	_, err := docs.Create(ctx, domain.CreateDocumentInput{
		ID:        "roadmapDocument",
		Title:     "Roadmap",
		Body:      "Initial roadmap document",
		Workspace: fixtures.ProductWorkspace,
		Actor:     fixtures.WorkspaceEditor,
	})
	if err != nil {
		return nil, fmt.Errorf("app: seed demo document: %w", err)
	}

	// --- HTTP layer ---
	handler := httpserver.NewServer(docs)

	port := 4001
	if v := os.Getenv("PORT"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &port); err != nil {
			return nil, fmt.Errorf("app: invalid PORT %q: %w", v, err)
		}
	}

	return &App{Handler: handler, Port: port}, nil
}
