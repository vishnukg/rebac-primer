package authz

import (
	"fmt"

	"rebac-primer/internal/rebac"
)

// ValidateRelationship checks that a relationship is well-formed before it is written.
//
// It returns a [*RelationshipValidationError] (which the HTTP adapter maps to HTTP 422)
// when any field is malformed:
//
//   - Resource must be a valid "type:id" with a known type (e.g. "document:roadmap").
//   - Relation must be non-empty.
//   - Subject  must be either a valid resource ("user:alice") or a valid subject
//     set ("team:platform#member").
//
// Why this matters: the graph evaluator matches relationships by exact, parseable
// strings. A relationship like {Resource: "roadmap"} (missing "document:") would be stored
// happily but never match any check — a silent authorization bug. Rejecting it at
// write time turns that latent bug into an immediate, explicit error.
func ValidateRelationship(relationship rebac.Relationship) error {
	resourceType, _, err := rebac.ParseResource(string(relationship.Resource))
	if err != nil {
		return &RelationshipValidationError{Message: fmt.Sprintf(
			"resource %q is not a valid type:id (%v)", relationship.Resource, err,
		)}
	}
	if relationship.Relation == "" {
		return &RelationshipValidationError{Message: "relation cannot be empty"}
	}
	if !relationDefinedFor(resourceType, relationship.Relation) {
		return &RelationshipValidationError{Message: fmt.Sprintf(
			"relation %q is not defined for %s resources", relationship.Relation, resourceType,
		)}
	}
	subjectType, subjectRelation, err := validateSubject(relationship.Subject)
	if err != nil {
		return &RelationshipValidationError{Message: fmt.Sprintf(
			"subject %q is not a valid resource or subject set (%v)", relationship.Subject, err,
		)}
	}
	if !subjectAllowed(resourceType, relationship.Relation, subjectType, subjectRelation) {
		return &RelationshipValidationError{Message: fmt.Sprintf(
			"subject %q is not allowed for %s#%s",
			relationship.Subject, resourceType, relationship.Relation,
		)}
	}
	return nil
}

// validateSubject accepts either a plain resource ("user:alice") or a subject set
// ("team:platform#member"), matching the two shapes Relationship.Subject can take.
func validateSubject(s rebac.Subject) (rebac.ResourceType, rebac.Relation, error) {
	if rebac.IsSubjectSet(s) {
		resource, relation, err := rebac.ParseSubjectSet(s)
		if err != nil {
			return "", "", err
		}
		resourceType, _, err := rebac.ParseResource(string(resource))
		return resourceType, relation, err
	}
	resourceType, _, err := rebac.ParseResource(string(s))
	return resourceType, "", err
}

// ValidateCheckRequest rejects malformed or model-unknown checks instead of
// silently turning caller mistakes into authorization denials.
func ValidateCheckRequest(req rebac.CheckRequest) error {
	subjectType, _, err := rebac.ParseResource(string(req.Subject))
	if err != nil || subjectType != rebac.ResourceTypeUser {
		return &CheckValidationError{Message: fmt.Sprintf(
			"check subject %q must be a valid user resource", req.Subject,
		)}
	}
	resourceType, _, err := rebac.ParseResource(string(req.Resource))
	if err != nil {
		return &CheckValidationError{Message: fmt.Sprintf(
			"check resource %q is invalid (%v)", req.Resource, err,
		)}
	}
	if !actionDefinedFor(resourceType, req.Action) {
		return &CheckValidationError{Message: fmt.Sprintf(
			"action %q is not defined for %s resources", req.Action, resourceType,
		)}
	}
	return nil
}

func relationDefinedFor(resourceType rebac.ResourceType, relation rebac.Relation) bool {
	switch resourceType {
	case rebac.ResourceTypeTeam:
		return relation == rebac.RelationTeamAdmin || relation == rebac.RelationTeamMember
	case rebac.ResourceTypeWorkspace:
		return relation == rebac.RelationWorkspaceOwner ||
			relation == rebac.RelationWorkspaceEditor ||
			relation == rebac.RelationWorkspaceViewer
	case rebac.ResourceTypeDocument:
		switch relation {
		case rebac.RelationDocumentWorkspace,
			rebac.RelationDocumentOwner,
			rebac.RelationDocumentEditor,
			rebac.RelationDocumentViewer:
			return true
		}
	}
	return false
}

func actionDefinedFor(resourceType rebac.ResourceType, action rebac.Action) bool {
	return len(actionRelationsFor(resourceType, action)) > 0
}

func isDocumentBaseRelation(relation rebac.Relation) bool {
	switch relation {
	case rebac.RelationDocumentOwner,
		rebac.RelationDocumentEditor,
		rebac.RelationDocumentViewer:
		return true
	}
	return false
}

func subjectAllowed(
	resourceType rebac.ResourceType,
	relation rebac.Relation,
	subjectType rebac.ResourceType,
	subjectRelation rebac.Relation,
) bool {
	if subjectRelation == "" {
		if resourceType == rebac.ResourceTypeDocument && relation == rebac.RelationDocumentWorkspace {
			return subjectType == rebac.ResourceTypeWorkspace
		}
		return subjectType == rebac.ResourceTypeUser
	}
	if subjectType != rebac.ResourceTypeTeam {
		return false
	}
	switch {
	case resourceType == rebac.ResourceTypeWorkspace && relation == rebac.RelationWorkspaceOwner:
		return subjectRelation == rebac.RelationTeamAdmin
	case resourceType == rebac.ResourceTypeWorkspace &&
		(relation == rebac.RelationWorkspaceEditor || relation == rebac.RelationWorkspaceViewer):
		return subjectRelation == rebac.RelationTeamMember
	}
	return false
}
