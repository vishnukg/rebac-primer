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
// contract: answer one CheckRequest. Both authz.Evaluator.Evaluate and
// authz.Service.Check have this exact signature.
type CheckFunc func(context.Context, rebac.CheckRequest) (rebac.CheckResult, error)

// Case is one row of the truth table: a question and its required answer.
type Case struct {
	Name     string
	User     rebac.Object
	Relation rebac.Relation
	Object   rebac.Object
	Allowed  bool
}

// Cases returns the canonical allow/deny matrix for the standard fixture
// scenario (fixtures.SeedRelationshipTuples): alice is a platform-team member,
// the team edits the product workspace, bob is a direct workspace viewer, casey
// has no relationships, and the roadmap document lives in the workspace.
//
// Every backend must produce these exact answers. To run it against OpenFGA, the
// store must hold the same tuples: the policy tuples from deployments/openfga/
// seed.sh plus the document's workspace tuple, which the OpenFGA contract test
// writes itself. The store must hold nothing else — in particular, starting the
// server seeds a demo document owned by alice, and that owner tuple changes the
// can_delete answers this contract pins down.
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
		{"bob is workspace viewer (direct)", fixtures.Bob, rebac.RelationWorkspaceViewer, ws, true},
		{"bob is not workspace editor", fixtures.Bob, rebac.RelationWorkspaceEditor, ws, false},
		{"casey is not workspace viewer", fixtures.Casey, rebac.RelationWorkspaceViewer, ws, false},

		// ── Team relations ────────────────────────────────────────────────────
		{"alice is team member (direct)", fixtures.Alice, rebac.RelationTeamMember, team, true},
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
