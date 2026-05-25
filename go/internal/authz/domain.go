package authz

import (
	"context"

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
func (d *authzDomain) WriteTuples(_ context.Context, tuples []shared.TupleKey) error {
	for _, t := range tuples {
		d.repository.Write(t)
	}
	return nil
}

// DeleteTuples removes relationship facts.
func (d *authzDomain) DeleteTuples(_ context.Context, tuples []shared.TupleKey) error {
	for _, t := range tuples {
		d.repository.Delete(t)
	}
	return nil
}

// ListTuples returns stored tuples, optionally filtered.
func (d *authzDomain) ListTuples(_ context.Context, filter ...TupleFilter) ([]shared.TupleKey, error) {
	return d.repository.FindAll(filter...), nil
}
