// Package authz is the in-process authorization service.
//
// It answers permission checks ("does user U have relation R on object O?") by
// walking a graph of relationship tuples, and it stores those tuples. The public
// surface is small:
//
//	Service          — what callers use: Check / WriteTuples / DeleteTuples / ListTuples.
//	New              — builds a Service from a TupleRepository and an Evaluator.
//	NewInMemoryStore — a TupleRepository backed by a map (the default store).
//	NewGraphEvaluator — an Evaluator that walks the tuple graph (the default strategy).
//
// Callers depend on the Service interface, never the concrete type. The store and
// the evaluation strategy are themselves interfaces, so either can be swapped: the
// openfga package, for instance, implements Service directly against a remote
// OpenFGA server.
package authz

import (
	"context"
	"fmt"

	"rebac-primer/internal/rebac"
)

// Service answers authorization questions and manages the tuples behind them.
// It is what HTTP handlers, tests, and other services call into; New returns the
// default in-process implementation.
type Service interface {
	Check(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error)
	WriteTuples(ctx context.Context, tuples []rebac.TupleKey) error
	DeleteTuples(ctx context.Context, tuples []rebac.TupleKey) error
	ListTuples(ctx context.Context, filter ...TupleFilter) ([]rebac.TupleKey, error)
}

// TupleRepository stores relationship tuples. The evaluator reads from it; the
// service's write operations mutate it.
//
// Every method takes a context.Context and returns an error. The in-memory store
// never actually fails, but a real backend (Postgres, a network store) can time
// out, drop its connection, or be cancelled mid-query — so the contract carries
// ctx and error from the start, and swapping backends stays a wiring change
// rather than an interface change.
type TupleRepository interface {
	// Has reports whether the exact (object, relation, user) tuple exists.
	Has(ctx context.Context, object rebac.Object, relation rebac.Relation, user rebac.Subject) (bool, error)

	// FindByObjectRelation returns all tuples matching (object, relation).
	// Used during graph traversal.
	FindByObjectRelation(ctx context.Context, object rebac.Object, relation rebac.Relation) ([]rebac.TupleKey, error)

	// FindAll returns all stored tuples, optionally filtered.
	FindAll(ctx context.Context, filter ...TupleFilter) ([]rebac.TupleKey, error)

	// Write adds a tuple (idempotent).
	Write(ctx context.Context, tuple rebac.TupleKey) error

	// Delete removes a tuple. No-op if it does not exist.
	Delete(ctx context.Context, tuple rebac.TupleKey) error
}

// TupleFilter narrows FindAll results. Zero-value fields mean "match any".
type TupleFilter struct {
	Object   rebac.Object
	Relation rebac.Relation
}

// Evaluator decides a single permission check. The service delegates Check to it,
// which lets the evaluation strategy vary (the in-process graph walk here, or a
// remote engine) without touching the service.
type Evaluator interface {
	Evaluate(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error)
}

// TupleValidationError signals that a tuple contains semantically invalid data.
// The HTTP layer maps this to 422 Unprocessable Entity.
type TupleValidationError struct {
	Message string
}

func (e *TupleValidationError) Error() string {
	return fmt.Sprintf("tuple validation: %s", e.Message)
}
