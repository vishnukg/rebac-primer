package authz

import (
	"context"

	"rebac-primer/internal/shared"
)

// ── Driven ports ──────────────────────────────────────────────────────────────
//
// A driven port is an interface the domain calls out to.  The domain says
// "I need a thing that can do X"; adapters in adapters/ supply the concrete X.
// The domain never imports adapters — dependency arrows point inward.

// TupleRepository is the persistence port for relationship tuples.
// The graph evaluator reads from it; write operations mutate it.
// Adapters decide the backend: in-memory, Postgres, OpenFGA, etc.
//
// Every method takes a context.Context and returns an error. The in-memory
// adapter never actually fails, but a real backend (Postgres, a network store)
// can time out, lose its connection, or be cancelled mid-query. Putting ctx and
// error in the port now means swapping in that backend later is a wiring change,
// not an interface change — and a slow check can be cancelled instead of hanging.
//
// Mirrors typescript/src/authz-service/core/ports/tupleRepository.ts.
type TupleRepository interface {
	// Has reports whether the exact (object, relation, user) tuple exists.
	Has(ctx context.Context, object shared.Object, relation shared.Relation, user shared.Subject) (bool, error)

	// FindByObjectRelation returns all tuples matching (object, relation).
	// Used during graph traversal.
	FindByObjectRelation(ctx context.Context, object shared.Object, relation shared.Relation) ([]shared.TupleKey, error)

	// FindAll returns all stored tuples, optionally filtered.
	FindAll(ctx context.Context, filter ...TupleFilter) ([]shared.TupleKey, error)

	// Write adds a tuple (idempotent).
	Write(ctx context.Context, tuple shared.TupleKey) error

	// Delete removes a tuple.  No-op if it does not exist.
	Delete(ctx context.Context, tuple shared.TupleKey) error
}

// TupleFilter narrows FindAll results.  Zero-value fields mean "match any".
type TupleFilter struct {
	Object   shared.Object
	Relation shared.Relation
}

// Evaluator is the port for graph-based permission evaluation.
// The authz domain delegates Check calls to this; adapters supply the strategy
// (in-process graph traversal, remote OpenFGA call, etc.).
//
// Mirrors typescript/src/authz-service/core/ports/evaluator.ts.
type Evaluator interface {
	Evaluate(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error)
}
