package graph_test

import (
	"context"
	"testing"

	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/shared"
)

// TestTrace is a learning aid, not an assertion-heavy test. It runs a few checks
// and PRINTS the evaluator's full step-by-step trace so you can watch the graph
// traversal happen. Unlike the other tests, it logs the trace on success too.
//
// Run it and read the output top to bottom:
//
//	go test -v -run TestTrace ./internal/authz/adapters/graph/
//
// Each sub-test is one question. The trace lines are the exact steps the
// evaluator took to answer it — see docs/27-graph-evaluator-walkthrough.md for a
// line-by-line explanation of the alice/can_edit trace.
func TestTrace(t *testing.T) {
	cases := []struct {
		name     string
		user     shared.Object
		relation shared.Relation
		object   shared.Object
	}{
		{"alice can_edit roadmap (allowed via team->workspace)", fixtures.Alice, shared.RelationDocumentCanEdit, fixtures.RoadmapDocument},
		{"bob can_read roadmap (allowed via direct viewer)", fixtures.Bob, shared.RelationDocumentCanRead, fixtures.RoadmapDocument},
		{"bob can_edit roadmap (denied: viewer is not editor)", fixtures.Bob, shared.RelationDocumentCanEdit, fixtures.RoadmapDocument},
		{"casey can_read roadmap (denied: no path)", fixtures.Casey, shared.RelationDocumentCanRead, fixtures.RoadmapDocument},
	}

	ev := newEvaluator()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ev.Evaluate(context.Background(), shared.CheckRequest{
				User:     tc.user,
				Relation: tc.relation,
				Object:   tc.object,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for i, line := range result.Trace {
				t.Logf("  [%d] %s", i, line)
			}
		})
	}
}
