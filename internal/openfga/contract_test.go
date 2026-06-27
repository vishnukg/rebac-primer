package openfga_test

import (
	"context"
	"os"
	"testing"

	"rebac-primer/internal/authz/contract"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/openfga"
	"rebac-primer/internal/rebac"
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
// Run it against a freshly seeded store, BEFORE starting the server: the server's
// startup seed makes alice the demo document's owner, and that extra owner tuple
// flips the can_delete answers the contract pins down.
func TestContract_OpenFGA(t *testing.T) {
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

	// seed.sh writes only the demo workspace/team policy tuples. The
	// document→workspace tuple and contract-only owner/admin tuples are normally
	// outside that bootstrap path; write them here so the test is self-contained.
	// WriteTuples is idempotent, so re-running the test against the same store is safe.
	tuples := []rebac.TupleKey{
		rebac.Tuple(fixtures.RoadmapDocument, rebac.RelationDocumentWorkspace, rebac.Subject(fixtures.ProductWorkspace)),
	}
	tuples = append(tuples, contract.ExtraTuples()...)
	err = svc.WriteTuples(context.Background(), tuples)
	if err != nil {
		t.Fatalf("seed contract tuples: %v", err)
	}

	contract.Run(t, svc.Check)
}
