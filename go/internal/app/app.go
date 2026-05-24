// Package app is the composition root.  It wires together the authz service,
// documents service, and HTTP layer and seeds the store with demo data.
//
// This is the single place in the codebase that knows every concrete type.
// Everything else depends only on interfaces (ports).
//
// Mirrors the TypeScript compose.ts files in each service directory.
package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	authzdb "rebac-primer/internal/authzservice/adapters/db"
	"rebac-primer/internal/authzservice/adapters/graph"
	authzdomain "rebac-primer/internal/authzservice/core/domain"
	docsauthn "rebac-primer/internal/documentsservice/adapters/authn"
	docsdb "rebac-primer/internal/documentsservice/adapters/db"
	docshttp "rebac-primer/internal/documentsservice/adapters/http"
	docsdomain "rebac-primer/internal/documentsservice/core/domain"
	"rebac-primer/internal/fixtures"
)

// App holds the fully-wired HTTP handler and the port to listen on.
type App struct {
	Handler http.Handler
	Port    int
}

// Config contains runtime configuration supplied by the environment.
type Config struct {
	Port int
}

// DefaultConfig returns local-development defaults.
func DefaultConfig() Config { return Config{Port: 4001} }

// ConfigFromEnv reads 12-factor style process configuration.
func ConfigFromEnv(lookup func(string) string) (Config, error) {
	cfg := DefaultConfig()
	if value := lookup("PORT"); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("app: invalid PORT %q", value)
		}
		cfg.Port = port
	}
	return cfg, nil
}

// New creates and seeds the application from environment variables.
func New(ctx context.Context) (*App, error) {
	cfg, err := ConfigFromEnv(os.Getenv)
	if err != nil {
		return nil, err
	}
	return NewWithConfig(ctx, cfg)
}

// NewWithConfig creates and seeds the application with explicit configuration.
//
// Wiring order mirrors the TypeScript compose.ts files:
//  1. Authz service: tuple store → graph evaluator → authz domain
//  2. Documents service: repo + authn + authz domain as AuthzClient → documents domain
//  3. HTTP server wraps the documents domain
func NewWithConfig(ctx context.Context, cfg Config) (*App, error) {
	// ── Authz service ──────────────────────────────────────────────────────────
	// adapter: in-memory tuple store, pre-seeded with fixture relationships
	tupleStore := authzdb.New(fixtures.SeedRelationshipTuples()...)

	// adapter: in-process graph evaluator
	evaluator := graph.NewGraphEvaluator(tupleStore)

	// domain: wire repository + evaluator into the AuthzService
	authzSvc := authzdomain.New(tupleStore, evaluator)

	// ── Documents service ──────────────────────────────────────────────────────
	// adapter: in-memory document repository
	docRepo := docsdb.New()

	// adapter: demo token verifier (AuthzService satisfies AuthzClient structurally)
	tokenVerifier := docsauthn.New(fixtures.DemoTokens())

	// domain: authzSvc satisfies ports.AuthzClient via Go structural typing
	// (it has Check and WriteTuples with matching signatures)
	docs := docsdomain.New(docRepo, authzSvc)

	// seed demo document
	_, err := docs.Create(ctx, docsdomain.CreateDocumentInput{
		ID:        "roadmapDocument",
		Title:     "Roadmap",
		Body:      "Initial roadmap document",
		Workspace: fixtures.ProductWorkspace,
		Actor:     fixtures.Alice,
	})
	if err != nil {
		return nil, fmt.Errorf("app: seed demo document: %w", err)
	}

	// ── HTTP layer ─────────────────────────────────────────────────────────────
	httpHandler := docshttp.NewServer(tokenVerifier, docs)

	return &App{Handler: httpHandler, Port: cfg.Port}, nil
}
