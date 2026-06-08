package authz_test

import (
	"context"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/rebac"
)

// seedStore builds a tuple store from the standard fixture tuples.
// Optional extra tuples can be appended for specific test cases.
func seedStore(extra ...rebac.TupleKey) *authz.InMemoryStore {
	all := append(fixtures.SeedRelationshipTuples(), extra...)
	return authz.NewInMemoryStore(all...)
}

// newEvaluator is a helper that wraps seedStore + NewGraphEvaluator.
func newEvaluator(extra ...rebac.TupleKey) *authz.GraphEvaluator {
	return authz.NewGraphEvaluator(seedStore(extra...))
}

func TestGraphEvaluator_TeamMemberCanEditDocument(t *testing.T) {
	// Arrange: alice is a member of platformTeam, which is an editor of
	// productWorkspace. roadmapDocument lives in productWorkspace. The graph
	// traversal should resolve this chain and grant can_edit.
	ev := newEvaluator()
	req := rebac.CheckRequest{
		User:     fixtures.Alice,
		Relation: rebac.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	}

	// Act
	result, err := ev.Evaluate(context.Background(), req)

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
	wantStep := "Resolve subject set team:platformTeam#member: does it contain user:alice?"
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

func TestGraphEvaluator_BobCanReadButNotEdit(t *testing.T) {
	ev := newEvaluator()
	ctx := context.Background()

	// can_read
	readResult, err := ev.Evaluate(ctx, rebac.CheckRequest{
		User:     fixtures.Bob,
		Relation: rebac.RelationDocumentCanRead,
		Object:   fixtures.RoadmapDocument,
	})
	if err != nil {
		t.Fatalf("unexpected error on read check: %v", err)
	}
	if !readResult.Allowed {
		t.Error("expected bob can_read=true but got false")
		for _, line := range readResult.Trace {
			t.Logf("  trace: %s", line)
		}
	}

	// can_edit
	editResult, err := ev.Evaluate(ctx, rebac.CheckRequest{
		User:     fixtures.Bob,
		Relation: rebac.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	})
	if err != nil {
		t.Fatalf("unexpected error on edit check: %v", err)
	}
	if editResult.Allowed {
		t.Error("expected bob can_edit=false but got true")
	}
}

func TestGraphEvaluator_CaseyIsDenied(t *testing.T) {
	ev := newEvaluator()
	req := rebac.CheckRequest{
		User:     fixtures.Casey,
		Relation: rebac.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	}

	result, err := ev.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected casey can_edit=false but got true")
	}
	if last := result.Trace[len(result.Trace)-1]; last != "Result: denied" {
		t.Errorf("expected last trace line %q, got %q", "Result: denied", last)
	}
}

func TestGraphEvaluator_CycleDetectionDoesNotHang(t *testing.T) {
	// A document whose workspace pointer points back to itself creates a cycle.
	// The visited set must prevent infinite recursion.
	cyclicDoc := rebac.Document("cyclicDoc")
	store := authz.NewInMemoryStore(
		rebac.Tuple(cyclicDoc, rebac.RelationDocumentWorkspace, rebac.Subject(cyclicDoc)),
		rebac.Tuple(cyclicDoc, rebac.RelationDocumentViewer, rebac.Subject(fixtures.Bob)),
	)
	ev := authz.NewGraphEvaluator(store)
	req := rebac.CheckRequest{
		User:     fixtures.Bob,
		Relation: rebac.RelationDocumentCanRead,
		Object:   cyclicDoc,
	}

	result, err := ev.Evaluate(context.Background(), req)
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

func TestGraphEvaluator_TeamAdminIsAlsoMember(t *testing.T) {
	extra := rebac.Tuple(fixtures.PlatformTeam, rebac.RelationTeamAdmin, rebac.Subject(fixtures.Casey))
	ev := newEvaluator(extra)
	req := rebac.CheckRequest{
		User:     fixtures.Casey,
		Relation: rebac.RelationTeamMember,
		Object:   fixtures.PlatformTeam,
	}

	result, err := ev.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected team admin to also satisfy member=true but got false")
	}
	wantStep := "team:platformTeam member includes admin"
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

// TestGraphEvaluator_PermissionMatrix uses a table-driven test to verify the
// full permission matrix for the three fixture users against the roadmap document.
func TestGraphEvaluator_PermissionMatrix(t *testing.T) {
	ev := newEvaluator()

	rows := []struct {
		name     string
		user     rebac.Object
		relation rebac.Relation
		want     bool
	}{
		// alice — inherits editor via team → workspace → document
		{"editor_can_read", fixtures.Alice, rebac.RelationDocumentCanRead, true},
		{"editor_can_comment", fixtures.Alice, rebac.RelationDocumentCanComment, true},
		{"editor_can_edit", fixtures.Alice, rebac.RelationDocumentCanEdit, true},
		{"editor_cannot_delete", fixtures.Alice, rebac.RelationDocumentCanDelete, false},

		// bob — inherits viewer via workspace → document
		{"viewer_can_read", fixtures.Bob, rebac.RelationDocumentCanRead, true},
		{"viewer_can_comment", fixtures.Bob, rebac.RelationDocumentCanComment, true},
		{"viewer_cannot_edit", fixtures.Bob, rebac.RelationDocumentCanEdit, false},
		{"viewer_cannot_delete", fixtures.Bob, rebac.RelationDocumentCanDelete, false},

		// casey — no tuples, no path
		{"outside_cannot_read", fixtures.Casey, rebac.RelationDocumentCanRead, false},
		{"outside_cannot_edit", fixtures.Casey, rebac.RelationDocumentCanEdit, false},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			result, err := ev.Evaluate(context.Background(), rebac.CheckRequest{
				User:     row.user,
				Relation: row.relation,
				Object:   fixtures.RoadmapDocument,
			})
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

// BenchmarkGraphEvaluator_Evaluate measures a single graph traversal.
// Run with: go test -bench=. -benchtime=5s ./internal/authz/adapters/graph/...
func BenchmarkGraphEvaluator_Evaluate(b *testing.B) {
	ev := newEvaluator()
	req := rebac.CheckRequest{
		User:     fixtures.Alice,
		Relation: rebac.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	}
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		ev.Evaluate(ctx, req) //nolint:errcheck
	}
}

// FuzzParseObject exercises ParseObject with arbitrary byte sequences.
// Run with: go test -fuzz=FuzzParseObject -fuzztime=30s ./internal/authz/adapters/graph/...
func FuzzParseObject(f *testing.F) {
	f.Add("user:alice")
	f.Add("team:platformTeam")
	f.Add("workspace:productWorkspace")
	f.Add("document:roadmapDocument")
	f.Add("")
	f.Add(":")
	f.Add("user:")
	f.Add(":alice")
	f.Add("unknown:something")

	f.Fuzz(func(t *testing.T, s string) {
		typ, id, err := rebac.ParseObject(s)
		if err != nil {
			return
		}
		var obj rebac.Object
		switch typ {
		case rebac.ObjectTypeUser:
			obj = rebac.User(id)
		case rebac.ObjectTypeTeam:
			obj = rebac.Team(id)
		case rebac.ObjectTypeWorkspace:
			obj = rebac.Workspace(id)
		case rebac.ObjectTypeDocument:
			obj = rebac.Document(id)
		default:
			t.Fatalf("ParseObject returned unrecognised type %q", typ)
		}
		if string(obj) != s {
			t.Errorf("round-trip failed: ParseObject(%q) -> type=%s id=%s -> Object=%q", s, typ, id, obj)
		}
	})
}
