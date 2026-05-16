// Package authz — OpenFGA adapter stub.
//
// This file is a placeholder for an OpenFGAAuthorizer that wraps the official
// Go SDK. It satisfies the Authorizer interface and can be swapped in for the
// GraphAuthorizer without changing any domain code.
//
// To activate this adapter:
//
//  1. Add the SDK dependency:
//     go get github.com/openfga/go-sdk@v0.6.3
//
//  2. Replace this stub with a real implementation that:
//     - Creates a store and uploads OpenFGAModel on first run.
//     - Writes tuples from the fixture set.
//     - Calls client.Check() for each Authorizer.Check() call.
//
// See docs/13-typescript-openfga-implementation.md for the TS counterpart.

package authz

import (
	"context"
	"fmt"
)

// OpenFGAConfig holds the connection details for an OpenFGA server.
type OpenFGAConfig struct {
	// APIURL is the base URL of the OpenFGA server, e.g. "http://localhost:8080".
	APIURL string
	// StoreID is the OpenFGA store ID to use.
	StoreID string
	// AuthorizationModelID is the model ID returned after uploading OpenFGAModel.
	AuthorizationModelID string
}

// OpenFGAAuthorizer is a stub that will wrap github.com/openfga/go-sdk.
// It currently returns an error so the compiler confirms interface satisfaction
// without requiring the SDK to be present.
type OpenFGAAuthorizer struct {
	cfg OpenFGAConfig
}

// NewOpenFGAAuthorizer creates an OpenFGAAuthorizer. Run `go get github.com/openfga/go-sdk`
// and implement the body of Check before using this in production.
func NewOpenFGAAuthorizer(cfg OpenFGAConfig) *OpenFGAAuthorizer {
	return &OpenFGAAuthorizer{cfg: cfg}
}

// Check satisfies the Authorizer interface.
// Replace this stub with a real call to the OpenFGA SDK.
func (o *OpenFGAAuthorizer) Check(_ context.Context, req CheckRequest) (CheckResult, error) {
	return CheckResult{}, fmt.Errorf(
		"OpenFGAAuthorizer is a stub: add github.com/openfga/go-sdk and implement Check "+
			"(tried to check %s has %s on %s against store %s)",
		req.User, req.Relation, req.Object, o.cfg.StoreID,
	)
}

// Ensure OpenFGAAuthorizer satisfies the Authorizer interface at compile time.
var _ Authorizer = (*OpenFGAAuthorizer)(nil)
