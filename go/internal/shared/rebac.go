// Package shared defines the core ReBAC primitives shared by the authz and
// documents services.
//
// These types are the Go equivalents of typescript/src/shared/rebac.ts:
//
//	Object / Relation / Subject  →  named string types (like TS branded types)
//	TupleKey                     →  one edge in the relationship graph
//	CheckRequest / CheckResult   →  the question and answer for a permission check
//
// Both services import this package; neither service owns it.
package shared

import (
	"fmt"
	"strings"
)

// ── Object types ──────────────────────────────────────────────────────────────

// ObjectType is the set of recognised entity kinds in the domain.
type ObjectType string

const (
	ObjectTypeUser      ObjectType = "user"
	ObjectTypeTeam      ObjectType = "team"
	ObjectTypeWorkspace ObjectType = "workspace"
	ObjectTypeDocument  ObjectType = "document"
)

// Object is a fully-qualified entity reference in "type:id" format,
// e.g. "user:alice" or "workspace:productWorkspace".
//
// Using a named string type (rather than plain string) makes it harder to
// accidentally pass a raw string where an Object is expected — the same idea
// as TypeScript's branded string types, enforced by the Go type system.
type Object string

// Relation names a directed edge in the permission graph, e.g. "editor" or "can_read".
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

	// Document computed relations (derived by the graph evaluator)
	RelationDocumentCanRead    Relation = "can_read"
	RelationDocumentCanComment Relation = "can_comment"
	RelationDocumentCanEdit    Relation = "can_edit"
	RelationDocumentCanDelete  Relation = "can_delete"
)

// Subject is either a plain Object ("user:alice") or a subject set
// ("team:platform#member"). IsSubjectSet distinguishes them.
type Subject string

// TupleKey is one edge in the relationship graph:
//
//	(object, relation, user)  →  "team:platform has member user:alice"
type TupleKey struct {
	Object   Object
	Relation Relation
	User     Subject
}

// ── Check types ───────────────────────────────────────────────────────────────

// CheckRequest asks "does User have Relation on Object?"
type CheckRequest struct {
	User     Object
	Relation Relation
	Object   Object
}

// CheckResult is the answer to a CheckRequest.
// Trace is an ordered log of the traversal steps — useful for debugging.
type CheckResult struct {
	Allowed bool
	Trace   []string
}

// ── Constructor helpers ───────────────────────────────────────────────────────

// User returns an Object for a user entity: "user:<id>".
func User(id string) Object { return newObject(ObjectTypeUser, id) }

// Team returns an Object for a team entity: "team:<id>".
func Team(id string) Object { return newObject(ObjectTypeTeam, id) }

// Workspace returns an Object for a workspace entity: "workspace:<id>".
func Workspace(id string) Object { return newObject(ObjectTypeWorkspace, id) }

// Document returns an Object for a document entity: "document:<id>".
func Document(id string) Object { return newObject(ObjectTypeDocument, id) }

// SubjectSet returns a subject-set string like "team:platformTeam#member".
// This is the Go equivalent of the TS subjectSet() helper.
func SubjectSet(obj Object, rel Relation) Subject {
	return Subject(fmt.Sprintf("%s#%s", obj, rel))
}

// Tuple builds a TupleKey. Short-form constructor for use in fixture files.
func Tuple(obj Object, rel Relation, subject Subject) TupleKey {
	return TupleKey{Object: obj, Relation: rel, User: subject}
}

// ── Parsing helpers ───────────────────────────────────────────────────────────

// ParseObject splits "type:id" into its constituent parts.
// Returns an error if the format is invalid or the type is unrecognised.
func ParseObject(s string) (ObjectType, string, error) {
	idx := strings.IndexByte(s, ':')
	if idx < 1 || idx == len(s)-1 {
		return "", "", fmt.Errorf("invalid object %q: want type:id", s)
	}
	typ := ObjectType(s[:idx])
	id := s[idx+1:]
	if !isObjectType(typ) {
		return "", "", fmt.Errorf("unknown object type %q in %q", typ, s)
	}
	return typ, id, nil
}

// ParseSubjectSet splits "team:platformTeam#member" into its object and relation.
func ParseSubjectSet(s Subject) (Object, Relation, error) {
	str := string(s)
	idx := strings.IndexByte(str, '#')
	if idx < 1 || idx == len(str)-1 {
		return "", "", fmt.Errorf("invalid subject set %q: want object#relation", s)
	}
	return Object(str[:idx]), Relation(str[idx+1:]), nil
}

// IsSubjectSet reports whether s is a subject-set reference (contains '#').
func IsSubjectSet(s Subject) bool {
	return strings.ContainsRune(string(s), '#')
}

// ── Private helpers ───────────────────────────────────────────────────────────

func newObject(typ ObjectType, id string) Object {
	if strings.TrimSpace(id) == "" {
		panic(fmt.Sprintf("shared: %s id cannot be empty", typ))
	}
	return Object(fmt.Sprintf("%s:%s", typ, id))
}

func isObjectType(t ObjectType) bool {
	switch t {
	case ObjectTypeUser, ObjectTypeTeam, ObjectTypeWorkspace, ObjectTypeDocument:
		return true
	}
	return false
}
