package authz

import (
	"context"
	"fmt"

	"rebac-primer/internal/rebac"
)

// Service answers authorization questions and manages relationship tuples.
// Construct it with [New]; its zero value is not usable.
//
// Its methods have pointer receivers because Service contains collaborators and
// should not be copied. Consumers normally accept *Service through a narrow
// interface declared in the consuming package.
type Service struct {
	repository TupleRepository
	evaluator  Evaluator
}

// New creates a Service from a TupleRepository and an Evaluator.
func New(repository TupleRepository, evaluator Evaluator) *Service {
	return &Service{repository: repository, evaluator: evaluator}
}

// Check delegates permission evaluation to the [Evaluator] port.
func (d *Service) Check(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error) {
	// Validate at the service boundary because callers may supply a different
	// Evaluator implementation that does not validate requests itself.
	if err := ValidateCheckRequest(req); err != nil {
		return rebac.CheckResult{}, err
	}
	return d.evaluator.Evaluate(ctx, req)
}

// WriteTuples persists new relationship facts.
//
// Every tuple is validated before any is written, so a single malformed tuple
// rejects the whole batch (returning a [TupleValidationError]) instead of
// leaving a half-applied write. Validation guards the graph: a tuple whose
// object or user does not parse would silently never match during a check, which
// is the kind of bug that quietly grants or denies the wrong access.
func (d *Service) WriteTuples(ctx context.Context, tuples []rebac.TupleKey) error {
	for _, t := range tuples {
		if err := ValidateTuple(t); err != nil {
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
func (d *Service) DeleteTuples(ctx context.Context, tuples []rebac.TupleKey) error {
	for _, t := range tuples {
		if err := d.repository.Delete(ctx, t); err != nil {
			return fmt.Errorf("delete tuple (%s, %s, %s): %w", t.Object, t.Relation, t.User, err)
		}
	}
	return nil
}

// ListTuples returns stored tuples, optionally filtered.
func (d *Service) ListTuples(ctx context.Context, filter ...TupleFilter) ([]rebac.TupleKey, error) {
	return d.repository.FindAll(ctx, filter...)
}
