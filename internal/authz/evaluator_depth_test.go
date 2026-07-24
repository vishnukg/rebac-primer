package authz

// White-box test (package graph) so it can set the unexported maxDepth field to a
// small value and trip the depth guard without building a 100-deep graph.

import (
	"fmt"
	"testing"

	"rebac-primer/internal/rebac"
)

func TestGraphEvaluator_ExceedingMaxDepthReturnsError(t *testing.T) {
	// Arrange
	// Build an acyclic chain of subject-sets: team:t0#member is satisfied by
	// team:t1#member, which is satisfied by team:t2#member, and so on. Each hop is
	// a distinct (resource, relation) pair, so the cycle guard never fires — only the
	// depth guard can stop it.
	var seed []rebac.Relationship
	const chain = 6
	for i := range chain {
		resource := rebac.Team(fmt.Sprintf("t%d", i))
		next := rebac.SubjectSet(rebac.Team(fmt.Sprintf("t%d", i+1)), rebac.RelationTeamMember)
		seed = append(seed, rebac.NewRelationship(next, rebac.RelationTeamMember, resource))
	}
	workspace := rebac.Workspace("deep")
	document := rebac.Document("deep")
	seed = append(seed,
		rebac.NewRelationship(
			rebac.SubjectSet(rebac.Team("t0"), rebac.RelationTeamMember),
			rebac.RelationWorkspaceEditor,
			workspace,
		),
		rebac.NewRelationship(
			rebac.Subject(workspace),
			rebac.RelationDocumentWorkspace,
			document,
		),
	)

	ev := NewGraphEvaluator(NewInMemoryStore(seed...))
	ev.maxDepth = 2 // force the guard to trip well before the chain ends

	// Act
	_, err := ev.Evaluate(t.Context(), rebac.CheckRequest{
		Subject:    rebac.User("nobody"),
		Permission: rebac.PermissionDocumentEdit,
		Resource:   document,
	})

	// Assert
	if err == nil {
		t.Fatal("expected a max-depth error, got nil")
	}
}
