package authz_test

import (
	"testing"

	"rebac-primer/internal/authz/contract"
)

// TestContract_FromScratchEvaluator holds the in-process graph evaluator to the
// canonical model contract (internal/authz/contract). It is the drift guard: if
// internal/authz/model.go ever diverges from the intended model — the same model
// that deployments/openfga/model.fga encodes — this test fails, pointing at the
// exact (user, relation, object) that changed.
//
// newEvaluator (defined in evaluator_test.go) seeds the store with
// fixtures.SeedRelationshipTuples(), which is the scenario the contract describes.
func TestContract_FromScratchEvaluator(t *testing.T) {
	ev := newEvaluator()
	contract.Run(t, ev.Evaluate)
}
