package authz

import (
	"context"
	"fmt"
)

// GraphAuthorizer traverses the in-memory tuple graph to answer Check requests.
// It implements the same recursive, cycle-safe algorithm as the TypeScript GraphAuthorizer.
//
// The algorithm in plain English:
//  1. Check whether the user appears directly in a tuple (object, relation, user).
//  2. Check whether any subject-set tuple for (object, relation) contains the user.
//  3. Expand the relation according to the model hierarchy (e.g. can_edit → editor → owner).
//  4. For document owner/editor/viewer, follow the "workspace" pointer and check the
//     same relation on the linked workspace object.
//
// Cycle detection uses a visited set keyed on "object#relation".
type GraphAuthorizer struct {
	store TupleReader
}

// NewGraphAuthorizer creates a GraphAuthorizer backed by the given TupleReader.
func NewGraphAuthorizer(store TupleReader) *GraphAuthorizer {
	return &GraphAuthorizer{store: store}
}

// Check satisfies the Authorizer interface.
func (g *GraphAuthorizer) Check(_ context.Context, req CheckRequest) (CheckResult, error) {
	trace := []string{
		fmt.Sprintf("Check whether %s has %s on %s", req.User, req.Relation, req.Object),
	}
	visited := make(map[string]bool)
	allowed := g.hasRelation(req.User, req.Object, req.Relation, &trace, visited)

	if allowed {
		trace = append(trace, "Result: allowed")
	} else {
		trace = append(trace, "Result: denied")
	}

	return CheckResult{Allowed: allowed, Trace: trace}, nil
}

// hasRelation is the recursive heart of the graph traversal.
// It appends reasoning steps to *trace as it explores the graph.
func (g *GraphAuthorizer) hasRelation(
	user Object,
	object Object,
	relation Relation,
	trace *[]string,
	visited map[string]bool,
) bool {
	visitKey := fmt.Sprintf("%s#%s", object, relation)
	if visited[visitKey] {
		*trace = append(*trace, fmt.Sprintf("Already evaluated %s; stop this branch", visitKey))
		return false
	}
	visited[visitKey] = true

	if g.hasDirectUserOrSubjectSet(user, object, relation, trace, visited) {
		return true
	}

	typ, _, err := ParseObject(string(object))
	if err != nil {
		return false
	}

	switch typ {
	case ObjectTypeTeam:
		return g.expandTeam(user, object, relation, trace, visited)
	case ObjectTypeWorkspace:
		return g.expandWorkspace(user, object, relation, trace, visited)
	case ObjectTypeDocument:
		return g.expandDocument(user, object, relation, trace, visited)
	}

	return false
}

// expandTeam applies the team model hierarchy:
//
//	team.member includes team.admin
func (g *GraphAuthorizer) expandTeam(
	user Object,
	object Object,
	relation Relation,
	trace *[]string,
	visited map[string]bool,
) bool {
	if relation == RelationTeamMember {
		*trace = append(*trace, "team.member includes team.admin")
		return g.hasRelation(user, object, RelationTeamAdmin, trace, visited)
	}
	return false
}

// expandWorkspace applies the workspace model hierarchy:
//
//	workspace.editor includes workspace.owner
//	workspace.viewer includes workspace.editor
func (g *GraphAuthorizer) expandWorkspace(
	user Object,
	object Object,
	relation Relation,
	trace *[]string,
	visited map[string]bool,
) bool {
	if relation == RelationWorkspaceEditor {
		*trace = append(*trace, "workspace.editor includes workspace.owner")
		return g.hasRelation(user, object, RelationWorkspaceOwner, trace, visited)
	}
	if relation == RelationWorkspaceViewer {
		*trace = append(*trace, "workspace.viewer includes workspace.editor")
		return g.hasRelation(user, object, RelationWorkspaceEditor, trace, visited)
	}
	return false
}

// expandDocument applies the document model hierarchy and workspace inheritance.
//
// Computed permissions:
//
//	can_read   → viewer
//	can_comment → viewer
//	can_edit   → editor
//	can_delete  → owner
//
// Base permission hierarchy:
//
//	viewer → editor → owner
//
// Workspace inheritance (for owner/editor/viewer):
//
//	check the same relation on the workspace linked via the "workspace" tuple.
func (g *GraphAuthorizer) expandDocument(
	user Object,
	object Object,
	relation Relation,
	trace *[]string,
	visited map[string]bool,
) bool {
	// Computed permission → base permission
	expansions := map[Relation]Relation{
		RelationDocumentCanRead:    RelationDocumentViewer,
		RelationDocumentCanComment: RelationDocumentViewer,
		RelationDocumentCanEdit:    RelationDocumentEditor,
		RelationDocumentCanDelete:  RelationDocumentOwner,
		RelationDocumentViewer:     RelationDocumentEditor,
		RelationDocumentEditor:     RelationDocumentOwner,
	}

	if implied, ok := expansions[relation]; ok {
		*trace = append(*trace, fmt.Sprintf("document.%s includes document.%s", relation, implied))
		if g.hasRelation(user, object, implied, trace, visited) {
			return true
		}
	}

	// Workspace inheritance: owner, editor, viewer can come from the workspace.
	if relation == RelationDocumentOwner || relation == RelationDocumentEditor || relation == RelationDocumentViewer {
		for _, parent := range g.store.FindByObjectRelation(object, RelationDocumentWorkspace) {
			*trace = append(*trace, fmt.Sprintf(
				"document.%s can inherit workspace.%s from %s", relation, relation, parent.User,
			))
			wsObj := Object(parent.User)
			wsTyp, _, err := ParseObject(string(wsObj))
			if err != nil || wsTyp != ObjectTypeWorkspace {
				continue
			}
			if g.hasRelation(user, wsObj, relation, trace, visited) {
				return true
			}
		}
	}

	return false
}

// hasDirectUserOrSubjectSet checks for a direct tuple match or a subject-set match.
func (g *GraphAuthorizer) hasDirectUserOrSubjectSet(
	user Object,
	object Object,
	relation Relation,
	trace *[]string,
	visited map[string]bool,
) bool {
	// Direct tuple: (object, relation, user)
	if g.store.Has(object, relation, Subject(user)) {
		*trace = append(*trace, fmt.Sprintf("Found direct tuple (%s, %s, %s)", object, relation, user))
		return true
	}

	// Subject-set tuples: (object, relation, team:x#member)
	for _, tk := range g.store.FindByObjectRelation(object, relation) {
		if IsSubjectSet(tk.User) && g.subjectSetContains(user, tk.User, trace, visited) {
			*trace = append(*trace, fmt.Sprintf("Found subject-set tuple (%s, %s, %s)", object, relation, tk.User))
			return true
		}
	}

	return false
}

// subjectSetContains resolves "team:x#member" to check whether user is a member of that set.
func (g *GraphAuthorizer) subjectSetContains(
	user Object,
	subject Subject,
	trace *[]string,
	visited map[string]bool,
) bool {
	ssObj, ssRel, err := ParseSubjectSet(subject)
	if err != nil {
		return false
	}
	*trace = append(*trace, fmt.Sprintf("Resolve subject set %s: does it contain %s?", subject, user))
	return g.hasRelation(user, ssObj, ssRel, trace, visited)
}
