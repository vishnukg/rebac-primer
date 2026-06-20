// GraphEvaluator answers permission checks by walking the relationship-tuple
// graph. It implements the [Evaluator] interface.
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
// The four fixture tuples create this graph in stored tuple direction:
//
//	document:roadmapDocument
//	  └─[workspace]─► workspace:productWorkspace
//	                    ├─[editor]─► team:platformTeam#member
//	                    │              └─[member]─► user:alice
//	                    └─[viewer]─► user:bob
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
// itself). Without a guard, the traversal would recurse forever. The active-path
// set records every (object#relation) pair on the current recursion path. If we
// encounter the same pair before unwinding, we found a cycle and stop that
// branch. Removing entries as calls return still allows a shared node to be
// evaluated through a different, independent path.

package authz

import (
	"context"
	"fmt"

	"rebac-primer/internal/rebac"
)

// defaultMaxDepth bounds how deep the recursive traversal may go in a single
// check. Cycle detection (the visited set) already stops loops; this is a second
// guard against a pathological or hostile graph that is deep but acyclic — it
// keeps one check from blowing the stack or hanging the request. OpenFGA enforces
// a comparable resolution-depth limit for the same reason.
const defaultMaxDepth = 100

// GraphEvaluator traverses the tuple graph to answer Check requests.
// Construct with [NewGraphEvaluator]; do not use the zero value directly.
type GraphEvaluator struct {
	store    TupleRepository
	maxDepth int
}

type relationVisit struct {
	object   rebac.Object
	relation rebac.Relation
}

// NewGraphEvaluator creates a GraphEvaluator backed by the given TupleRepository.
func NewGraphEvaluator(store TupleRepository) *GraphEvaluator {
	return &GraphEvaluator{store: store, maxDepth: defaultMaxDepth}
}

// Compile-time assertion: *GraphEvaluator must implement [Evaluator].
var _ Evaluator = (*GraphEvaluator)(nil)

// resolution holds the mutable state for one Check call: the request's context,
// the running trace, and the visited set. Bundling it in a struct keeps the
// recursive helpers' signatures small (they take only the node being visited plus
// the depth) instead of threading ctx, trace, and visited through every call.
// A fresh resolution is created per Evaluate, so concurrent checks never share
// state — the GraphEvaluator itself stays immutable and safe to share.
type resolution struct {
	ev       *GraphEvaluator
	ctx      context.Context
	trace    []string
	visiting map[relationVisit]bool
}

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
func (g *GraphEvaluator) Evaluate(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error) {
	// Validate here even though Service.Check also validates. GraphEvaluator
	// is exported and used directly by tests and teaching examples, so each
	// public entry point protects its own contract.
	if err := ValidateCheckRequest(req); err != nil {
		return rebac.CheckResult{}, err
	}
	r := &resolution{
		ev:  g,
		ctx: ctx,
		// Start the trace with the question being asked.
		trace: []string{
			fmt.Sprintf("Check whether %s has %s on %s", req.User, req.Relation, req.Object),
		},
		visiting: make(map[relationVisit]bool),
	}

	allowed, err := r.hasRelation(req.User, req.Object, req.Relation, 0)
	if err != nil {
		// Return the partial trace alongside the error so callers can still see how
		// far the traversal got before it was cancelled or hit a store failure.
		return rebac.CheckResult{Trace: r.trace}, err
	}

	if allowed {
		r.trace = append(r.trace, "Result: allowed")
	} else {
		r.trace = append(r.trace, "Result: denied")
	}

	return rebac.CheckResult{Allowed: allowed, Trace: r.trace}, nil
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
func (r *resolution) hasRelation(
	user rebac.Object,
	object rebac.Object,
	relation rebac.Relation,
	depth int,
) (bool, error) {
	// ── Cancellation guard ──────────────────────────────────────────────────────
	// If the caller's context was cancelled or timed out, abandon the walk now
	// rather than doing more work whose answer nobody is waiting for.
	if err := r.ctx.Err(); err != nil {
		return false, err
	}

	// ── Depth guard ─────────────────────────────────────────────────────────────
	// A second safety net beyond the cycle check below: bound total recursion depth
	// so a deep (but acyclic) or hostile graph cannot exhaust the stack or hang the
	// request. Exceeding it is an error, not a silent "denied".
	if depth > r.ev.maxDepth {
		return false, fmt.Errorf("graph: max resolution depth %d exceeded at %s#%s", r.ev.maxDepth, object, relation)
	}

	// ── Cycle guard ───────────────────────────────────────────────────────────
	// If this pair is already on the active recursion path, stop the cycle.
	visitKey := relationVisit{object: object, relation: relation}
	if r.visiting[visitKey] {
		r.trace = append(r.trace, fmt.Sprintf("Cycle detected at %s#%s; stop this branch", object, relation))
		return false, nil
	}
	r.visiting[visitKey] = true
	defer delete(r.visiting, visitKey)

	// ── Steps 1 & 2: direct tuple + subject-set ───────────────────────────────
	// Look in the tuple store.  This covers both:
	//   1. a direct tuple  (object, relation, user:alice)
	//   2. a subject-set   (object, relation, team:foo#member) where alice is a member
	found, err := r.hasTuple(user, object, relation, depth)
	if err != nil {
		return false, err
	}
	if found {
		return true, nil
	}

	// ── Step 3 & 4: permission-model expansion ────────────────────────────────
	// The tuple store said "no".  Ask the permission model whether this relation
	// can be satisfied by a stronger relation on the same object, or (for
	// documents) inherited from the parent workspace.
	typ, _, err := rebac.ParseObject(string(object))
	if err != nil {
		// Unknown object type — cannot expand further.
		return false, nil
	}

	switch typ {
	case rebac.ObjectTypeTeam:
		// e.g. "member" is satisfied by "admin"
		return r.expandByRules(teamRules, user, object, relation, depth)
	case rebac.ObjectTypeWorkspace:
		// e.g. "viewer" is satisfied by "editor", "editor" by "owner"
		return r.expandByRules(workspaceRules, user, object, relation, depth)
	case rebac.ObjectTypeDocument:
		// Documents have both rule expansion AND workspace inheritance.
		return r.expandDocument(user, object, relation, depth)
	}

	return false, nil
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
func (r *resolution) hasTuple(
	user rebac.Object,
	object rebac.Object,
	relation rebac.Relation,
	depth int,
) (bool, error) {
	// Step 1: direct lookup.
	// The store answers "does this exact (object, relation, user) triple exist?"
	direct, err := r.ev.store.Has(r.ctx, object, relation, rebac.Subject(user))
	if err != nil {
		return false, fmt.Errorf("store.Has(%s, %s, %s): %w", object, relation, user, err)
	}
	if direct {
		r.trace = append(r.trace, fmt.Sprintf("Found direct tuple (%s, %s, %s)", object, relation, user))
		return true, nil
	}

	// Step 2: subject-set lookup.
	// Scan all tuples for (object, relation, *).  For each one whose "user" field
	// is a subject set (contains '#'), recursively check whether our user is a
	// member of that set.
	candidates, err := r.ev.store.FindByObjectRelation(r.ctx, object, relation)
	if err != nil {
		return false, fmt.Errorf("store.FindByObjectRelation(%s, %s): %w", object, relation, err)
	}
	for _, tk := range candidates {
		if !rebac.IsSubjectSet(tk.User) {
			continue
		}
		contains, err := r.subjectSetContains(user, tk.User, depth)
		if err != nil {
			return false, err
		}
		if contains {
			r.trace = append(r.trace, fmt.Sprintf("Found subject-set tuple (%s, %s, %s)", object, relation, tk.User))
			return true, nil
		}
	}

	return false, nil
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
func (r *resolution) subjectSetContains(
	user rebac.Object,
	subject rebac.Subject,
	depth int,
) (bool, error) {
	// Split "team:platformTeam#member" into (team:platformTeam, member).
	ssObj, ssRel, err := rebac.ParseSubjectSet(subject)
	if err != nil {
		return false, nil
	}
	r.trace = append(r.trace, fmt.Sprintf("Resolve subject set %s: does it contain %s?", subject, user))
	// Recurse: check membership in the group.
	return r.hasRelation(user, ssObj, ssRel, depth+1)
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
func (r *resolution) expandByRules(
	rules impliedBy,
	user rebac.Object,
	object rebac.Object,
	relation rebac.Relation,
	depth int,
) (bool, error) {
	// rules[relation] is the list of stronger relations that imply relation.
	// If the key is missing, the slice is nil and the loop body never runs.
	for _, implied := range rules[relation] {
		r.trace = append(r.trace, fmt.Sprintf("%s %s includes %s", object, relation, implied))
		ok, err := r.hasRelation(user, object, implied, depth+1)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
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
func (r *resolution) expandDocument(
	user rebac.Object,
	object rebac.Object,
	relation rebac.Relation,
	depth int,
) (bool, error) {
	// Step 3: rule expansion (e.g. can_edit → editor, editor → owner).
	ok, err := r.expandByRules(documentRules, user, object, relation, depth)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}

	// Step 4: workspace inheritance.
	// Only base relations (owner, editor, viewer) propagate from workspace to
	// document.  Computed permissions (can_edit, can_read, …) are derived at
	// the document level — inheriting them would double-expand the rules.
	if relation == rebac.RelationDocumentOwner ||
		relation == rebac.RelationDocumentEditor ||
		relation == rebac.RelationDocumentViewer {

		// A document can have multiple workspace tuples in theory.
		// In practice the fixtures have exactly one: roadmapDocument → productWorkspace.
		parents, err := r.ev.store.FindByObjectRelation(r.ctx, object, rebac.RelationDocumentWorkspace)
		if err != nil {
			return false, fmt.Errorf("store.FindByObjectRelation(%s, workspace): %w", object, err)
		}
		for _, parent := range parents {
			r.trace = append(r.trace, fmt.Sprintf(
				"%s %s can inherit %s from %s", object, relation, relation, parent.User,
			))

			// parent.User is the Subject field of the workspace tuple.
			// Its value is the workspace's Object string, e.g. "workspace:productWorkspace".
			// We cast Subject → Object to use it as the next node in the traversal.
			wsObj := rebac.Object(parent.User)

			// Safety check: the tuple should point at a workspace, not some other type.
			wsTyp, _, err := rebac.ParseObject(string(wsObj))
			if err != nil || wsTyp != rebac.ObjectTypeWorkspace {
				continue
			}

			// Recurse: does the user have this relation on the parent workspace?
			ok, err := r.hasRelation(user, wsObj, relation, depth+1)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
	}

	return false, nil
}
