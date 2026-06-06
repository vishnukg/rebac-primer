// Package contract defines the canonical authorization truth table — the single
// source of truth for "what the model means" — and a runner that checks any
// backend against it.
//
// # Why this exists
//
// The repo encodes the same permission model three times: the from-scratch Go
// rule tables (permissionmodel.go), the TypeScript tables (permissionModel.ts),
// and the OpenFGA DSL (deployments/openfga/model.fga). Nothing forces them to
// agree. This package pins the intended behavior as data so a drift in any one of
// them fails a test instead of silently changing who can access what.
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
	"rebac-primer/internal/shared"
)

// CheckFunc is the single operation a backend must provide to be held to the
// contract: answer one CheckRequest. Both authz.Evaluator.Evaluate and
// authz.Service.Check have this exact signature.
type CheckFunc func(context.Context, shared.CheckRequest) (shared.CheckResult, error)

// Case is one row of the truth table: a question and its required answer.
type Case struct {
	Name     string
	User     shared.Object
	Relation shared.Relation
	Object   shared.Object
	Allowed  bool
}

// Cases returns the canonical allow/deny matrix for the standard fixture
// scenario (fixtures.SeedRelationshipTuples): alice is a platform-team member,
// the team edits the product workspace, bob is a direct workspace viewer, casey
// has no relationships, and the roadmap document lives in the workspace.
//
// Every backend must produce these exact answers. To run it against OpenFGA, the
// store must hold the same tuples (the policy tuples from deployments/openfga/
// seed.sh plus the document's workspace tuple, which the server writes on
// startup).
func Cases() []Case {
	doc := fixtures.RoadmapDocument
	ws := fixtures.ProductWorkspace
	team := fixtures.PlatformTeam

	return []Case{
		// ── Document computed permissions ─────────────────────────────────────
		// alice: team member → workspace editor → document editor (inherited).
		{"alice can_read roadmap", fixtures.Alice, shared.RelationDocumentCanRead, doc, true},
		{"alice can_comment roadmap", fixtures.Alice, shared.RelationDocumentCanComment, doc, true},
		{"alice can_edit roadmap", fixtures.Alice, shared.RelationDocumentCanEdit, doc, true},
		{"alice cannot can_delete roadmap (not owner)", fixtures.Alice, shared.RelationDocumentCanDelete, doc, false},

		// bob: direct workspace viewer → document viewer (inherited).
		{"bob can_read roadmap", fixtures.Bob, shared.RelationDocumentCanRead, doc, true},
		{"bob can_comment roadmap", fixtures.Bob, shared.RelationDocumentCanComment, doc, true},
		{"bob cannot can_edit roadmap (viewer only)", fixtures.Bob, shared.RelationDocumentCanEdit, doc, false},
		{"bob cannot can_delete roadmap", fixtures.Bob, shared.RelationDocumentCanDelete, doc, false},

		// casey: no relationships → no access.
		{"casey cannot can_read roadmap", fixtures.Casey, shared.RelationDocumentCanRead, doc, false},
		{"casey cannot can_edit roadmap", fixtures.Casey, shared.RelationDocumentCanEdit, doc, false},
		{"casey cannot can_delete roadmap", fixtures.Casey, shared.RelationDocumentCanDelete, doc, false},

		// ── Workspace base relations ──────────────────────────────────────────
		{"alice is workspace editor (via team#member)", fixtures.Alice, shared.RelationWorkspaceEditor, ws, true},
		{"alice is workspace viewer (editor implies viewer)", fixtures.Alice, shared.RelationWorkspaceViewer, ws, true},
		{"alice is not workspace owner", fixtures.Alice, shared.RelationWorkspaceOwner, ws, false},
		{"bob is workspace viewer (direct)", fixtures.Bob, shared.RelationWorkspaceViewer, ws, true},
		{"bob is not workspace editor", fixtures.Bob, shared.RelationWorkspaceEditor, ws, false},
		{"casey is not workspace viewer", fixtures.Casey, shared.RelationWorkspaceViewer, ws, false},

		// ── Team relations ────────────────────────────────────────────────────
		{"alice is team member (direct)", fixtures.Alice, shared.RelationTeamMember, team, true},
		{"bob is not team member", fixtures.Bob, shared.RelationTeamMember, team, false},
		{"casey is not team member", fixtures.Casey, shared.RelationTeamMember, team, false},
	}
}

// Run executes every canonical case against check and fails the test on any
// mismatch. Pass evaluator.Evaluate (from-scratch) or service.Check (OpenFGA).
func Run(t *testing.T, check CheckFunc) {
	t.Helper()
	ctx := context.Background()

	for _, c := range Cases() {
		t.Run(c.Name, func(t *testing.T) {
			result, err := check(ctx, shared.CheckRequest{
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
