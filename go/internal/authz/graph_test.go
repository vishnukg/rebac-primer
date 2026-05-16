package authz_test

import (
	"context"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/fixtures"
)

// seedStore builds a store from the standard fixture tuples.
// Optional extra tuples can be appended for specific test cases.
func seedStore(extra ...authz.TupleKey) *authz.InMemoryTupleStore {
	all := append(fixtures.SeedRelationshipTuples(), extra...)
	return authz.NewInMemoryTupleStore(all...)
}

func TestGraphAuthorizer_TeamMemberCanEditDocument(t *testing.T) {
	// Arrange: workspaceEditor is a member of platformTeam, which is an editor of
	// productWorkspace. roadmapDocument lives in productWorkspace. The graph
	// traversal should resolve this chain and grant can_edit.
	store := seedStore()
	auth := authz.NewGraphAuthorizer(store)
	req := authz.CheckRequest{
		User:     fixtures.WorkspaceEditor,
		Relation: authz.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	}

	// Act
	result, err := auth.Check(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed=true but got false")
		for _, line := range result.Trace {
			t.Logf("  trace: %s", line)
		}
	}
	// The trace must show the subject-set resolution step so readers can see how
	// the chain team → workspace → document is walked.
	wantStep := "Resolve subject set team:platformTeam#member: does it contain user:workspaceEditor?"
	found := false
	for _, line := range result.Trace {
		if line == wantStep {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected trace to contain:\n  %q\ngot trace:", wantStep)
		for _, line := range result.Trace {
			t.Logf("  %s", line)
		}
	}
}

func TestGraphAuthorizer_WorkspaceViewerCanReadButNotEdit(t *testing.T) {
	// Arrange: workspaceViewer has viewer on productWorkspace.
	// viewer → can_read should be allowed; viewer → can_edit should be denied.
	store := seedStore()
	auth := authz.NewGraphAuthorizer(store)
	ctx := context.Background()

	// Act: can_read
	readResult, err := auth.Check(ctx, authz.CheckRequest{
		User:     fixtures.WorkspaceViewer,
		Relation: authz.RelationDocumentCanRead,
		Object:   fixtures.RoadmapDocument,
	})

	// Assert: can_read
	if err != nil {
		t.Fatalf("unexpected error on read check: %v", err)
	}
	if !readResult.Allowed {
		t.Error("expected workspaceViewer can_read=true but got false")
		for _, line := range readResult.Trace {
			t.Logf("  trace: %s", line)
		}
	}

	// Act: can_edit
	editResult, err := auth.Check(ctx, authz.CheckRequest{
		User:     fixtures.WorkspaceViewer,
		Relation: authz.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	})

	// Assert: can_edit
	if err != nil {
		t.Fatalf("unexpected error on edit check: %v", err)
	}
	if editResult.Allowed {
		t.Error("expected workspaceViewer can_edit=false but got true")
		for _, line := range editResult.Trace {
			t.Logf("  trace: %s", line)
		}
	}
}

func TestGraphAuthorizer_OutsideCollaboratorIsDenied(t *testing.T) {
	// Arrange: outsideCollaborator has no tuples in the graph.
	store := seedStore()
	auth := authz.NewGraphAuthorizer(store)
	req := authz.CheckRequest{
		User:     fixtures.OutsideCollaborator,
		Relation: authz.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	}

	// Act
	result, err := auth.Check(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected outsideCollaborator can_edit=false but got true")
	}
	if last := result.Trace[len(result.Trace)-1]; last != "Result: denied" {
		t.Errorf("expected last trace line %q, got %q", "Result: denied", last)
	}
}

func TestGraphAuthorizer_CycleDetectionDoesNotHang(t *testing.T) {
	// Arrange: a document whose workspace pointer points back to itself creates a
	// cycle in the graph. The visited set must prevent infinite recursion.
	cyclicDoc := authz.Document("cyclicDoc")
	store := authz.NewInMemoryTupleStore(
		authz.Tuple(cyclicDoc, authz.RelationDocumentWorkspace, authz.Subject(cyclicDoc)),
		authz.Tuple(cyclicDoc, authz.RelationDocumentViewer, authz.Subject(fixtures.WorkspaceViewer)),
	)
	auth := authz.NewGraphAuthorizer(store)
	req := authz.CheckRequest{
		User:     fixtures.WorkspaceViewer,
		Relation: authz.RelationDocumentCanRead,
		Object:   cyclicDoc,
	}

	// Act
	result, err := auth.Check(context.Background(), req)

	// Assert: the direct viewer tuple must still grant can_read even with the cycle.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected can_read=true even with cyclic workspace pointer")
		for _, line := range result.Trace {
			t.Logf("  trace: %s", line)
		}
	}
}

func TestGraphAuthorizer_TeamAdminIsAlsoMember(t *testing.T) {
	// Arrange: outsideCollaborator is an admin of platformTeam.
	// The model rule "team.member includes team.admin" must make them a member too.
	extra := authz.Tuple(fixtures.PlatformTeam, authz.RelationTeamAdmin, authz.Subject(fixtures.OutsideCollaborator))
	store := seedStore(extra)
	auth := authz.NewGraphAuthorizer(store)
	req := authz.CheckRequest{
		User:     fixtures.OutsideCollaborator,
		Relation: authz.RelationTeamMember,
		Object:   fixtures.PlatformTeam,
	}

	// Act
	result, err := auth.Check(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected team admin to also satisfy member=true but got false")
		for _, line := range result.Trace {
			t.Logf("  trace: %s", line)
		}
	}
	wantStep := "team.member includes team.admin"
	found := false
	for _, line := range result.Trace {
		if line == wantStep {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected trace to contain %q", wantStep)
		for _, line := range result.Trace {
			t.Logf("  trace: %s", line)
		}
	}
}

// TestGraphAuthorizer_PermissionMatrix uses a table-driven test to verify the
// full permission matrix for the three fixture users against the roadmap document.
//
// Table-driven tests shine when you have many inputs and one behaviour to check.
// Each row is a sub-test (t.Run) so failures identify themselves by name rather
// than by index. Run a single row with: go test -run TestGraphAuthorizer_PermissionMatrix/viewer_cannot_edit
func TestGraphAuthorizer_PermissionMatrix(t *testing.T) {
	store := seedStore()
	auth := authz.NewGraphAuthorizer(store)

	rows := []struct {
		name     string
		user     authz.Object
		relation authz.Relation
		want     bool
	}{
		// workspaceEditor — inherits editor via team → workspace → document
		{"editor_can_read", fixtures.WorkspaceEditor, authz.RelationDocumentCanRead, true},
		{"editor_can_comment", fixtures.WorkspaceEditor, authz.RelationDocumentCanComment, true},
		{"editor_can_edit", fixtures.WorkspaceEditor, authz.RelationDocumentCanEdit, true},
		{"editor_cannot_delete", fixtures.WorkspaceEditor, authz.RelationDocumentCanDelete, false},

		// workspaceViewer — inherits viewer via workspace → document
		{"viewer_can_read", fixtures.WorkspaceViewer, authz.RelationDocumentCanRead, true},
		{"viewer_can_comment", fixtures.WorkspaceViewer, authz.RelationDocumentCanComment, true},
		{"viewer_cannot_edit", fixtures.WorkspaceViewer, authz.RelationDocumentCanEdit, false},
		{"viewer_cannot_delete", fixtures.WorkspaceViewer, authz.RelationDocumentCanDelete, false},

		// outsideCollaborator — no tuples, no path
		{"outside_cannot_read", fixtures.OutsideCollaborator, authz.RelationDocumentCanRead, false},
		{"outside_cannot_edit", fixtures.OutsideCollaborator, authz.RelationDocumentCanEdit, false},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			// Arrange (already done above — shared store and authorizer)

			// Act
			result, err := auth.Check(context.Background(), authz.CheckRequest{
				User:     row.user,
				Relation: row.relation,
				Object:   fixtures.RoadmapDocument,
			})

			// Assert
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Allowed != row.want {
				t.Errorf("got allowed=%v, want %v", result.Allowed, row.want)
				for _, line := range result.Trace {
					t.Logf("  trace: %s", line)
				}
			}
		})
	}
}

// BenchmarkGraphAuthorizer_Check measures a single graph traversal end-to-end.
// Run with: go test -bench=. -benchtime=5s ./internal/authz/...
func BenchmarkGraphAuthorizer_Check(b *testing.B) {
	store := seedStore()
	auth := authz.NewGraphAuthorizer(store)
	req := authz.CheckRequest{
		User:     fixtures.WorkspaceEditor,
		Relation: authz.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	}
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		auth.Check(ctx, req) //nolint:errcheck // benchmark ignores errors
	}
}

// FuzzParseObject exercises ParseObject with arbitrary byte sequences.
// Run with: go test -fuzz=FuzzParseObject -fuzztime=30s ./internal/authz/...
//
// The fuzzer finds inputs that cause panics or unexpected behaviour. Our
// invariant: ParseObject must never panic, and valid round-trips must hold.
func FuzzParseObject(f *testing.F) {
	// Seed corpus — known valid and known invalid inputs.
	f.Add("user:alice")
	f.Add("team:platformTeam")
	f.Add("workspace:productWorkspace")
	f.Add("document:roadmapDocument")
	f.Add("") // empty — must return error, not panic
	f.Add(":")
	f.Add("user:")
	f.Add(":alice")
	f.Add("unknown:something")

	f.Fuzz(func(t *testing.T, s string) {
		// ParseObject must never panic regardless of input.
		typ, id, err := authz.ParseObject(s)
		if err != nil {
			return // invalid input — that is fine
		}
		// If parsing succeeded, round-tripping through the constructor must
		// produce the same string.
		var obj authz.Object
		switch typ {
		case authz.ObjectTypeUser:
			obj = authz.User(id)
		case authz.ObjectTypeTeam:
			obj = authz.Team(id)
		case authz.ObjectTypeWorkspace:
			obj = authz.Workspace(id)
		case authz.ObjectTypeDocument:
			obj = authz.Document(id)
		default:
			t.Fatalf("ParseObject returned unrecognised type %q", typ)
		}
		if string(obj) != s {
			t.Errorf("round-trip failed: ParseObject(%q) -> type=%s id=%s -> Object=%q", s, typ, id, obj)
		}
	})
}
