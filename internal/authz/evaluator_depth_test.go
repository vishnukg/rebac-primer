package authz

// White-box test (package graph) so it can set the unexported maxDepth field to a
// small value and trip the depth guard without building a 100-deep graph.

import (
	"context"
	"fmt"
	"testing"

	"rebac-primer/internal/rebac"
)

func TestGraphEvaluator_ExceedingMaxDepthReturnsError(t *testing.T) {
	// Build an acyclic chain of subject-sets: team:t0#member is satisfied by
	// team:t1#member, which is satisfied by team:t2#member, and so on. Each hop is
	// a distinct (object, relation) pair, so the cycle guard never fires — only the
	// depth guard can stop it.
	var seed []rebac.TupleKey
	const chain = 6
	for i := range chain {
		obj := rebac.Team(fmt.Sprintf("t%d", i))
		next := rebac.SubjectSet(rebac.Team(fmt.Sprintf("t%d", i+1)), rebac.RelationTeamMember)
		seed = append(seed, rebac.Tuple(obj, rebac.RelationTeamMember, next))
	}

	ev := NewGraphEvaluator(NewInMemoryStore(seed...))
	ev.maxDepth = 2 // force the guard to trip well before the chain ends

	_, err := ev.Evaluate(context.Background(), rebac.CheckRequest{
		User:     rebac.User("nobody"),
		Relation: rebac.RelationTeamMember,
		Object:   rebac.Team("t0"),
	})
	if err == nil {
		t.Fatal("expected a max-depth error, got nil")
	}
}
