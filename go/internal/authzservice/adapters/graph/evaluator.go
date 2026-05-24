// Package graph provides the in-process graph traversal adapter for the authz
// service.  It satisfies the ports.Evaluator interface.
//
// Mirrors typescript/src/authz-service/adapters/graph/makeGraphEvaluator.ts.
package graph

import (
	"context"
	"fmt"

	"rebac-primer/internal/authzservice/core/ports"
	"rebac-primer/internal/shared"
)

// GraphEvaluator traverses the in-memory tuple graph to answer Check requests.
//
// The algorithm:
//  1. Check whether the user appears directly in a tuple (object, relation, user).
//  2. Check whether any subject-set tuple for (object, relation) contains the user.
//  3. Expand the relation according to the permission model hierarchy.
//  4. For document owner/editor/viewer, follow the "workspace" tuple and check
//     the same relation on the linked workspace object.
//
// Cycle detection uses a visited set keyed on "object#relation".
type GraphEvaluator struct {
	store ports.TupleRepository
}

// NewGraphEvaluator creates a GraphEvaluator backed by the given TupleRepository.
func NewGraphEvaluator(store ports.TupleRepository) *GraphEvaluator {
	return &GraphEvaluator{store: store}
}

// Evaluate satisfies the ports.Evaluator interface.
func (g *GraphEvaluator) Evaluate(_ context.Context, req shared.CheckRequest) (shared.CheckResult, error) {
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

	return shared.CheckResult{Allowed: allowed, Trace: trace}, nil
}

// Compile-time assertion: *GraphEvaluator must implement ports.Evaluator.
var _ ports.Evaluator = (*GraphEvaluator)(nil)

// ── Traversal ─────────────────────────────────────────────────────────────────
//
// 1. Skip if already evaluated this (object, relation) pair (cycle guard).
// 2. Check for a stored tuple (direct or via a subject-set group).
// 3. Expand implied relations from the permission model for this object type.

func (g *GraphEvaluator) hasRelation(
	user shared.Object,
	object shared.Object,
	relation shared.Relation,
	trace *[]string,
	visited map[string]bool,
) bool {
	visitKey := fmt.Sprintf("%s#%s", object, relation)
	if visited[visitKey] {
		*trace = append(*trace, fmt.Sprintf("Already evaluated %s; stop this branch", visitKey))
		return false
	}
	visited[visitKey] = true

	if g.hasTuple(user, object, relation, trace, visited) {
		return true
	}

	typ, _, err := shared.ParseObject(string(object))
	if err != nil {
		return false
	}

	switch typ {
	case shared.ObjectTypeTeam:
		return g.expandByRules(teamRules, user, object, relation, trace, visited)
	case shared.ObjectTypeWorkspace:
		return g.expandByRules(workspaceRules, user, object, relation, trace, visited)
	case shared.ObjectTypeDocument:
		return g.expandDocument(user, object, relation, trace, visited)
	}

	return false
}

// ── Tuple lookup ──────────────────────────────────────────────────────────────

func (g *GraphEvaluator) hasTuple(
	user shared.Object,
	object shared.Object,
	relation shared.Relation,
	trace *[]string,
	visited map[string]bool,
) bool {
	if g.store.Has(object, relation, shared.Subject(user)) {
		*trace = append(*trace, fmt.Sprintf("Found direct tuple (%s, %s, %s)", object, relation, user))
		return true
	}

	for _, tk := range g.store.FindByObjectRelation(object, relation) {
		if shared.IsSubjectSet(tk.User) && g.subjectSetContains(user, tk.User, trace, visited) {
			*trace = append(*trace, fmt.Sprintf("Found subject-set tuple (%s, %s, %s)", object, relation, tk.User))
			return true
		}
	}

	return false
}

func (g *GraphEvaluator) subjectSetContains(
	user shared.Object,
	subject shared.Subject,
	trace *[]string,
	visited map[string]bool,
) bool {
	ssObj, ssRel, err := shared.ParseSubjectSet(subject)
	if err != nil {
		return false
	}
	*trace = append(*trace, fmt.Sprintf("Resolve subject set %s: does it contain %s?", subject, user))
	return g.hasRelation(user, ssObj, ssRel, trace, visited)
}

// ── Permission model expansion ─────────────────────────────────────────────────

func (g *GraphEvaluator) expandByRules(
	rules impliedBy,
	user shared.Object,
	object shared.Object,
	relation shared.Relation,
	trace *[]string,
	visited map[string]bool,
) bool {
	for _, implied := range rules[relation] {
		*trace = append(*trace, fmt.Sprintf("%s %s includes %s", object, relation, implied))
		if g.hasRelation(user, object, implied, trace, visited) {
			return true
		}
	}
	return false
}

// expandDocument applies documentRules plus workspace inheritance: owner/editor/viewer
// can be inherited from the document's parent workspace via a "workspace" tuple.
func (g *GraphEvaluator) expandDocument(
	user shared.Object,
	object shared.Object,
	relation shared.Relation,
	trace *[]string,
	visited map[string]bool,
) bool {
	if g.expandByRules(documentRules, user, object, relation, trace, visited) {
		return true
	}

	if relation == shared.RelationDocumentOwner ||
		relation == shared.RelationDocumentEditor ||
		relation == shared.RelationDocumentViewer {
		for _, parent := range g.store.FindByObjectRelation(object, shared.RelationDocumentWorkspace) {
			*trace = append(*trace, fmt.Sprintf(
				"%s %s can inherit %s from %s", object, relation, relation, parent.User,
			))
			wsObj := shared.Object(parent.User)
			wsTyp, _, err := shared.ParseObject(string(wsObj))
			if err != nil || wsTyp != shared.ObjectTypeWorkspace {
				continue
			}
			if g.hasRelation(user, wsObj, relation, trace, visited) {
				return true
			}
		}
	}

	return false
}
