package graph

// White-box test (package graph) so it can set the unexported maxDepth field to a
// small value and trip the depth guard without building a 100-deep graph.

import (
	"context"
	"fmt"
	"testing"

	authzdb "rebac-primer/internal/authz/adapters/db"
	"rebac-primer/internal/shared"
)

func TestGraphEvaluator_ExceedingMaxDepthReturnsError(t *testing.T) {
	// Build an acyclic chain of subject-sets: team:t0#member is satisfied by
	// team:t1#member, which is satisfied by team:t2#member, and so on. Each hop is
	// a distinct (object, relation) pair, so the cycle guard never fires — only the
	// depth guard can stop it.
	var seed []shared.TupleKey
	const chain = 6
	for i := 0; i < chain; i++ {
		obj := shared.Team(fmt.Sprintf("t%d", i))
		next := shared.SubjectSet(shared.Team(fmt.Sprintf("t%d", i+1)), shared.RelationTeamMember)
		seed = append(seed, shared.Tuple(obj, shared.RelationTeamMember, next))
	}

	ev := NewGraphEvaluator(authzdb.New(seed...))
	ev.maxDepth = 2 // force the guard to trip well before the chain ends

	_, err := ev.Evaluate(context.Background(), shared.CheckRequest{
		User:     shared.User("nobody"),
		Relation: shared.RelationTeamMember,
		Object:   shared.Team("t0"),
	})
	if err == nil {
		t.Fatal("expected a max-depth error, got nil")
	}
}
