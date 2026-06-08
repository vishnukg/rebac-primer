package openfga_test

import (
	"os"
	"testing"

	"rebac-primer/internal/authz/contract"
	"rebac-primer/internal/openfga"
)

// TestContract_OpenFGA holds the OpenFGA backend to the *same* canonical contract
// as the from-scratch evaluator. When both pass, the two backends provably agree
// on the model — that is the parity guarantee.
//
// It skips unless a store is configured, so `go test ./...` stays green offline.
// To run it:
//
//	make openfga-up && make openfga-seed   # start OpenFGA, write model + policy tuples
//	make go-server-openfga                 # start once so the document tuple is written
//	OPENFGA_API_URL=http://127.0.0.1:8080 \
//	OPENFGA_STORE_ID=<id> OPENFGA_MODEL_ID=<id> \
//	  go test -run TestContract_OpenFGA ./internal/openfga
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

	contract.Run(t, svc.Check)
}
