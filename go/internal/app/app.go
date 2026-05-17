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
	"strconv"

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

// Config contains runtime configuration supplied by the environment.
type Config struct {
	Port int
}

// DefaultConfig returns local-development defaults.
func DefaultConfig() Config {
	return Config{Port: 4001}
}

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

// New creates and seeds the application. It:
//  1. Builds the in-memory tuple store and seeds it with fixture tuples.
//  2. Creates the graph authorizer and domain service.
//  3. Creates a demo "Roadmap" document.
//  4. Returns an http.Handler ready to serve requests.
func New(ctx context.Context) (*App, error) {
	cfg, err := ConfigFromEnv(os.Getenv)
	if err != nil {
		return nil, err
	}
	return NewWithConfig(ctx, cfg)
}

// NewWithConfig creates and seeds the application with explicit configuration.
func NewWithConfig(ctx context.Context, cfg Config) (*App, error) {
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
		Actor:     fixtures.Alice,
	})
	if err != nil {
		return nil, fmt.Errorf("app: seed demo document: %w", err)
	}

	// --- HTTP layer ---
	handler := httpserver.NewServer(docs)

	return &App{Handler: handler, Port: cfg.Port}, nil
}
