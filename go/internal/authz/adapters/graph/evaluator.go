// Package graph provides the in-process graph traversal adapter for the authz
// service.  It satisfies the [authz.Evaluator] interface.
//
// # Graphs in one paragraph
//
// A graph is a set of nodes connected by edges.  In this system:
//
//   - Nodes  = entities  (a user, a team, a workspace, a document)
//   - Edges  = relationship tuples  (object, relation, user)
//
// For example, the tuple (team:platformTeam, member, user:alice) is a directed
// edge: "there is a 'member' edge from team:platformTeam to user:alice."
//
// The four fixture tuples create this graph:
//
//	user:alice ──[member]──────────────► team:platformTeam
//	                                             │
//	                                       [editor via #member]
//	                                             │
//	                                             ▼
//	user:bob ──[viewer]──────────► workspace:productWorkspace
//	                                             ▲
//	                                       [workspace]
//	                                             │
//	                               document:roadmapDocument
//
// # What a permission check is
//
// A check answers: "starting at <object>, can I find a path through the tuple
// graph that eventually reaches <user> by following relations that imply
// <relation>?"
//
// For example, "does user:alice have can_edit on document:roadmapDocument?"
// becomes: "is there a path from document:roadmapDocument to user:alice through
// edges that satisfy the can_edit permission?"
//
// The answer, for the fixture tuples, is yes — through this chain:
//
//	document:roadmapDocument
//	  --[workspace]--> workspace:productWorkspace
//	  --[editor via team:platformTeam#member]--> team:platformTeam
//	  --[member]--> user:alice
//
// # The traversal algorithm (depth-first search)
//
// The evaluator performs depth-first search (DFS): it picks a branch and
// follows it all the way down before trying another.  For each (object,
// relation) it visits, it tries four things in order:
//
//  1. Direct lookup      — is there a tuple (object, relation, user) in the store?
//  2. Subject-set        — is there a tuple (object, relation, group#rel) where
//     user is a member of that group?
//  3. Rule expansion     — does the permission model say this relation is implied
//     by a stronger relation? If so, recurse with that relation.
//  4. Workspace inherit  — (documents only) follow the "workspace" pointer to the
//     parent workspace and check the same relation there.
//
// If any branch returns true, the whole check is allowed.  If every branch is
// exhausted without finding the user, the check is denied.
//
// # Cycle detection
//
// Some graphs can have cycles (e.g. a document whose workspace pointer points to
// itself).  Without a guard, the traversal would recurse forever.  The visited
// set records every (object#relation) pair already evaluated in this request.
// If we visit the same pair again, we stop that branch and return false.
//
// Mirrors typescript/src/authz-service/adapters/graph/makeGraphEvaluator.ts.
package graph

import (
	"context"
	"fmt"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/shared"
)

// GraphEvaluator traverses the in-memory tuple graph to answer Check requests.
// Construct with [NewGraphEvaluator]; do not use the zero value directly.
type GraphEvaluator struct {
	store authz.TupleRepository
}

// NewGraphEvaluator creates a GraphEvaluator backed by the given TupleRepository.
func NewGraphEvaluator(store authz.TupleRepository) *GraphEvaluator {
	return &GraphEvaluator{store: store}
}

// Compile-time assertion: *GraphEvaluator must implement [authz.Evaluator].
var _ authz.Evaluator = (*GraphEvaluator)(nil)

// Evaluate is the entry point for a permission check.
//
// It answers: "does req.User have req.Relation on req.Object?"
//
// Example input:
//
//	req.User     = "user:alice"
//	req.Relation = "can_edit"
//	req.Object   = "document:roadmapDocument"
//
// It returns a CheckResult with Allowed=true/false and a Trace: a human-readable
// log of every step the traversal took, useful for debugging.  The trace is
// what you see when you run the tests with -v.
func (g *GraphEvaluator) Evaluate(_ context.Context, req shared.CheckRequest) (shared.CheckResult, error) {
	// Start the trace with the question being asked.
	trace := []string{
		fmt.Sprintf("Check whether %s has %s on %s", req.User, req.Relation, req.Object),
	}

	// visited prevents infinite loops on cyclic graphs.
	// Key format: "object#relation", e.g. "workspace:productWorkspace#editor".
	visited := make(map[string]bool)

	allowed := g.hasRelation(req.User, req.Object, req.Relation, &trace, visited)

	if allowed {
		trace = append(trace, "Result: allowed")
	} else {
		trace = append(trace, "Result: denied")
	}

	return shared.CheckResult{Allowed: allowed, Trace: trace}, nil
}

// ── Core traversal ────────────────────────────────────────────────────────────

// hasRelation is the recursive heart of the traversal.
//
// It answers: "does user have relation on object?" by trying — in order —
// a direct tuple lookup, subject-set expansion, permission-model rule expansion,
// and (for documents) workspace inheritance.
//
// Concrete trace for "alice / can_edit / document:roadmapDocument":
//
//	hasRelation(alice, document:roadmapDocument, can_edit)
//	  step 1: hasTuple → no direct tuple for alice/can_edit
//	  step 3: expand: can_edit is implied by editor (documentRules)
//	    hasRelation(alice, document:roadmapDocument, editor)
//	      step 1: hasTuple → no direct tuple for alice/editor
//	      step 3: expand: editor is implied by owner (documentRules)
//	        hasRelation(alice, document:roadmapDocument, owner)
//	          step 1: hasTuple → no direct tuple
//	          step 3: no rules for document/owner
//	          step 4: workspace inherit → check owner on workspace:productWorkspace
//	            hasRelation(alice, workspace:productWorkspace, owner) → false ✗
//	          → false
//	      step 4: workspace inherit → check editor on workspace:productWorkspace
//	        hasRelation(alice, workspace:productWorkspace, editor)
//	          step 1: hasTuple direct → miss
//	          step 2: hasTuple subject-set → team:platformTeam#member found!
//	            subjectSetContains(alice, team:platformTeam#member)
//	              hasRelation(alice, team:platformTeam, member)
//	                step 1: hasTuple direct → (team:platformTeam, member, alice) FOUND ✓
//	              → true ✓
//	          → true ✓
//	        → true ✓
//	      → true ✓
//	    → true ✓
//	  → true ✓ (can_edit satisfied by editor path)
func (g *GraphEvaluator) hasRelation(
	user shared.Object,
	object shared.Object,
	relation shared.Relation,
	trace *[]string,
	visited map[string]bool,
) bool {
	// ── Cycle guard ───────────────────────────────────────────────────────────
	// If we have already visited this (object, relation) pair in this request,
	// stop.  Without this, a cyclic graph would recurse forever.
	// The key intentionally omits the user — if we could not reach user from
	// this (object, relation) before, we cannot reach them now via the same path.
	visitKey := fmt.Sprintf("%s#%s", object, relation)
	if visited[visitKey] {
		*trace = append(*trace, fmt.Sprintf("Already evaluated %s; stop this branch", visitKey))
		return false
	}
	visited[visitKey] = true

	// ── Steps 1 & 2: direct tuple + subject-set ───────────────────────────────
	// Look in the tuple store.  This covers both:
	//   1. a direct tuple  (object, relation, user:alice)
	//   2. a subject-set   (object, relation, team:foo#member) where alice is a member
	if g.hasTuple(user, object, relation, trace, visited) {
		return true
	}

	// ── Step 3 & 4: permission-model expansion ────────────────────────────────
	// The tuple store said "no".  Ask the permission model whether this relation
	// can be satisfied by a stronger relation on the same object, or (for
	// documents) inherited from the parent workspace.
	typ, _, err := shared.ParseObject(string(object))
	if err != nil {
		// Unknown object type — cannot expand further.
		return false
	}

	switch typ {
	case shared.ObjectTypeTeam:
		// e.g. "member" is satisfied by "admin"
		return g.expandByRules(teamRules, user, object, relation, trace, visited)
	case shared.ObjectTypeWorkspace:
		// e.g. "viewer" is satisfied by "editor", "editor" by "owner"
		return g.expandByRules(workspaceRules, user, object, relation, trace, visited)
	case shared.ObjectTypeDocument:
		// Documents have both rule expansion AND workspace inheritance.
		return g.expandDocument(user, object, relation, trace, visited)
	}

	return false
}

// ── Tuple lookup (steps 1 & 2) ────────────────────────────────────────────────

// hasTuple checks the tuple store for a match.
//
// It has two sub-steps:
//
//  1. Direct match — "does (object, relation, user:alice) exist literally?"
//     Example: (team:platformTeam, member, user:alice) → yes, stop here.
//
//  2. Subject-set match — "does any tuple (object, relation, group#rel) exist
//     where alice is a member of that group?"
//     Example: (workspace:productWorkspace, editor, team:platformTeam#member)
//     → check if alice has 'member' on team:platformTeam → recursion.
//
// The subject-set check is what allows "grant access to a whole team with one
// tuple" — you write (workspace, editor, team#member) once and every team
// member gets it automatically.
func (g *GraphEvaluator) hasTuple(
	user shared.Object,
	object shared.Object,
	relation shared.Relation,
	trace *[]string,
	visited map[string]bool,
) bool {
	// Step 1: direct lookup.
	// The store answers "does this exact (object, relation, user) triple exist?"
	if g.store.Has(object, relation, shared.Subject(user)) {
		*trace = append(*trace, fmt.Sprintf("Found direct tuple (%s, %s, %s)", object, relation, user))
		return true
	}

	// Step 2: subject-set lookup.
	// Scan all tuples for (object, relation, *).  For each one whose "user" field
	// is a subject set (contains '#'), recursively check whether our user is a
	// member of that set.
	for _, tk := range g.store.FindByObjectRelation(object, relation) {
		if shared.IsSubjectSet(tk.User) && g.subjectSetContains(user, tk.User, trace, visited) {
			*trace = append(*trace, fmt.Sprintf("Found subject-set tuple (%s, %s, %s)", object, relation, tk.User))
			return true
		}
	}

	return false
}

// subjectSetContains resolves a subject-set reference and checks membership.
//
// A subject set is a string like "team:platformTeam#member".  It means
// "everyone who has the 'member' relation on team:platformTeam".
//
// To check whether alice is in team:platformTeam#member, we split the string:
//
//	object   = "team:platformTeam"
//	relation = "member"
//
// …and recursively ask: "does user:alice have member on team:platformTeam?"
// That is another hasRelation call — which might find a direct tuple, or expand
// further.  This is where the graph traversal "goes up" through groups.
func (g *GraphEvaluator) subjectSetContains(
	user shared.Object,
	subject shared.Subject,
	trace *[]string,
	visited map[string]bool,
) bool {
	// Split "team:platformTeam#member" into (team:platformTeam, member).
	ssObj, ssRel, err := shared.ParseSubjectSet(subject)
	if err != nil {
		return false
	}
	*trace = append(*trace, fmt.Sprintf("Resolve subject set %s: does it contain %s?", subject, user))
	// Recurse: check membership in the group.
	return g.hasRelation(user, ssObj, ssRel, trace, visited)
}

// ── Permission-model expansion (step 3) ───────────────────────────────────────

// expandByRules consults the permission model's implied-by table.
//
// The table says things like "can_edit is implied by editor" and "editor is
// implied by owner".  If we failed to find <relation> directly, we check each
// stronger relation that would satisfy it.
//
// Example — checking "editor" on workspace:productWorkspace:
//
//	workspaceRules["editor"] = ["owner"]
//	→ try hasRelation(alice, workspace:productWorkspace, owner)
//	→ if that returns true, editor is also satisfied.
//
// This is how role hierarchies work: you define the pyramid once in the rule
// table, not in every tuple.
func (g *GraphEvaluator) expandByRules(
	rules impliedBy,
	user shared.Object,
	object shared.Object,
	relation shared.Relation,
	trace *[]string,
	visited map[string]bool,
) bool {
	// rules[relation] is the list of stronger relations that imply relation.
	// If the key is missing, the slice is nil and the loop body never runs.
	for _, implied := range rules[relation] {
		*trace = append(*trace, fmt.Sprintf("%s %s includes %s", object, relation, implied))
		if g.hasRelation(user, object, implied, trace, visited) {
			return true
		}
	}
	return false
}

// ── Workspace inheritance for documents (step 4) ─────────────────────────────

// expandDocument handles the extra rules that apply to documents:
//
//  1. Rule expansion (same as other types — see expandByRules).
//  2. Workspace inheritance — a document can inherit owner/editor/viewer from
//     its parent workspace.
//
// Workspace inheritance works like this:
//
//	document:roadmapDocument --[workspace]--> workspace:productWorkspace
//
// If user alice has "editor" on workspace:productWorkspace, she also has
// "editor" on document:roadmapDocument — even without a direct document tuple.
//
// In code: follow every "workspace" tuple on this document to its parent
// workspace, then recursively check the same relation on that workspace.
// Only owner, editor, and viewer are inheritable — computed permissions like
// can_edit are resolved at the document level by expandByRules, not inherited.
func (g *GraphEvaluator) expandDocument(
	user shared.Object,
	object shared.Object,
	relation shared.Relation,
	trace *[]string,
	visited map[string]bool,
) bool {
	// Step 3: rule expansion (e.g. can_edit → editor, editor → owner).
	if g.expandByRules(documentRules, user, object, relation, trace, visited) {
		return true
	}

	// Step 4: workspace inheritance.
	// Only base relations (owner, editor, viewer) propagate from workspace to
	// document.  Computed permissions (can_edit, can_read, …) are derived at
	// the document level — inheriting them would double-expand the rules.
	if relation == shared.RelationDocumentOwner ||
		relation == shared.RelationDocumentEditor ||
		relation == shared.RelationDocumentViewer {

		// A document can have multiple workspace tuples in theory.
		// In practice the fixtures have exactly one: roadmapDocument → productWorkspace.
		for _, parent := range g.store.FindByObjectRelation(object, shared.RelationDocumentWorkspace) {
			*trace = append(*trace, fmt.Sprintf(
				"%s %s can inherit %s from %s", object, relation, relation, parent.User,
			))

			// parent.User is the Subject field of the workspace tuple.
			// Its value is the workspace's Object string, e.g. "workspace:productWorkspace".
			// We cast Subject → Object to use it as the next node in the traversal.
			wsObj := shared.Object(parent.User)

			// Safety check: the tuple should point at a workspace, not some other type.
			wsTyp, _, err := shared.ParseObject(string(wsObj))
			if err != nil || wsTyp != shared.ObjectTypeWorkspace {
				continue
			}

			// Recurse: does the user have this relation on the parent workspace?
			if g.hasRelation(user, wsObj, relation, trace, visited) {
				return true
			}
		}
	}

	return false
}
