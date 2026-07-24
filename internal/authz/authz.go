// Package authz is the in-process authorization service.
//
// It answers permission checks ("does subject S have permission P on resource R?")
// by walking a graph of relationships, and it stores those relationships. The public
// surface is small:
//
//	Service               — the in-process authorization implementation.
//	New                   — builds a *Service from relationship-store ports and an Evaluator.
//	NewInMemoryStore      — relationship read/write/list ports backed by a map.
//	NewGraphEvaluator     — an Evaluator that walks the relationship graph.
//	ValidateRelationship  — validates facts before writes reach a backend.
//
// Consumers define the smallest interface they need. The store and evaluation
// strategy are interfaces here because this package consumes them.
package authz

import (
	"context"
	"fmt"

	"rebac-primer/internal/rebac"
)

// RelationshipReader is the read side of relationship storage used by
// GraphEvaluator during graph traversal.
//
// Every method takes a context.Context and returns an error. The in-memory store
// never actually fails, but a real backend (Postgres, a network store) can time
// out, drop its connection, or be cancelled mid-query — so the contract carries
// ctx and error from the start, and swapping backends stays a wiring change
// rather than an interface change.
type RelationshipReader interface {
	// Has reports whether the exact (subject, relation, resource) fact exists.
	Has(ctx context.Context, subject rebac.Subject, relation rebac.Relation, resource rebac.Resource) (bool, error)

	// FindByResourceRelation returns all relationships matching (resource, relation).
	// Used during graph traversal.
	FindByResourceRelation(ctx context.Context, resource rebac.Resource, relation rebac.Relation) ([]rebac.Relationship, error)
}

// RelationshipLister is the relationship enumeration capability used by
// administrative surfaces. It returns stored facts, not effective access.
type RelationshipLister interface {
	// FindAll returns all stored relationships, optionally filtered.
	FindAll(ctx context.Context, filter ...RelationshipFilter) ([]rebac.Relationship, error)
}

// RelationshipWriter is the mutation side of relationship storage used by
// Service.WriteRelationships and Service.DeleteRelationships.
type RelationshipWriter interface {
	// Write adds a relationship (idempotent).
	Write(ctx context.Context, relationship rebac.Relationship) error

	// Delete removes a relationship. No-op if it does not exist.
	Delete(ctx context.Context, relationship rebac.Relationship) error
}

// RelationshipRepository is the complete relationship-store capability used by the in-process
// authorization service. Narrower collaborators should usually accept
// RelationshipReader, RelationshipWriter, or RelationshipLister instead.
type RelationshipRepository interface {
	RelationshipReader
	RelationshipLister
	RelationshipWriter
}

// RelationshipFilter narrows FindAll results. Zero-value fields mean "match any".
type RelationshipFilter struct {
	Resource rebac.Resource
	Relation rebac.Relation
}

// Evaluator decides a single permission check. The service delegates Check to it,
// which lets the evaluation strategy vary (the in-process graph walk here, or a
// remote engine) without touching the service.
type Evaluator interface {
	Evaluate(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error)
}

// RelationshipValidationError signals that a relationship contains invalid data.
// The HTTP layer maps this to 422 Unprocessable Entity.
type RelationshipValidationError struct {
	Message string
}

func (e *RelationshipValidationError) Error() string {
	return fmt.Sprintf("relationship validation: %s", e.Message)
}

// CheckValidationError signals that a permission check is semantically invalid.
type CheckValidationError struct {
	Message string
}

func (e *CheckValidationError) Error() string {
	return fmt.Sprintf("check validation: %s", e.Message)
}
