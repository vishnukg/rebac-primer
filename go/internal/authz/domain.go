package authz

import (
	"context"
	"fmt"

	"rebac-primer/internal/shared"
)

// authzDomain is the concrete implementation wired by [New].
// It is unexported — callers hold a [Service] interface value.
type authzDomain struct {
	repository TupleRepository
	evaluator  Evaluator
}

// New creates a [Service] from its two driven ports.
// This is the composition function — the only place that knows both concrete
// types and wires them together.
//
// Mirrors makeAuthzDomain() in typescript/src/authz-service/core/domain/makeAuthzDomain.ts.
func New(repository TupleRepository, evaluator Evaluator) Service {
	return &authzDomain{repository: repository, evaluator: evaluator}
}

// Check delegates permission evaluation to the [Evaluator] port.
func (d *authzDomain) Check(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error) {
	return d.evaluator.Evaluate(ctx, req)
}

// WriteTuples persists new relationship facts.
//
// Every tuple is validated before any is written, so a single malformed tuple
// rejects the whole batch (returning a [TupleValidationError]) instead of
// leaving a half-applied write. Validation guards the graph: a tuple whose
// object or user does not parse would silently never match during a check, which
// is the kind of bug that quietly grants or denies the wrong access.
func (d *authzDomain) WriteTuples(ctx context.Context, tuples []shared.TupleKey) error {
	for _, t := range tuples {
		if err := validateTuple(t); err != nil {
			return err
		}
	}
	for _, t := range tuples {
		if err := d.repository.Write(ctx, t); err != nil {
			return fmt.Errorf("write tuple (%s, %s, %s): %w", t.Object, t.Relation, t.User, err)
		}
	}
	return nil
}

// DeleteTuples removes relationship facts.
//
// Deletes are intentionally lenient: removing a malformed or non-existent tuple
// is a harmless no-op, so we do not validate here. Rejecting a delete would only
// make it harder to clean up bad data that somehow got in.
func (d *authzDomain) DeleteTuples(ctx context.Context, tuples []shared.TupleKey) error {
	for _, t := range tuples {
		if err := d.repository.Delete(ctx, t); err != nil {
			return fmt.Errorf("delete tuple (%s, %s, %s): %w", t.Object, t.Relation, t.User, err)
		}
	}
	return nil
}

// ListTuples returns stored tuples, optionally filtered.
func (d *authzDomain) ListTuples(ctx context.Context, filter ...TupleFilter) ([]shared.TupleKey, error) {
	return d.repository.FindAll(ctx, filter...)
}
