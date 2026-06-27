// Package contract defines the canonical authorization truth table — the single
// source of truth for "what the model means" — and a runner that checks any
// backend against it.
//
// # Why this exists
//
// The same permission model is encoded more than once — the from-scratch rule
// tables in model.go and the OpenFGA DSL (deployments/openfga/model.fga) — and
// nothing forces the encodings to agree. This package pins the intended behavior
// as data so a drift in any one of them fails a test instead of silently changing
// who can access what.
//
// Cases() is the matrix; Run() executes it against a CheckFunc. The from-scratch
// evaluator's Evaluate and the OpenFGA service's Check both satisfy CheckFunc, so
// both backends are held to the same contract.
//
// Note: this package imports "testing" even though it is not a _test.go file.
// That is intentional — it is a shared test helper, like the standard library's
// testing/quick. Only test files import it, so it never ends up in the server
// binary.
package contract

import (
	"context"
	"testing"

	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/rebac"
)

// CheckFunc is the single operation a backend must provide to be held to the
// contract: answer one CheckRequest. Both an evaluator's Evaluate method and a
// backend service's Check method have this exact signature.
type CheckFunc func(context.Context, rebac.CheckRequest) (rebac.CheckResult, error)

// Case is one row of the truth table: a question and its required answer.
type Case struct {
	Name     string
	User     rebac.Object
	Relation rebac.Relation
	Object   rebac.Object
	Allowed  bool
}

// ExtraTuples returns contract-only tuples that exercise policy paths not
// covered by the demo story alone. They are deliberately kept out of
// fixtures.SeedRelationshipTuples so the public demo remains small, but both
// backends write them before running this contract.
func ExtraTuples() []rebac.TupleKey {
	return []rebac.TupleKey{
		// Direct document ownership: proves owner -> can_delete and owner ->
		// editor -> viewer -> can_read/can_comment.
		rebac.Tuple(fixtures.RoadmapDocument, rebac.RelationDocumentOwner, rebac.Subject(fixtures.Dana)),
		// Team admin path: proves admin implies member, and team#admin can own a
		// workspace.
		rebac.Tuple(fixtures.PlatformTeam, rebac.RelationTeamAdmin, rebac.Subject(fixtures.Erin)),
		rebac.Tuple(fixtures.ProductWorkspace, rebac.RelationWorkspaceOwner, rebac.SubjectSet(fixtures.PlatformTeam, rebac.RelationTeamAdmin)),
	}
}

// Cases returns the canonical allow/deny matrix for the standard fixture
// scenario plus ExtraTuples: alice is a platform-team member, the team edits the
// product workspace, bob is a direct workspace viewer, casey has no
// relationships, dana directly owns the roadmap document, erin is a platform
// team admin, and the roadmap document lives in the workspace.
//
// Every backend must produce these exact answers. To run it against OpenFGA, the
// store must hold the same tuples: the policy tuples from deployments/openfga/
// seed.sh plus the document's workspace tuple and ExtraTuples, which the OpenFGA
// contract test writes itself. The store must hold no unrelated tuples — in
// particular, starting the server seeds a demo document owned by alice, and that
// owner tuple changes the can_delete answers this contract pins down.
func Cases() []Case {
	doc := fixtures.RoadmapDocument
	ws := fixtures.ProductWorkspace
	team := fixtures.PlatformTeam

	return []Case{
		// ── Document computed permissions ─────────────────────────────────────
		// alice: team member → workspace editor → document editor (inherited).
		{"alice can_read roadmap", fixtures.Alice, rebac.RelationDocumentCanRead, doc, true},
		{"alice can_comment roadmap", fixtures.Alice, rebac.RelationDocumentCanComment, doc, true},
		{"alice can_edit roadmap", fixtures.Alice, rebac.RelationDocumentCanEdit, doc, true},
		{"alice cannot can_delete roadmap (not owner)", fixtures.Alice, rebac.RelationDocumentCanDelete, doc, false},

		// dana: direct document owner.
		{"dana can_read roadmap (direct owner)", fixtures.Dana, rebac.RelationDocumentCanRead, doc, true},
		{"dana can_comment roadmap (direct owner)", fixtures.Dana, rebac.RelationDocumentCanComment, doc, true},
		{"dana can_edit roadmap (direct owner)", fixtures.Dana, rebac.RelationDocumentCanEdit, doc, true},
		{"dana can_delete roadmap (direct owner)", fixtures.Dana, rebac.RelationDocumentCanDelete, doc, true},

		// erin: team admin -> workspace owner via team#admin -> document owner.
		{"erin can_read roadmap (workspace owner)", fixtures.Erin, rebac.RelationDocumentCanRead, doc, true},
		{"erin can_edit roadmap (workspace owner)", fixtures.Erin, rebac.RelationDocumentCanEdit, doc, true},
		{"erin can_delete roadmap (workspace owner)", fixtures.Erin, rebac.RelationDocumentCanDelete, doc, true},

		// bob: direct workspace viewer → document viewer (inherited).
		{"bob can_read roadmap", fixtures.Bob, rebac.RelationDocumentCanRead, doc, true},
		{"bob can_comment roadmap", fixtures.Bob, rebac.RelationDocumentCanComment, doc, true},
		{"bob cannot can_edit roadmap (viewer only)", fixtures.Bob, rebac.RelationDocumentCanEdit, doc, false},
		{"bob cannot can_delete roadmap", fixtures.Bob, rebac.RelationDocumentCanDelete, doc, false},

		// casey: no relationships → no access.
		{"casey cannot can_read roadmap", fixtures.Casey, rebac.RelationDocumentCanRead, doc, false},
		{"casey cannot can_edit roadmap", fixtures.Casey, rebac.RelationDocumentCanEdit, doc, false},
		{"casey cannot can_delete roadmap", fixtures.Casey, rebac.RelationDocumentCanDelete, doc, false},

		// ── Workspace base relations ──────────────────────────────────────────
		{"alice is workspace editor (via team#member)", fixtures.Alice, rebac.RelationWorkspaceEditor, ws, true},
		{"alice is workspace viewer (editor implies viewer)", fixtures.Alice, rebac.RelationWorkspaceViewer, ws, true},
		{"alice is not workspace owner", fixtures.Alice, rebac.RelationWorkspaceOwner, ws, false},
		{"erin is workspace owner (via team#admin)", fixtures.Erin, rebac.RelationWorkspaceOwner, ws, true},
		{"erin is workspace editor (owner implies editor)", fixtures.Erin, rebac.RelationWorkspaceEditor, ws, true},
		{"erin is workspace viewer (owner implies viewer)", fixtures.Erin, rebac.RelationWorkspaceViewer, ws, true},
		{"bob is workspace viewer (direct)", fixtures.Bob, rebac.RelationWorkspaceViewer, ws, true},
		{"bob is not workspace editor", fixtures.Bob, rebac.RelationWorkspaceEditor, ws, false},
		{"casey is not workspace viewer", fixtures.Casey, rebac.RelationWorkspaceViewer, ws, false},

		// ── Team relations ────────────────────────────────────────────────────
		{"alice is team member (direct)", fixtures.Alice, rebac.RelationTeamMember, team, true},
		{"erin is team admin (direct)", fixtures.Erin, rebac.RelationTeamAdmin, team, true},
		{"erin is team member (admin implies member)", fixtures.Erin, rebac.RelationTeamMember, team, true},
		{"bob is not team member", fixtures.Bob, rebac.RelationTeamMember, team, false},
		{"casey is not team member", fixtures.Casey, rebac.RelationTeamMember, team, false},
	}
}

// Run executes every canonical case against check and fails the test on any
// mismatch. Pass evaluator.Evaluate (from-scratch) or service.Check (OpenFGA).
func Run(t *testing.T, check CheckFunc) {
	t.Helper()
	ctx := context.Background()

	for _, c := range Cases() {
		t.Run(c.Name, func(t *testing.T) {
			result, err := check(ctx, rebac.CheckRequest{
				User:     c.User,
				Relation: c.Relation,
				Object:   c.Object,
			})
			if err != nil {
				t.Fatalf("check returned error: %v", err)
			}
			if result.Allowed != c.Allowed {
				t.Errorf("Check(%s, %s, %s) = allowed:%v, want allowed:%v",
					c.User, c.Relation, c.Object, result.Allowed, c.Allowed)
			}
		})
	}
}
