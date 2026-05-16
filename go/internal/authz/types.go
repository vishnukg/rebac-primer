// Package authz defines the types and interfaces for relationship-based access control.
//
// The central concept is a "tuple": a triple of (object, relation, subject) that
// asserts a relationship. For example:
//
//	(workspace:productWorkspace, editor, team:platformTeam#member)
//
// An Authorizer answers "does this user have this relation on this object?" by
// traversing the tuple graph.
package authz

import (
	"context"
	"fmt"
	"strings"
)

// ObjectType is the set of recognised entity kinds in the domain.
type ObjectType string

const (
	ObjectTypeUser      ObjectType = "user"
	ObjectTypeTeam      ObjectType = "team"
	ObjectTypeWorkspace ObjectType = "workspace"
	ObjectTypeDocument  ObjectType = "document"
)

// Object is a fully-qualified entity reference in "type:id" format.
// Using a named string type (rather than a plain string) makes it harder to
// accidentally pass a raw string where an Object is expected — the same idea as
// TypeScript's branded string types, enforced at compile time via named types.
type Object string

// Relation names a directed edge in the permission graph.
type Relation string

const (
	// Team relations
	RelationTeamAdmin  Relation = "admin"
	RelationTeamMember Relation = "member"

	// Workspace relations
	RelationWorkspaceOwner  Relation = "owner"
	RelationWorkspaceEditor Relation = "editor"
	RelationWorkspaceViewer Relation = "viewer"

	// Document structural relation
	RelationDocumentWorkspace Relation = "workspace"

	// Document base relations
	RelationDocumentOwner  Relation = "owner"
	RelationDocumentEditor Relation = "editor"
	RelationDocumentViewer Relation = "viewer"

	// Document computed relations
	RelationDocumentCanRead    Relation = "can_read"
	RelationDocumentCanComment Relation = "can_comment"
	RelationDocumentCanEdit    Relation = "can_edit"
	RelationDocumentCanDelete  Relation = "can_delete"
)

// Subject is either a plain Object ("user:alice") or a subject set
// ("team:platform#member").  Both are represented as strings; IsSubjectSet
// distinguishes them.
type Subject string

// TupleKey is one edge in the relationship graph.
type TupleKey struct {
	Object   Object
	Relation Relation
	User     Subject
}

// CheckRequest is the input to Authorizer.Check.
type CheckRequest struct {
	User     Object
	Relation Relation
	Object   Object
}

// CheckResult is the output of Authorizer.Check.
type CheckResult struct {
	Allowed bool
	Trace   []string
}

// Authorizer is the single interface the domain depends on.
// Implementations include GraphAuthorizer (in-memory) and OpenFGAAuthorizer (SDK adapter).
type Authorizer interface {
	Check(ctx context.Context, req CheckRequest) (CheckResult, error)
}

// --- helper constructors ---

// User returns an Object for a user entity.
func User(id string) Object {
	return newObject(ObjectTypeUser, id)
}

// Team returns an Object for a team entity.
func Team(id string) Object {
	return newObject(ObjectTypeTeam, id)
}

// Workspace returns an Object for a workspace entity.
func Workspace(id string) Object {
	return newObject(ObjectTypeWorkspace, id)
}

// Document returns an Object for a document entity.
func Document(id string) Object {
	return newObject(ObjectTypeDocument, id)
}

// SubjectSet returns a subject-set string like "team:platformTeam#member".
// This is the Go equivalent of the TS subjectSet() helper.
func SubjectSet(obj Object, rel Relation) Subject {
	return Subject(fmt.Sprintf("%s#%s", obj, rel))
}

// Tuple builds a TupleKey.
func Tuple(obj Object, rel Relation, subject Subject) TupleKey {
	return TupleKey{Object: obj, Relation: rel, User: subject}
}

// --- parsing helpers ---

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
// Returns an error if the format is invalid.
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

// --- private helpers ---

func newObject(typ ObjectType, id string) Object {
	if strings.TrimSpace(id) == "" {
		panic(fmt.Sprintf("authz: %s id cannot be empty", typ))
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
