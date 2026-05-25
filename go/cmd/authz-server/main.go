// Command authz-server starts the authz service as a standalone HTTP server.
//
// This mirrors typescript/src/authz-service/compose.ts — when run as a separate
// process the documents service connects to it via the authz HTTP client adapter
// instead of using the in-process authz service.
//
// Usage:
//
//	PORT=4100 go run ./cmd/authz-server
//
// The server listens on PORT (default 4100) and exposes:
//
//	GET    /health
//	POST   /check      { user, relation, object }           → { allowed, trace }
//	POST   /tuples     { tuples: [{object,relation,user}] } → { written }
//	DELETE /tuples     { tuples: [{object,relation,user}] } → { deleted }
//	GET    /tuples     ?object=...&relation=...             → { tuples }
//
// In the default monolithic setup (cmd/server) the authz service runs in-process.
// Run this binary when you want the two services to communicate over HTTP.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"rebac-primer/internal/authz"
	authzdb "rebac-primer/internal/authz/adapters/db"
	"rebac-primer/internal/authz/adapters/graph"
	authzhttp "rebac-primer/internal/authz/adapters/http"
	"rebac-primer/internal/fixtures"
)

func main() {
	port := readPort()

	// ── Wire: tuple store → graph evaluator → authz service → HTTP server ──────
	// Mirrors the TypeScript composeAuthzService() wiring in compose.ts.

	// adapter: in-memory tuple store, pre-seeded with fixture relationships
	store := authzdb.New(fixtures.SeedRelationshipTuples()...)

	// adapter: in-process graph evaluator (driven port: authz.Evaluator)
	evaluator := graph.NewGraphEvaluator(store)

	// domain: wire store + evaluator into the authz.Service driving port
	svc := authz.New(store, evaluator)

	// driving adapter: HTTP server wraps the service
	handler := authzhttp.NewServer(svc)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Authz service listening on http://127.0.0.1%s", addr)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func readPort() int {
	if v := os.Getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			log.Fatalf("invalid PORT %q", v)
		}
		return p
	}
	return 4100
}
