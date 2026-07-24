// Package rebac defines the authorization-domain vocabulary used across the project:
//
//	Resource / Relation / Subject  →  named types for relationship facts
//	Permission                     →  an authority derived by policy
//	Relationship                   →  one fact in the relationship graph
//	CheckRequest / CheckResult     →  a permission question and its decision
//
// The authz and documents packages both import it; neither owns it.
package rebac

import (
	"fmt"
	"strings"
)

// ── Resource types ────────────────────────────────────────────────────────────

// ResourceType is the set of recognised entity kinds in the domain.
type ResourceType string

const (
	ResourceTypeUser      ResourceType = "user"
	ResourceTypeTeam      ResourceType = "team"
	ResourceTypeWorkspace ResourceType = "workspace"
	ResourceTypeDocument  ResourceType = "document"
)

// Resource is a fully-qualified entity reference in "type:id" format,
// e.g. "user:alice" or "workspace:productWorkspace".
//
// Using a named string type (rather than plain string) makes it harder to
// accidentally pass a raw string where a Resource is expected, with the type
// checker enforcing it.
type Resource string

// Relation names an association used as stored or derivable policy evidence,
// such as "member", "editor", or "workspace". Relations are valid in stored
// Relationship facts and policy hierarchy rules.
type Relation string

const (
	// Team relations
	RelationTeamAdmin  Relation = "admin"
	RelationTeamMember Relation = "member"

	// Workspace relations
	RelationWorkspaceOwner  Relation = "owner"
	RelationWorkspaceEditor Relation = "editor"
	RelationWorkspaceViewer Relation = "viewer"

	// Document structural relation (links a document to its parent workspace)
	RelationDocumentWorkspace Relation = "workspace"

	// Document base relations
	RelationDocumentOwner  Relation = "owner"
	RelationDocumentEditor Relation = "editor"
	RelationDocumentViewer Relation = "viewer"
)

// Permission names an authority the policy can derive for a subject and
// resource. Permissions are checked, never stored as relationship facts.
//
// OpenFGA represents permissions in its relation field. The OpenFGA adapter
// performs that vocabulary translation; the application domain keeps the
// concepts distinct.
type Permission string

const (
	PermissionWorkspaceCreateDocument Permission = "can_create_document"

	PermissionDocumentRead    Permission = "can_read"
	PermissionDocumentComment Permission = "can_comment"
	PermissionDocumentEdit    Permission = "can_edit"
	PermissionDocumentDelete  Permission = "can_delete"
)

// Subject is either a plain Resource ("user:alice") or a subject set
// ("team:platform#member"). IsSubjectSet distinguishes them.
type Subject string

// Relationship stores one durable authorization fact:
//
//	(subject, relation, resource)
//
// For example, (user:alice, member, team:platform) reads "Alice is a member of
// the platform team." OpenFGA calls the same fields user, relation, and object;
// its adapter translates at the infrastructure boundary.
type Relationship struct {
	Subject  Subject  `json:"subject"`
	Relation Relation `json:"relation"`
	Resource Resource `json:"resource"`
}

// ── Check types ───────────────────────────────────────────────────────────────

// CheckRequest asks whether Subject has Permission on Resource.
type CheckRequest struct {
	Subject    Resource   `json:"subject"`
	Permission Permission `json:"permission"`
	Resource   Resource   `json:"resource"`
}

// CheckResult is the decision for a CheckRequest.
// Trace is an ordered log of the traversal steps — useful for debugging.
type CheckResult struct {
	Allowed bool     `json:"allowed"`
	Trace   []string `json:"trace"`
}

// ── Constructor helpers ───────────────────────────────────────────────────────

// User returns a Resource for a user entity: "user:<id>".
func User(id string) Resource { return newResource(ResourceTypeUser, id) }

// Team returns a Resource for a team entity: "team:<id>".
func Team(id string) Resource { return newResource(ResourceTypeTeam, id) }

// Workspace returns a Resource for a workspace entity: "workspace:<id>".
func Workspace(id string) Resource { return newResource(ResourceTypeWorkspace, id) }

// Document returns a Resource for a document entity: "document:<id>".
func Document(id string) Resource { return newResource(ResourceTypeDocument, id) }

// SubjectSet returns a subject-set string like "team:platformTeam#member".
func SubjectSet(resource Resource, relation Relation) Subject {
	return Subject(fmt.Sprintf("%s#%s", resource, relation))
}

// NewRelationship builds a relationship in canonical subject-relation-resource
// order.
func NewRelationship(subject Subject, relation Relation, resource Resource) Relationship {
	return Relationship{Subject: subject, Relation: relation, Resource: resource}
}

// ── Parsing helpers ───────────────────────────────────────────────────────────

// ParseResource splits "type:id" into its constituent parts.
// Returns an error if the format is invalid or the type is unrecognised.
func ParseResource(s string) (ResourceType, string, error) {
	idx := strings.IndexByte(s, ':')
	if idx < 1 || idx == len(s)-1 {
		return "", "", fmt.Errorf("invalid resource %q: want type:id", s)
	}
	typ := ResourceType(s[:idx])
	id := s[idx+1:]
	if !isResourceType(typ) {
		return "", "", fmt.Errorf("unknown resource type %q in %q", typ, s)
	}
	if strings.TrimSpace(id) == "" {
		return "", "", fmt.Errorf("invalid resource %q: id cannot be blank", s)
	}
	return typ, id, nil
}

// ParseSubjectSet splits "team:platformTeam#member" into its resource and relation.
func ParseSubjectSet(s Subject) (Resource, Relation, error) {
	str := string(s)
	idx := strings.IndexByte(str, '#')
	if idx < 1 || idx == len(str)-1 {
		return "", "", fmt.Errorf("invalid subject set %q: want resource#relation", s)
	}
	return Resource(str[:idx]), Relation(str[idx+1:]), nil
}

// IsSubjectSet reports whether s is a subject-set reference (contains '#').
func IsSubjectSet(s Subject) bool {
	return strings.ContainsRune(string(s), '#')
}

// ── Private helpers ───────────────────────────────────────────────────────────

func newResource(typ ResourceType, id string) Resource {
	if strings.TrimSpace(id) == "" {
		panic(fmt.Sprintf("rebac: %s id cannot be empty", typ))
	}
	return Resource(fmt.Sprintf("%s:%s", typ, id))
}

func isResourceType(t ResourceType) bool {
	switch t {
	case ResourceTypeUser, ResourceTypeTeam, ResourceTypeWorkspace, ResourceTypeDocument:
		return true
	}
	return false
}
