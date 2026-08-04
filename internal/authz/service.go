package authz

import (
	"context"
	"fmt"

	"rebac-primer/internal/rebac"
)

// Service answers authorization questions and manages relationships.
// Construct it with [New]; its zero value is not usable.
//
// Its methods have pointer receivers because Service contains collaborators and
// should not be copied. Consumers normally accept *Service through a narrow
// interface declared in the consuming package.
type Service struct {
	writer    RelationshipWriter
	lister    RelationshipLister
	evaluator Evaluator
}

// New creates a Service from a RelationshipRepository and an Evaluator.
func New(repository RelationshipRepository, evaluator Evaluator) *Service {
	return &Service{writer: repository, lister: repository, evaluator: evaluator}
}

// Check delegates action evaluation to the [Evaluator] port.
func (d *Service) Check(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error) {
	// Validate at the service boundary because callers may supply a different
	// Evaluator implementation that does not validate requests itself.
	if err := ValidateCheckRequest(req); err != nil {
		return rebac.CheckResult{}, err
	}
	return d.evaluator.Evaluate(ctx, req)
}

// WriteRelationships persists new relationship facts.
//
// Every relationship is validated before any is written, so a single malformed
// fact rejects the whole batch (returning a [RelationshipValidationError])
// instead of leaving a half-applied write. Validation guards the graph: a
// relationship whose resource or subject does not parse would silently never
// match during a check, which is the kind of bug that quietly grants or denies
// the wrong access.
func (d *Service) WriteRelationships(ctx context.Context, relationships []rebac.Relationship) error {
	for _, relationship := range relationships {
		if err := ValidateRelationship(relationship); err != nil {
			return err
		}
	}
	for _, relationship := range relationships {
		if err := d.writer.Write(ctx, relationship); err != nil {
			return fmt.Errorf("write relationship (%s, %s, %s): %w",
				relationship.Subject, relationship.Relation, relationship.Resource, err)
		}
	}
	return nil
}

// DeleteRelationships removes relationship facts.
//
// Deletes are intentionally lenient: removing a malformed or non-existent
// relationship is a harmless no-op, so we do not validate here. Rejecting a
// delete would only make it harder to clean up bad data that somehow got in.
func (d *Service) DeleteRelationships(ctx context.Context, relationships []rebac.Relationship) error {
	for _, relationship := range relationships {
		if err := d.writer.Delete(ctx, relationship); err != nil {
			return fmt.Errorf("delete relationship (%s, %s, %s): %w",
				relationship.Subject, relationship.Relation, relationship.Resource, err)
		}
	}
	return nil
}

// ListRelationships returns stored relationships, optionally filtered.
func (d *Service) ListRelationships(ctx context.Context, filter ...RelationshipFilter) ([]rebac.Relationship, error) {
	return d.lister.FindAll(ctx, filter...)
}
