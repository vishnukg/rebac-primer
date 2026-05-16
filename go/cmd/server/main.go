// Command server starts the Go ReBAC HTTP server.
//
// Usage:
//
//	PORT=4001 go run ./cmd/server
//
// The server listens on PORT (default 4001) and exposes:
//
//	GET  /health
//	POST /documents
//	GET  /documents/{id}?actorId=...
//	PATCH /documents/{id}
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"rebac-primer/internal/app"
)

func main() {
	ctx := context.Background()

	a, err := app.New(ctx)
	if err != nil {
		log.Fatalf("failed to initialise app: %v", err)
	}

	addr := fmt.Sprintf(":%d", a.Port)
	log.Printf("Go ReBAC server listening on http://127.0.0.1%s", addr)

	if err := http.ListenAndServe(addr, a.Handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
