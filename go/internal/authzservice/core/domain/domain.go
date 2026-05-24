// Package domain assembles the authz service's use cases.
//
// It depends only on the ports (TupleRepository, Evaluator) — it has no
// knowledge of HTTP, in-memory stores, or graph algorithms.
//
// Mirrors typescript/src/authz-service/core/domain/makeAuthzDomain.ts.
package domain

import (
	"context"

	"rebac-primer/internal/authzservice/core/ports"
	"rebac-primer/internal/shared"
)

// AuthzService is the driving port — what callers (HTTP handlers, tests, other
// services) depend on.  It declares every operation the authz service offers.
type AuthzService interface {
	Check(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error)
	WriteTuples(ctx context.Context, tuples []shared.TupleKey) error
	DeleteTuples(ctx context.Context, tuples []shared.TupleKey) error
	ListTuples(ctx context.Context, filter ...ports.TupleFilter) ([]shared.TupleKey, error)
}

// authzDomain is the concrete implementation wired by New.
type authzDomain struct {
	repository ports.TupleRepository
	evaluator  ports.Evaluator
}

// New creates an AuthzService from its two driven ports.
// This is the composition function — it is the only place that knows both
// concrete types and can wire them together.
func New(repository ports.TupleRepository, evaluator ports.Evaluator) AuthzService {
	return &authzDomain{repository: repository, evaluator: evaluator}
}

// Check delegates evaluation to the Evaluator port.
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
func (d *authzDomain) ListTuples(_ context.Context, filter ...ports.TupleFilter) ([]shared.TupleKey, error) {
	return d.repository.FindAll(filter...), nil
}
