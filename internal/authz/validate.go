package authz

import (
	"fmt"

	"rebac-primer/internal/rebac"
)

// ValidateTuple checks that a tuple is well-formed before it is written.
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
func ValidateTuple(t rebac.TupleKey) error {
	objectType, _, err := rebac.ParseObject(string(t.Object))
	if err != nil {
		return &TupleValidationError{Message: fmt.Sprintf("object %q is not a valid type:id (%v)", t.Object, err)}
	}
	if t.Relation == "" {
		return &TupleValidationError{Message: "relation cannot be empty"}
	}
	if !relationDefinedFor(objectType, t.Relation) {
		return &TupleValidationError{Message: fmt.Sprintf("relation %q is not defined for %s objects", t.Relation, objectType)}
	}
	if isComputedRelation(objectType, t.Relation) {
		return &TupleValidationError{Message: fmt.Sprintf("relation %q is computed and cannot be written", t.Relation)}
	}

	subjectType, subjectRelation, err := validateSubject(t.User)
	if err != nil {
		return &TupleValidationError{Message: fmt.Sprintf("user %q is not a valid object or subject set (%v)", t.User, err)}
	}
	if !subjectAllowed(objectType, t.Relation, subjectType, subjectRelation) {
		return &TupleValidationError{Message: fmt.Sprintf(
			"user %q is not allowed for %s#%s", t.User, objectType, t.Relation,
		)}
	}
	return nil
}

// validateSubject accepts either a plain object ("user:alice") or a subject set
// ("team:platform#member"), matching the two shapes the User field can take.
func validateSubject(s rebac.Subject) (rebac.ObjectType, rebac.Relation, error) {
	if rebac.IsSubjectSet(s) {
		obj, relation, err := rebac.ParseSubjectSet(s)
		if err != nil {
			return "", "", err
		}
		objectType, _, err := rebac.ParseObject(string(obj))
		return objectType, relation, err
	}
	objectType, _, err := rebac.ParseObject(string(s))
	return objectType, "", err
}

// ValidateCheckRequest rejects malformed or model-unknown checks instead of
// silently turning caller mistakes into authorization denials.
func ValidateCheckRequest(req rebac.CheckRequest) error {
	userType, _, err := rebac.ParseObject(string(req.User))
	if err != nil || userType != rebac.ObjectTypeUser {
		return &TupleValidationError{Message: fmt.Sprintf("check user %q must be a valid user object", req.User)}
	}
	objectType, _, err := rebac.ParseObject(string(req.Object))
	if err != nil {
		return &TupleValidationError{Message: fmt.Sprintf("check object %q is invalid (%v)", req.Object, err)}
	}
	if !relationDefinedFor(objectType, req.Relation) {
		return &TupleValidationError{Message: fmt.Sprintf("relation %q is not defined for %s objects", req.Relation, objectType)}
	}
	if !relationCheckableForUser(objectType, req.Relation) {
		return &TupleValidationError{Message: fmt.Sprintf("relation %q on %s objects cannot be checked for a user", req.Relation, objectType)}
	}
	return nil
}

func relationDefinedFor(objectType rebac.ObjectType, relation rebac.Relation) bool {
	switch objectType {
	case rebac.ObjectTypeTeam:
		return relation == rebac.RelationTeamAdmin || relation == rebac.RelationTeamMember
	case rebac.ObjectTypeWorkspace:
		return relation == rebac.RelationWorkspaceOwner ||
			relation == rebac.RelationWorkspaceEditor ||
			relation == rebac.RelationWorkspaceViewer
	case rebac.ObjectTypeDocument:
		switch relation {
		case rebac.RelationDocumentWorkspace,
			rebac.RelationDocumentOwner,
			rebac.RelationDocumentEditor,
			rebac.RelationDocumentViewer,
			rebac.RelationDocumentCanRead,
			rebac.RelationDocumentCanComment,
			rebac.RelationDocumentCanEdit,
			rebac.RelationDocumentCanDelete:
			return true
		}
	}
	return false
}

func isComputedRelation(objectType rebac.ObjectType, relation rebac.Relation) bool {
	if objectType != rebac.ObjectTypeDocument {
		return false
	}
	switch relation {
	case rebac.RelationDocumentCanRead,
		rebac.RelationDocumentCanComment,
		rebac.RelationDocumentCanEdit,
		rebac.RelationDocumentCanDelete:
		return true
	}
	return false
}

func relationCheckableForUser(objectType rebac.ObjectType, relation rebac.Relation) bool {
	if objectType == rebac.ObjectTypeDocument && relation == rebac.RelationDocumentWorkspace {
		return false
	}
	return true
}

func subjectAllowed(
	objectType rebac.ObjectType,
	relation rebac.Relation,
	subjectType rebac.ObjectType,
	subjectRelation rebac.Relation,
) bool {
	if subjectRelation == "" {
		if objectType == rebac.ObjectTypeDocument && relation == rebac.RelationDocumentWorkspace {
			return subjectType == rebac.ObjectTypeWorkspace
		}
		return subjectType == rebac.ObjectTypeUser
	}
	if subjectType != rebac.ObjectTypeTeam {
		return false
	}
	switch {
	case objectType == rebac.ObjectTypeWorkspace && relation == rebac.RelationWorkspaceOwner:
		return subjectRelation == rebac.RelationTeamAdmin
	case objectType == rebac.ObjectTypeWorkspace &&
		(relation == rebac.RelationWorkspaceEditor || relation == rebac.RelationWorkspaceViewer):
		return subjectRelation == rebac.RelationTeamMember
	}
	return false
}
