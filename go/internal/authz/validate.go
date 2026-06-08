package authz

import (
	"fmt"

	"rebac-primer/internal/rebac"
)

// validateTuple checks that a tuple is well-formed before it is written.
//
// It returns a [*TupleValidationError] (which the HTTP adapter maps to HTTP 422)
// when any field is malformed:
//
//   - Object   must be a valid "type:id" with a known type (e.g. "document:roadmap").
//   - Relation must be non-empty.
//   - User     must be either a valid object ("user:alice") or a valid subject
//     set ("team:platform#member").
//
// Why this matters: the graph evaluator matches tuples by exact, parseable
// strings. A tuple like {Object: "roadmap"} (missing "document:") would be stored
// happily but never match any check — a silent authorization bug. Rejecting it at
// write time turns that latent bug into an immediate, explicit error.
func validateTuple(t rebac.TupleKey) error {
	if _, _, err := rebac.ParseObject(string(t.Object)); err != nil {
		return &TupleValidationError{Message: fmt.Sprintf("object %q is not a valid type:id (%v)", t.Object, err)}
	}
	if t.Relation == "" {
		return &TupleValidationError{Message: "relation cannot be empty"}
	}
	if err := validateSubject(t.User); err != nil {
		return &TupleValidationError{Message: fmt.Sprintf("user %q is not a valid object or subject set (%v)", t.User, err)}
	}
	return nil
}

// validateSubject accepts either a plain object ("user:alice") or a subject set
// ("team:platform#member"), matching the two shapes the User field can take.
func validateSubject(s rebac.Subject) error {
	if rebac.IsSubjectSet(s) {
		_, _, err := rebac.ParseSubjectSet(s)
		return err
	}
	_, _, err := rebac.ParseObject(string(s))
	return err
}
