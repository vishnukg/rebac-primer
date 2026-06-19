// Package authzhttp is a teaching example showing the authz service exposed over
// HTTP — the client/server seam for ReBAC. It is NOT wired into cmd/server (which
// calls the authz service in-process); see docs/33-client-server-rebac.md.
//
// Routes:
//
//	GET    /health
//	POST   /check      { user, relation, object }           → { allowed, trace }
//	POST   /tuples     { tuples: [{object,relation,user}] } → { written }
//	DELETE /tuples     { tuples: [{object,relation,user}] } → { deleted }
//	GET    /tuples     ?object=...&relation=...             → { tuples }
//
// Product services call POST /check to ask "can this user do that?".
// Product services call POST /tuples when relationships change.
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
	WriteTuples(ctx context.Context, tuples []rebac.TupleKey) error
	DeleteTuples(ctx context.Context, tuples []rebac.TupleKey) error
	ListTuples(ctx context.Context, filter ...authz.TupleFilter) ([]rebac.TupleKey, error)
}

// NewServer returns an http.Handler with all authz routes registered.
func NewServer(svc AuthorizationService) http.Handler {
	h := &handler{authz: svc}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("POST /check", h.handleCheck)
	mux.HandleFunc("POST /tuples", h.handleWriteTuples)
	mux.HandleFunc("DELETE /tuples", h.handleDeleteTuples)
	mux.HandleFunc("GET /tuples", h.handleListTuples)

	return mux
}
