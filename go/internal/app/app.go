// Package app is the composition root.  It wires together every concrete type
// and returns an [App] holding the fully-configured HTTP handler.
//
// This is the single place in the codebase that knows about concrete adapters.
// Everything else depends only on interfaces (ports).
//
// Wiring order (mirrors the TypeScript compose.ts files in each service):
//
//  1. Authz service:   tuple store → graph evaluator → authz.New()
//  2. Documents service: repo + token verifier + authz service → documents.New()
//  3. HTTP layer:      documents server wraps the domain
//
// Mirrors typescript/src/authz-service/compose.ts
// and   typescript/src/documents-service/compose.ts.
package app

import (
	"context"
	"fmt"
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
func NewWithConfig(ctx context.Context, cfg Config) (*App, error) {
	// ── Authz service ──────────────────────────────────────────────────────────
	// adapter: in-memory tuple store, pre-seeded with fixture relationships
	tupleStore := authzdb.New(fixtures.SeedRelationshipTuples()...)

	// adapter: in-process graph evaluator (driven port: authz.Evaluator)
	evaluator := graph.NewGraphEvaluator(tupleStore)

	// domain: wire store + evaluator into the authz.Service driving port
	authzSvc := authz.New(tupleStore, evaluator)

	// ── Documents service ──────────────────────────────────────────────────────
	// adapter: in-memory document repository (driven port: documents.DocumentRepository)
	docRepo := docsdb.New()

	// adapter: demo token verifier (driven port: documents.Authenticator)
	tokenVerifier := docsauthn.New(fixtures.DemoTokens())

	// domain: authzSvc satisfies documents.AuthzClient via Go structural typing
	// (it has Check and WriteTuples with the same signatures).
	//
	// In a distributed deployment, swap authzSvc for an HTTP client:
	//   authzclient.NewClient("http://127.0.0.1:4100")
	// Domain code is unchanged — only this wiring line differs.
	docsSvc := documents.New(docRepo, authzSvc)

	// seed demo document
	_, err := docsSvc.Create(ctx, documents.CreateDocumentInput{
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
	// driving adapter: HTTP server wraps the documents domain
	httpHandler := docshttp.NewServer(tokenVerifier, docsSvc)

	return &App{Handler: httpHandler, Port: cfg.Port}, nil
}
