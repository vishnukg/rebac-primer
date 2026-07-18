package authz_test

import (
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/authz/contract"
	"rebac-primer/internal/fixtures"
)

// TestContract_FromScratchEvaluator holds the in-process graph evaluator to the
// canonical model contract (internal/authz/contract). It is the drift guard: if
// internal/authz/model.go ever diverges from the intended model — the same model
// that deployments/openfga/model.fga encodes — this test fails, pointing at the
// exact (user, relation, object) that changed.
func TestContract_FromScratchEvaluator(t *testing.T) {
	// Arrange
	tuples := append(fixtures.SeedRelationshipTuples(), contract.ExtraTuples()...)
	store := authz.NewInMemoryStore(tuples...)
	ev := authz.NewGraphEvaluator(store)

	// Act: the contract runs each check as an independent subtest.
	contract.Run(t, ev.Evaluate)
	// Assert: contract.Run reports each result through testing.T.
}
