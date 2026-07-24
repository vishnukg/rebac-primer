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
	Name       string
	Subject    rebac.Resource
	Permission rebac.Permission
	Resource   rebac.Resource
	Allowed    bool
}

// ExtraRelationships returns contract-only relationships that exercise policy paths not
// covered by the demo story alone. They are deliberately kept out of
// fixtures.SeedRelationships so the public demo remains small, but both
// backends write them before running this contract.
func ExtraRelationships() []rebac.Relationship {
	return []rebac.Relationship{
		// Direct document ownership: proves owner -> can_delete and owner ->
		// editor -> viewer -> can_read/can_comment.
		rebac.NewRelationship(
			rebac.Subject(fixtures.Dana),
			rebac.RelationDocumentOwner,
			fixtures.RoadmapDocument,
		),
		// Team admin path: proves admin implies member, and team#admin can own a
		// workspace.
		rebac.NewRelationship(
			rebac.Subject(fixtures.Erin),
			rebac.RelationTeamAdmin,
			fixtures.PlatformTeam,
		),
		rebac.NewRelationship(
			rebac.SubjectSet(fixtures.PlatformTeam, rebac.RelationTeamAdmin),
			rebac.RelationWorkspaceOwner,
			fixtures.ProductWorkspace,
		),
	}
}

// Cases returns the canonical allow/deny matrix for the standard fixture
// scenario plus ExtraRelationships: alice is a platform-team member, the team edits the
// product workspace, bob is a direct workspace viewer, casey has no
// relationships, dana directly owns the roadmap document, erin is a platform
// team admin, and the roadmap document lives in the workspace.
//
// Every backend must produce these exact answers. The OpenFGA contract test
// writes SeedRelationships and ExtraRelationships itself. The store must hold no
// unrelated relationships—in particular, starting the application seeds a demo document
// owned by alice, and that owner relationship changes the can_delete answers this
// contract pins down.
func Cases() []Case {
	doc := fixtures.RoadmapDocument
	ws := fixtures.ProductWorkspace

	return []Case{
		// ── Document computed permissions ─────────────────────────────────────
		// alice: team member → workspace editor → document editor (inherited).
		{"alice can_read roadmap", fixtures.Alice, rebac.PermissionDocumentRead, doc, true},
		{"alice can_comment roadmap", fixtures.Alice, rebac.PermissionDocumentComment, doc, true},
		{"alice can_edit roadmap", fixtures.Alice, rebac.PermissionDocumentEdit, doc, true},
		{"alice lacks can_delete on roadmap (not owner)", fixtures.Alice, rebac.PermissionDocumentDelete, doc, false},

		// dana: direct document owner.
		{"dana can_read roadmap (direct owner)", fixtures.Dana, rebac.PermissionDocumentRead, doc, true},
		{"dana can_comment roadmap (direct owner)", fixtures.Dana, rebac.PermissionDocumentComment, doc, true},
		{"dana can_edit roadmap (direct owner)", fixtures.Dana, rebac.PermissionDocumentEdit, doc, true},
		{"dana can_delete roadmap (direct owner)", fixtures.Dana, rebac.PermissionDocumentDelete, doc, true},

		// erin: team admin -> workspace owner via team#admin -> document owner.
		{"erin can_read roadmap (workspace owner)", fixtures.Erin, rebac.PermissionDocumentRead, doc, true},
		{"erin can_edit roadmap (workspace owner)", fixtures.Erin, rebac.PermissionDocumentEdit, doc, true},
		{"erin can_delete roadmap (workspace owner)", fixtures.Erin, rebac.PermissionDocumentDelete, doc, true},

		// bob: direct workspace viewer → document viewer (inherited).
		{"bob can_read roadmap", fixtures.Bob, rebac.PermissionDocumentRead, doc, true},
		{"bob can_comment roadmap", fixtures.Bob, rebac.PermissionDocumentComment, doc, true},
		{"bob lacks can_edit on roadmap (viewer only)", fixtures.Bob, rebac.PermissionDocumentEdit, doc, false},
		{"bob lacks can_delete on roadmap", fixtures.Bob, rebac.PermissionDocumentDelete, doc, false},

		// casey: no relationships → no access.
		{"casey lacks can_read on roadmap", fixtures.Casey, rebac.PermissionDocumentRead, doc, false},
		{"casey lacks can_edit on roadmap", fixtures.Casey, rebac.PermissionDocumentEdit, doc, false},
		{"casey lacks can_delete on roadmap", fixtures.Casey, rebac.PermissionDocumentDelete, doc, false},

		// ── Workspace permission ──────────────────────────────────────────────
		{"alice can create documents in workspace", fixtures.Alice, rebac.PermissionWorkspaceCreateDocument, ws, true},
		{"erin can create documents in workspace", fixtures.Erin, rebac.PermissionWorkspaceCreateDocument, ws, true},
		{"bob cannot create documents in workspace", fixtures.Bob, rebac.PermissionWorkspaceCreateDocument, ws, false},
		{"casey cannot create documents in workspace", fixtures.Casey, rebac.PermissionWorkspaceCreateDocument, ws, false},
	}
}

// Run executes every canonical case against check and fails the test on any
// mismatch. Pass evaluator.Evaluate (from-scratch) or service.Check (OpenFGA).
func Run(t *testing.T, check CheckFunc) {
	t.Helper()

	for _, c := range Cases() {
		t.Run(c.Name, func(t *testing.T) {
			// Arrange
			ctx := t.Context()
			req := rebac.CheckRequest{
				Subject:    c.Subject,
				Permission: c.Permission,
				Resource:   c.Resource,
			}

			// Act
			result, err := check(ctx, req)

			// Assert
			if err != nil {
				t.Fatalf("check returned error: %v", err)
			}
			if result.Allowed != c.Allowed {
				t.Errorf("Check(%s, %s, %s) = allowed:%v, want allowed:%v",
					c.Subject, c.Permission, c.Resource, result.Allowed, c.Allowed)
			}
		})
	}
}
