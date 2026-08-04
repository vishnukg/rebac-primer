package authz_test

import (
	"slices"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/rebac"
)

func TestGraphEvaluator_TeamMemberCanEditDocument(t *testing.T) {
	// Arrange: alice is a member of platformTeam, which is an editor of
	// productWorkspace. roadmapDocument lives in productWorkspace. The graph
	// traversal should resolve this chain and grant can_edit.
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	ev := authz.NewGraphEvaluator(store)
	req := rebac.CheckRequest{
		Subject:  fixtures.Alice,
		Action:   rebac.ActionDocumentEdit,
		Resource: fixtures.RoadmapDocument,
	}

	// Act
	result, err := ev.Evaluate(t.Context(), req)

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
	if !slices.Contains(result.Trace, wantStep) {
		t.Errorf("expected trace to contain:\n  %q\ngot trace:", wantStep)
		for _, line := range result.Trace {
			t.Logf("  %s", line)
		}
	}
}

func TestGraphEvaluator_BobCanReadButNotEdit(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	ev := authz.NewGraphEvaluator(store)
	ctx := t.Context()

	// Act: check both actions.
	readResult, err := ev.Evaluate(ctx, rebac.CheckRequest{
		Subject:  fixtures.Bob,
		Action:   rebac.ActionDocumentRead,
		Resource: fixtures.RoadmapDocument,
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

	editResult, err := ev.Evaluate(ctx, rebac.CheckRequest{
		Subject:  fixtures.Bob,
		Action:   rebac.ActionDocumentEdit,
		Resource: fixtures.RoadmapDocument,
	})
	if err != nil {
		t.Fatalf("unexpected error on edit check: %v", err)
	}

	// Assert
	if editResult.Allowed {
		t.Error("expected bob can_edit=false but got true")
	}
}

func TestGraphEvaluator_CaseyIsDenied(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	ev := authz.NewGraphEvaluator(store)
	req := rebac.CheckRequest{
		Subject:  fixtures.Casey,
		Action:   rebac.ActionDocumentEdit,
		Resource: fixtures.RoadmapDocument,
	}

	// Act
	result, err := ev.Evaluate(t.Context(), req)

	// Assert
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
	// Arrange
	// These malformed low-level relationships make two team usersets refer to each
	// other. Service.WriteRelationships would reject them, but the evaluator must still
	// terminate if corrupt data bypasses that boundary.
	teamA := rebac.Team("a")
	teamB := rebac.Team("b")
	workspace := rebac.Workspace("cyclic")
	document := rebac.Document("cyclic")
	store := authz.NewInMemoryStore(
		rebac.NewRelationship(rebac.SubjectSet(teamB, rebac.RelationTeamMember), rebac.RelationTeamMember, teamA),
		rebac.NewRelationship(rebac.SubjectSet(teamA, rebac.RelationTeamMember), rebac.RelationTeamMember, teamB),
		rebac.NewRelationship(rebac.SubjectSet(teamA, rebac.RelationTeamMember), rebac.RelationWorkspaceEditor, workspace),
		rebac.NewRelationship(rebac.Subject(workspace), rebac.RelationDocumentWorkspace, document),
	)
	ev := authz.NewGraphEvaluator(store)
	req := rebac.CheckRequest{
		Subject:  fixtures.Casey,
		Action:   rebac.ActionDocumentEdit,
		Resource: document,
	}

	// Act
	result, err := ev.Evaluate(t.Context(), req)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected a cycle with no direct user relationship to be denied")
	}
	if !slices.Contains(result.Trace, "Cycle detected at team:a#member; stop this branch") {
		t.Errorf("expected trace to report the cycle, got:")
		for _, line := range result.Trace {
			t.Logf("  %s", line)
		}
	}
}

func TestGraphEvaluator_IgnoresStoredActionValue(t *testing.T) {
	// Arrange
	// can_edit is computed from editor; the model does not permit a can_edit
	// relationship. Seed the low-level store directly to prove corrupted data cannot
	// bypass the model and over-grant access.
	store := authz.NewInMemoryStore(rebac.NewRelationship(
		rebac.Subject(fixtures.Casey),
		rebac.Relation(rebac.ActionDocumentEdit),
		fixtures.RoadmapDocument,
	))
	ev := authz.NewGraphEvaluator(store)

	// Act
	result, err := ev.Evaluate(t.Context(), rebac.CheckRequest{
		Subject:  fixtures.Casey,
		Action:   rebac.ActionDocumentEdit,
		Resource: fixtures.RoadmapDocument,
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("invalid relationship using can_edit as its relation granted access; want denial")
	}
}

func TestGraphEvaluator_TeamAdminIsAlsoMember(t *testing.T) {
	// Arrange
	extra := rebac.NewRelationship(
		rebac.Subject(fixtures.Casey),
		rebac.RelationTeamAdmin,
		fixtures.PlatformTeam,
	)
	relationships := append(fixtures.SeedRelationships(), extra)
	store := authz.NewInMemoryStore(relationships...)
	ev := authz.NewGraphEvaluator(store)
	req := rebac.CheckRequest{
		Subject:  fixtures.Casey,
		Action:   rebac.ActionDocumentEdit,
		Resource: fixtures.RoadmapDocument,
	}

	// Act
	result, err := ev.Evaluate(t.Context(), req)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected team admin to inherit member and can_edit, but got false")
	}
	wantStep := "team:platformTeam member includes admin"
	if !slices.Contains(result.Trace, wantStep) {
		t.Errorf("expected trace to contain %q", wantStep)
		for _, line := range result.Trace {
			t.Logf("  trace: %s", line)
		}
	}
}

// TestGraphEvaluator_ActionMatrix uses a table-driven test to verify the
// full action matrix for the three fixture users against the roadmap document.
func TestGraphEvaluator_ActionMatrix(t *testing.T) {
	// Arrange
	rows := []struct {
		name    string
		subject rebac.Resource
		action  rebac.Action
		want    bool
	}{
		// alice — inherits editor via team → workspace → document
		{"editor_can_read", fixtures.Alice, rebac.ActionDocumentRead, true},
		{"editor_can_comment", fixtures.Alice, rebac.ActionDocumentComment, true},
		{"editor_can_edit", fixtures.Alice, rebac.ActionDocumentEdit, true},
		{"editor_cannot_delete", fixtures.Alice, rebac.ActionDocumentDelete, false},

		// bob — inherits viewer via workspace → document
		{"viewer_can_read", fixtures.Bob, rebac.ActionDocumentRead, true},
		{"viewer_can_comment", fixtures.Bob, rebac.ActionDocumentComment, true},
		{"viewer_cannot_edit", fixtures.Bob, rebac.ActionDocumentEdit, false},
		{"viewer_cannot_delete", fixtures.Bob, rebac.ActionDocumentDelete, false},

		// casey — no relationships, no path
		{"outside_cannot_read", fixtures.Casey, rebac.ActionDocumentRead, false},
		{"outside_cannot_edit", fixtures.Casey, rebac.ActionDocumentEdit, false},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			// Arrange: every subtest owns its evaluator and store.
			store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
			ev := authz.NewGraphEvaluator(store)

			// Act
			result, err := ev.Evaluate(t.Context(), rebac.CheckRequest{
				Subject:  row.subject,
				Action:   row.action,
				Resource: fixtures.RoadmapDocument,
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

// BenchmarkGraphEvaluator_Evaluate measures a single graph traversal.
// Run with: go test -bench=. -benchtime=5s ./internal/authz
func BenchmarkGraphEvaluator_Evaluate(b *testing.B) {
	store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
	ev := authz.NewGraphEvaluator(store)
	req := rebac.CheckRequest{
		Subject:  fixtures.Alice,
		Action:   rebac.ActionDocumentEdit,
		Resource: fixtures.RoadmapDocument,
	}
	ctx := b.Context()

	b.ResetTimer()
	for range b.N {
		if _, err := ev.Evaluate(ctx, req); err != nil {
			b.Fatal(err)
		}
	}
}
