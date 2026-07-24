package openfga_test

import (
	"os"
	"testing"

	"rebac-primer/internal/authz/contract"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/openfga"
)

// TestContract_OpenFGA holds the OpenFGA backend to the *same* canonical contract
// as the from-scratch evaluator. When both pass, the two backends provably agree
// on the model — that is the parity guarantee.
//
// It skips unless a store is configured, so `go test ./...` stays green offline.
// To run it:
//
//	make openfga/up && make openfga/seed   # start OpenFGA, write model + policy tuples
//	set -a; . deployments/openfga/.ids.env; set +a
//	go test -run TestContract_OpenFGA ./internal/openfga
//
// Run it against a store containing this model, BEFORE starting the application:
// the application's startup seed makes alice the demo document's owner, and that
// extra owner tuple flips the can_delete answers the contract pins down. The test
// writes every relationship it needs, so the seed script is convenient but not a
// hidden data dependency.
func TestContract_OpenFGA(t *testing.T) {
	// Arrange
	apiURL := os.Getenv("OPENFGA_API_URL")
	storeID := os.Getenv("OPENFGA_STORE_ID")
	modelID := os.Getenv("OPENFGA_MODEL_ID")
	if apiURL == "" || storeID == "" || modelID == "" {
		t.Skip("set OPENFGA_API_URL, OPENFGA_STORE_ID, and OPENFGA_MODEL_ID to run the OpenFGA contract test")
	}

	svc, err := openfga.New(openfga.Config{APIURL: apiURL, StoreID: storeID, ModelID: modelID})
	if err != nil {
		t.Fatalf("new openfga service: %v", err)
	}

	// Arrange the complete relationship graph inside this test. WriteRelationships is
	// idempotent, so the same contract also works against a store previously
	// populated by seed.sh.
	tuples := append(fixtures.SeedRelationships(), contract.ExtraRelationships()...)
	err = svc.WriteRelationships(t.Context(), tuples)
	if err != nil {
		t.Fatalf("seed contract tuples: %v", err)
	}

	// Act and Assert: each contract row performs one Check and verifies its result.
	contract.Run(t, svc.Check)
}
