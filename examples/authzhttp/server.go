// Package authzhttp is a teaching example showing the authz service exposed over
// HTTP — the client/server seam for ReBAC. It is NOT wired into cmd/server (which
// calls the authz service in-process); see docs/33-client-server-rebac.md.
//
// Routes:
//
//	GET    /health
//	POST   /check           { subject, permission, resource }                   → { allowed, trace }
//	POST   /relationships   { relationships: [{subject,relation,resource}] }    → { written }
//	DELETE /relationships   { relationships: [{subject,relation,resource}] }    → { deleted }
//	GET    /relationships   ?resource=...&relation=...                          → { relationships }
//
// Product services call POST /check to ask "can this subject do that?".
// Product services call POST /relationships when relationships change.
//
// No external router framework is used — Go 1.22+ ServeMux handles
// method+path patterns like "POST /check" natively.
package authzhttp

import (
	"context"
	"net/http"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/rebac"
)

// AuthorizationService is the capability exposed by this HTTP adapter.
// It is intentionally declared by the consumer rather than by either backend.
type AuthorizationService interface {
	Check(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error)
	WriteRelationships(ctx context.Context, relationships []rebac.Relationship) error
	DeleteRelationships(ctx context.Context, relationships []rebac.Relationship) error
	ListRelationships(ctx context.Context, filter ...authz.RelationshipFilter) ([]rebac.Relationship, error)
}

// NewServer returns an http.Handler with all authz routes registered.
func NewServer(svc AuthorizationService) http.Handler {
	h := &handler{authz: svc}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("POST /check", h.handleCheck)
	mux.HandleFunc("POST /relationships", h.handleWriteRelationships)
	mux.HandleFunc("DELETE /relationships", h.handleDeleteRelationships)
	mux.HandleFunc("GET /relationships", h.handleListRelationships)

	return mux
}
