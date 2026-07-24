// GraphEvaluator answers permission checks by walking the relationship
// graph. It implements the [Evaluator] interface.
//
// # Graphs in one paragraph
//
// A graph is a set of nodes connected by edges.  In this system:
//
//   - Nodes  = entities  (a user, a team, a workspace, a document)
//   - Edges  = relationships
//
// A relationship is represented as (subject, relation, resource). For example:
//
//	(user:alice, member, team:platformTeam)
//
// reads "user:alice is a member of team:platformTeam".
//
// # What a permission check is
//
// A check answers: "does <subject> have <permission> on <resource>?"
//
// For example, "does user:alice have can_edit on document:roadmapDocument?"
// maps can_edit to the editor relation, then asks whether Alice belongs to the
// effective editor set for that document.
//
// The learner-facing relationship chain is:
//
//	user:alice --member of--> team:platformTeam
//	team:platformTeam#member --editor of--> workspace:productWorkspace
//	workspace:productWorkspace --workspace of--> document:roadmapDocument
//
// The implementation resolves that chain in reverse, beginning with the
// requested resource and the relation required by the permission, then searching
// for the subject.
//
// # The traversal algorithm (depth-first search)
//
// The evaluator performs depth-first search (DFS): it picks a branch and
// follows it all the way down before trying another. After the checked permission
// is mapped to a required relation, the evaluator tries four things in order:
//
//  1. Direct lookup      — is there a relationship (subject, relation, resource)?
//  2. Subject-set        — is there a relationship (group#rel, relation, resource)
//     where the checked subject is a member of that group?
//  3. Rule expansion     — does the policy model say this relation is implied
//     by a stronger relation? If so, recurse with that relation.
//  4. Workspace inherit  — (documents only) follow the "workspace" pointer to the
//     parent workspace and check the same relation there.
//
// If any branch returns true, the whole check is allowed.  If every branch is
// exhausted without finding the user, the check is denied.
//
// # Cycle detection
//
// Malformed relationship data can contain cycles (for example, team:a#member can
// contain team:b#member while team:b#member contains team:a#member). Without a
// guard, the traversal would recurse forever. The active-path set records every
// (resource#relation) pair on the current recursion path. If we encounter the same
// pair before unwinding, we found a cycle and stop that branch. Removing entries
// as calls return still allows a shared node to be evaluated through a different,
// independent path.

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

// GraphEvaluator traverses the relationship graph to answer Check requests.
// Construct with [NewGraphEvaluator]; do not use the zero value directly.
type GraphEvaluator struct {
	store    RelationshipReader
	maxDepth int
}

type relationVisit struct {
	resource rebac.Resource
	relation rebac.Relation
}

// NewGraphEvaluator creates a GraphEvaluator backed by the given RelationshipReader.
func NewGraphEvaluator(store RelationshipReader) *GraphEvaluator {
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
// It answers: "does req.Subject have req.Permission on req.Resource?"
//
// Example input:
//
//	req.Subject    = "user:alice"
//	req.Permission = "can_edit"
//	req.Resource   = "document:roadmapDocument"
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
	resourceType, _, err := rebac.ParseResource(string(req.Resource))
	if err != nil {
		return rebac.CheckResult{}, err
	}
	r := &resolution{
		ev:  g,
		ctx: ctx,
		// Start the trace with the question being asked.
		trace: []string{
			fmt.Sprintf("Check whether %s has permission %s on %s",
				req.Subject, req.Permission, req.Resource),
		},
		visiting: make(map[relationVisit]bool),
	}

	var allowed bool
	for _, relation := range permissionRelationsFor(resourceType, req.Permission) {
		r.trace = append(r.trace, fmt.Sprintf(
			"Permission %s requires relation %s", req.Permission, relation,
		))
		allowed, err = r.hasRelation(req.Subject, req.Resource, relation, 0)
		if err != nil {
			// Return the partial trace alongside the error so callers can still see how
			// far the traversal got before it was cancelled or hit a store failure.
			return rebac.CheckResult{Trace: r.trace}, err
		}
		if allowed {
			break
		}
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
// It answers: "does subject have relation on resource?" by trying — in order —
// a direct relationship lookup, subject-set expansion, policy-model rule expansion,
// and (for documents) workspace inheritance.
//
// Concrete trace for "alice / can_edit / document:roadmapDocument":
//
//	permission can_edit requires relation editor
//	hasRelation(alice, document:roadmapDocument, editor)
//	      step 1: hasRelationship → no direct relationship for alice/editor
//	      step 3: expand: editor is implied by owner (documentRules)
//	        hasRelation(alice, document:roadmapDocument, owner)
//	          step 1: hasRelationship → no direct relationship
//	          step 3: no rules for document/owner
//	          step 4: workspace inherit → check owner on workspace:productWorkspace
//	            hasRelation(alice, workspace:productWorkspace, owner) → false ✗
//	          → false
//	      step 4: workspace inherit → check editor on workspace:productWorkspace
//	        hasRelation(alice, workspace:productWorkspace, editor)
//	          step 1: hasRelationship direct → miss
//	          step 2: hasRelationship subject-set → team:platformTeam#member found!
//	            subjectSetContains(alice, team:platformTeam#member)
//	              hasRelation(alice, team:platformTeam, member)
//	                step 1: direct relationship → (alice, member, team:platformTeam) FOUND ✓
//	              → true ✓
//	          → true ✓
//	        → true ✓
//	      → true ✓
//	→ true ✓ (can_edit satisfied by editor path)
func (r *resolution) hasRelation(
	subject rebac.Resource,
	resource rebac.Resource,
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
		return false, fmt.Errorf("graph: max resolution depth %d exceeded at %s#%s", r.ev.maxDepth, resource, relation)
	}

	// ── Cycle guard ───────────────────────────────────────────────────────────
	// If this pair is already on the active recursion path, stop the cycle.
	visitKey := relationVisit{resource: resource, relation: relation}
	if r.visiting[visitKey] {
		r.trace = append(r.trace, fmt.Sprintf("Cycle detected at %s#%s; stop this branch", resource, relation))
		return false, nil
	}
	r.visiting[visitKey] = true
	defer delete(r.visiting, visitKey)

	// ── Steps 1 & 2: direct relationship + subject-set ────────────────────────
	// Look in the relationship store. This covers both:
	//   1. a direct fact (subject, relation, resource)
	//   2. a subject-set fact (team:foo#member, relation, resource)
	typ, _, err := rebac.ParseResource(string(resource))
	if err != nil {
		// ValidateCheckRequest has already checked the resource, so this is only a
		// defensive guard if hasRelation is reused internally in the future.
		return false, fmt.Errorf("graph: parse resource %q: %w", resource, err)
	}
	found, err := r.hasRelationship(subject, resource, relation, depth)
	if err != nil {
		return false, err
	}
	if found {
		return true, nil
	}

	// ── Steps 3 & 4: policy-model expansion ───────────────────────────────────
	// The relationship store said "no". Ask the policy model whether a stronger
	// relation on the same resource or an inherited workspace relation satisfies it.
	switch typ {
	case rebac.ResourceTypeTeam:
		return r.expandByRules(teamRules, subject, resource, relation, depth)
	case rebac.ResourceTypeWorkspace:
		return r.expandByRules(workspaceRules, subject, resource, relation, depth)
	case rebac.ResourceTypeDocument:
		return r.expandDocument(subject, resource, relation, depth)
	default:
		return false, nil
	}
}

// ── Relationship lookup (steps 1 & 2) ────────────────────────────────────────

// hasRelationship checks the relationship store for a match.
func (r *resolution) hasRelationship(
	subject rebac.Resource,
	resource rebac.Resource,
	relation rebac.Relation,
	depth int,
) (bool, error) {
	direct, err := r.ev.store.Has(r.ctx, rebac.Subject(subject), relation, resource)
	if err != nil {
		return false, fmt.Errorf("store.Has(%s, %s, %s): %w", subject, relation, resource, err)
	}
	if direct {
		r.trace = append(r.trace, fmt.Sprintf(
			"Found direct relationship (%s, %s, %s)", subject, relation, resource,
		))
		return true, nil
	}

	candidates, err := r.ev.store.FindByResourceRelation(r.ctx, resource, relation)
	if err != nil {
		return false, fmt.Errorf(
			"store.FindByResourceRelation(%s, %s): %w", resource, relation, err,
		)
	}
	for _, relationship := range candidates {
		if !rebac.IsSubjectSet(relationship.Subject) {
			continue
		}
		contains, err := r.subjectSetContains(subject, relationship.Subject, depth)
		if err != nil {
			return false, err
		}
		if contains {
			r.trace = append(r.trace, fmt.Sprintf(
				"Found subject-set relationship (%s, %s, %s)",
				relationship.Subject, relation, resource,
			))
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
//	resource = "team:platformTeam"
//	relation = "member"
//
// …and recursively ask: "does user:alice have member on team:platformTeam?"
// That is another hasRelation call—which might find a direct relationship or expand
// further.  This is where the graph traversal "goes up" through groups.
func (r *resolution) subjectSetContains(
	subjectResource rebac.Resource,
	subject rebac.Subject,
	depth int,
) (bool, error) {
	// Split "team:platformTeam#member" into (team:platformTeam, member).
	ssObj, ssRel, err := rebac.ParseSubjectSet(subject)
	if err != nil {
		return false, nil
	}
	r.trace = append(r.trace, fmt.Sprintf(
		"Resolve subject set %s: does it contain %s?", subject, subjectResource,
	))
	// Recurse: check membership in the group.
	return r.hasRelation(subjectResource, ssObj, ssRel, depth+1)
}

// ── Relation-hierarchy expansion (step 3) ─────────────────────────────────────

// expandByRules consults the policy model's relation-hierarchy table.
//
// The table says things like "viewer is implied by editor" and "editor is
// implied by owner". If we failed to find <relation> directly, we check each
// stronger relation that would satisfy it. Permission-to-relation mapping has
// already happened in Evaluate before this traversal begins.
//
// Example — checking "editor" on workspace:productWorkspace:
//
//	workspaceRules["editor"] = ["owner"]
//	→ try hasRelation(alice, workspace:productWorkspace, owner)
//	→ if that returns true, editor is also satisfied.
//
// This is how role hierarchies work: you define the pyramid once in the rule
// table, not in every relationship.
func (r *resolution) expandByRules(
	rules impliedBy,
	subject rebac.Resource,
	resource rebac.Resource,
	relation rebac.Relation,
	depth int,
) (bool, error) {
	// rules[relation] is the list of stronger relations that imply relation.
	// If the key is missing, the slice is nil and the loop body never runs.
	for _, implied := range rules[relation] {
		r.trace = append(r.trace, fmt.Sprintf("%s %s includes %s", resource, relation, implied))
		ok, err := r.hasRelation(subject, resource, implied, depth+1)
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
// "editor" on document:roadmapDocument—even without a direct document relationship.
//
// In code: follow every "workspace" relationship on this document to its parent
// workspace, then recursively check the same relation on that workspace.
// Only owner, editor, and viewer are inheritable. Evaluate maps a permission
// such as can_edit to one of those relations before traversal begins.
func (r *resolution) expandDocument(
	subject rebac.Resource,
	resource rebac.Resource,
	relation rebac.Relation,
	depth int,
) (bool, error) {
	// Step 3: same-resource relation expansion (e.g. editor → owner).
	ok, err := r.expandByRules(documentRules, subject, resource, relation, depth)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}

	// Step 4: workspace inheritance.
	// Only base relations (owner, editor, viewer) propagate from workspace to
	// document. Permissions have already been mapped to a required relation.
	if isDocumentBaseRelation(relation) {

		// A document can have multiple workspace relationships in theory.
		// In practice the fixtures have exactly one: roadmapDocument → productWorkspace.
		parents, err := r.ev.store.FindByResourceRelation(
			r.ctx, resource, rebac.RelationDocumentWorkspace,
		)
		if err != nil {
			return false, fmt.Errorf(
				"store.FindByResourceRelation(%s, workspace): %w", resource, err,
			)
		}
		for _, parent := range parents {
			r.trace = append(r.trace, fmt.Sprintf(
				"%s %s can inherit %s from %s", resource, relation, relation, parent.Subject,
			))

			// The relationship subject is the parent workspace resource.
			workspace := rebac.Resource(parent.Subject)

			// Safety check: the relationship should point at a workspace.
			wsTyp, _, err := rebac.ParseResource(string(workspace))
			if err != nil || wsTyp != rebac.ResourceTypeWorkspace {
				continue
			}

			// Recurse: does the subject have this relation on the parent workspace?
			ok, err := r.hasRelation(subject, workspace, relation, depth+1)
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
