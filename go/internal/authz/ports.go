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
// Mirrors typescript/src/authz-service/core/ports/tupleRepository.ts.
type TupleRepository interface {
	// Has reports whether the exact (object, relation, user) tuple exists.
	Has(object shared.Object, relation shared.Relation, user shared.Subject) bool

	// FindByObjectRelation returns all tuples matching (object, relation).
	// Used during graph traversal.
	FindByObjectRelation(object shared.Object, relation shared.Relation) []shared.TupleKey

	// FindAll returns all stored tuples, optionally filtered.
	FindAll(filter ...TupleFilter) []shared.TupleKey

	// Write adds a tuple (idempotent).
	Write(tuple shared.TupleKey)

	// Delete removes a tuple.  No-op if it does not exist.
	Delete(tuple shared.TupleKey)
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
