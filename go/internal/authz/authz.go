// Package authz is the authz service's core.
//
// It defines the [Service] driving port and the driven ports ([TupleRepository],
// [Evaluator]) that adapters must satisfy.  The concrete implementation lives in
// [New] — everything else in this package is an interface or a domain type.
//
// Hexagonal architecture in one diagram:
//
//	                    ┌──────────────────────────────┐
//	   driving adapters │           authz              │  driven adapters
//	   (HTTP handler)   │                              │  (db, graph, openfga)
//	        ───────────►│  Service                     │
//	                    │    Check()                   │──►  Evaluator
//	                    │    WriteTuples()             │
//	                    │    DeleteTuples()            │──►  TupleRepository
//	                    │    ListTuples()              │
//	                    └──────────────────────────────┘
//
// Mirrors typescript/src/authz-service/core/domain/types.ts
// and typescript/src/authz-service/core/ports/.
package authz

import (
	"context"
	"fmt"

	"rebac-primer/internal/shared"
)

// ── Driving port ──────────────────────────────────────────────────────────────

// Service is the driving port — what HTTP handlers, tests, and other services
// call into.  The concrete implementation is returned by [New].
//
// Mirrors typescript/src/authz-service/core/domain/types.ts (AuthzService).
type Service interface {
	Check(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error)
	WriteTuples(ctx context.Context, tuples []shared.TupleKey) error
	DeleteTuples(ctx context.Context, tuples []shared.TupleKey) error
	ListTuples(ctx context.Context, filter ...TupleFilter) ([]shared.TupleKey, error)
}

// ── Domain errors ─────────────────────────────────────────────────────────────

// TupleValidationError signals that a tuple contains semantically invalid data.
// The HTTP adapter maps this to 422 Unprocessable Entity.
//
// Mirrors typescript/src/authz-service/core/domain/types.ts (TupleValidationError).
type TupleValidationError struct {
	Message string
}

func (e *TupleValidationError) Error() string {
	return fmt.Sprintf("tuple validation: %s", e.Message)
}
