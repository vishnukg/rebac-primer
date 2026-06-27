// Package authz is the in-process authorization service.
//
// It answers permission checks ("does user U have relation R on object O?") by
// walking a graph of relationship tuples, and it stores those tuples. The public
// surface is small:
//
//	Service          — the in-process authorization implementation.
//	New              — builds a *Service from tuple write/list ports and an Evaluator.
//	NewInMemoryStore — tuple read/write/list ports backed by a map.
//	NewGraphEvaluator — an Evaluator that walks the tuple graph (the default strategy).
//	ValidateTuple    — validates tuple shape before writes reach a backend.
//
// Consumers define the smallest interface they need. The store and evaluation
// strategy are interfaces here because this package consumes them.
package authz

import (
	"context"
	"fmt"

	"rebac-primer/internal/rebac"
)

// TupleReader is the read side of relationship tuple storage used by
// GraphEvaluator during graph traversal.
//
// Every method takes a context.Context and returns an error. The in-memory store
// never actually fails, but a real backend (Postgres, a network store) can time
// out, drop its connection, or be cancelled mid-query — so the contract carries
// ctx and error from the start, and swapping backends stays a wiring change
// rather than an interface change.
type TupleReader interface {
	// Has reports whether the exact (object, relation, user) tuple exists.
	Has(ctx context.Context, object rebac.Object, relation rebac.Relation, user rebac.Subject) (bool, error)

	// FindByObjectRelation returns all tuples matching (object, relation).
	// Used during graph traversal.
	FindByObjectRelation(ctx context.Context, object rebac.Object, relation rebac.Relation) ([]rebac.TupleKey, error)
}

// TupleLister is the tuple enumeration capability used by administrative
// surfaces such as ListTuples. It returns stored facts, not effective access.
type TupleLister interface {
	// FindAll returns all stored tuples, optionally filtered.
	FindAll(ctx context.Context, filter ...TupleFilter) ([]rebac.TupleKey, error)
}

// TupleWriter is the mutation side of relationship tuple storage used by
// Service.WriteTuples and Service.DeleteTuples.
type TupleWriter interface {
	// Write adds a tuple (idempotent).
	Write(ctx context.Context, tuple rebac.TupleKey) error

	// Delete removes a tuple. No-op if it does not exist.
	Delete(ctx context.Context, tuple rebac.TupleKey) error
}

// TupleRepository is the complete tuple-store capability used by the in-process
// authorization service. Narrower collaborators should usually accept
// TupleReader, TupleWriter, or TupleLister instead.
type TupleRepository interface {
	TupleReader
	TupleLister
	TupleWriter
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
