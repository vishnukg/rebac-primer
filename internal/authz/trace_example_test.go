package authz_test

import (
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/rebac"
)

// TestTrace is a learning aid, not an assertion-heavy test. It runs a few checks
// and PRINTS the evaluator's full step-by-step trace so you can watch the graph
// traversal happen. Unlike the other tests, it logs the trace on success too.
//
// Run it and read the output top to bottom:
//
//	go test -v -run TestTrace ./internal/authz
//
// Each sub-test is one question. The trace lines are the exact steps the
// evaluator took to answer it — see docs/27-graph-evaluator-walkthrough.md for a
// line-by-line explanation of the alice/can_edit trace.
func TestTrace(t *testing.T) {
	cases := []struct {
		name     string
		subject  rebac.Resource
		action   rebac.Action
		resource rebac.Resource
	}{
		{"alice can_edit roadmap (allowed via team->workspace)", fixtures.Alice, rebac.ActionDocumentEdit, fixtures.RoadmapDocument},
		{"bob can_read roadmap (allowed via direct viewer)", fixtures.Bob, rebac.ActionDocumentRead, fixtures.RoadmapDocument},
		{"bob can_edit roadmap (denied: viewer is not editor)", fixtures.Bob, rebac.ActionDocumentEdit, fixtures.RoadmapDocument},
		{"casey can_read roadmap (denied: no path)", fixtures.Casey, rebac.ActionDocumentRead, fixtures.RoadmapDocument},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			store := authz.NewInMemoryStore(fixtures.SeedRelationships()...)
			ev := authz.NewGraphEvaluator(store)

			// Act
			result, err := ev.Evaluate(t.Context(), rebac.CheckRequest{
				Subject:  tc.subject,
				Action:   tc.action,
				Resource: tc.resource,
			})

			// Assert and report the trace.
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for i, line := range result.Trace {
				t.Logf("  [%d] %s", i, line)
			}
		})
	}
}
