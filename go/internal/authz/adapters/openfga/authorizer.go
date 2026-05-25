// Package openfga provides an OpenFGA SDK adapter for the authz service.
//
// Authorizer adapts the official OpenFGA Go SDK to this project's
// [authz.Evaluator] interface.  Domain code depends on the interface, not on
// SDK types — swapping between the in-process graph evaluator and OpenFGA
// requires changing only the wiring in app.go.
//
// Mirrors typescript/src/adapters/authz/makeOpenFgaAuthorizer.ts.
package openfga

import (
	"context"
	"fmt"

	openfga "github.com/openfga/go-sdk"
	fgaclient "github.com/openfga/go-sdk/client"

	"rebac-primer/internal/shared"
)

// Config holds the connection details for an OpenFGA server.
type Config struct {
	// APIURL is the base URL of the OpenFGA server, e.g. "http://localhost:8080".
	APIURL string
	// StoreID is the OpenFGA store ID to use.
	StoreID string
	// AuthorizationModelID is the model ID returned after uploading the model.
	AuthorizationModelID string
}

// sdkClient is an internal interface that narrows the full SDK to the two
// methods we actually use.  Kept unexported so tests can swap it out.
type sdkClient interface {
	Check(ctx context.Context, req fgaclient.ClientCheckRequest) (*fgaclient.ClientCheckResponse, error)
	WriteTuples(ctx context.Context, tuples []fgaclient.ClientTupleKey) error
}

type realSDKClient struct{ client *fgaclient.OpenFgaClient }

func (r *realSDKClient) Check(ctx context.Context, req fgaclient.ClientCheckRequest) (*fgaclient.ClientCheckResponse, error) {
	return r.client.Check(ctx).Body(req).Execute()
}

func (r *realSDKClient) WriteTuples(ctx context.Context, tuples []fgaclient.ClientTupleKey) error {
	_, err := r.client.WriteTuples(ctx).Body(tuples).Execute()
	return err
}

// Authorizer adapts the official OpenFGA Go SDK to this repo's [authz.Evaluator]
// interface.  It also writes tuples, making it a drop-in replacement for the
// in-process graph evaluator + in-memory store when targeting a real OpenFGA server.
type Authorizer struct {
	client sdkClient
}

// New creates an Authorizer backed by the official OpenFGA SDK.
func New(cfg Config) (*Authorizer, error) {
	sdk, err := fgaclient.NewSdkClient(&fgaclient.ClientConfiguration{
		ApiUrl:               cfg.APIURL,
		StoreId:              cfg.StoreID,
		AuthorizationModelId: cfg.AuthorizationModelID,
	})
	if err != nil {
		return nil, fmt.Errorf("openfga: create sdk client: %w", err)
	}
	return &Authorizer{client: &realSDKClient{client: sdk}}, nil
}

// Evaluate satisfies [authz.Evaluator] by asking OpenFGA to evaluate the
// relationship graph remotely.
func (a *Authorizer) Evaluate(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error) {
	resp, err := a.client.Check(ctx, fgaclient.ClientCheckRequest{
		User:     string(req.User),
		Relation: string(req.Relation),
		Object:   string(req.Object),
	})
	if err != nil {
		return shared.CheckResult{}, fmt.Errorf("openfga: check %s has %s on %s: %w",
			req.User, req.Relation, req.Object, err)
	}
	if resp == nil {
		return shared.CheckResult{}, fmt.Errorf("openfga: check %s has %s on %s: empty response",
			req.User, req.Relation, req.Object)
	}

	return shared.CheckResult{
		Allowed: resp.GetAllowed(),
		Trace:   []string{"OpenFGA evaluated the relationship graph remotely"},
	}, nil
}

// WriteTuples writes relationship facts to OpenFGA without exposing SDK types
// to callers.
func (a *Authorizer) WriteTuples(ctx context.Context, tuples []shared.TupleKey) error {
	sdkTuples := make([]fgaclient.ClientTupleKey, 0, len(tuples))
	for _, t := range tuples {
		sdkTuples = append(sdkTuples, *openfga.NewTupleKey(
			string(t.User),
			string(t.Relation),
			string(t.Object),
		))
	}
	if err := a.client.WriteTuples(ctx, sdkTuples); err != nil {
		return fmt.Errorf("openfga: write tuples: %w", err)
	}
	return nil
}
