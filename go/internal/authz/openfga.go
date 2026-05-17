package authz

import (
	"context"
	"fmt"

	openfga "github.com/openfga/go-sdk"
	fgaclient "github.com/openfga/go-sdk/client"
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

type openFGAClient interface {
	Check(ctx context.Context, req fgaclient.ClientCheckRequest) (*fgaclient.ClientCheckResponse, error)
	WriteTuples(ctx context.Context, tuples []fgaclient.ClientTupleKey) error
}

type sdkOpenFGAClient struct {
	client *fgaclient.OpenFgaClient
}

func (s *sdkOpenFGAClient) Check(ctx context.Context, req fgaclient.ClientCheckRequest) (*fgaclient.ClientCheckResponse, error) {
	return s.client.Check(ctx).Body(req).Execute()
}

func (s *sdkOpenFGAClient) WriteTuples(ctx context.Context, tuples []fgaclient.ClientTupleKey) error {
	_, err := s.client.WriteTuples(ctx).Body(tuples).Execute()
	return err
}

// OpenFGAAuthorizer adapts the official OpenFGA Go SDK to this repo's
// Authorizer interface. Domain code depends on Authorizer, not SDK request types.
type OpenFGAAuthorizer struct {
	client openFGAClient
}

// NewOpenFGAAuthorizer creates an OpenFGAAuthorizer backed by the official SDK.
func NewOpenFGAAuthorizer(cfg OpenFGAConfig) (*OpenFGAAuthorizer, error) {
	sdk, err := fgaclient.NewSdkClient(&fgaclient.ClientConfiguration{
		ApiUrl:               cfg.APIURL,
		StoreId:              cfg.StoreID,
		AuthorizationModelId: cfg.AuthorizationModelID,
	})
	if err != nil {
		return nil, fmt.Errorf("openfga: create sdk client: %w", err)
	}

	return newOpenFGAAuthorizerWithClient(&sdkOpenFGAClient{client: sdk}), nil
}

func newOpenFGAAuthorizerWithClient(client openFGAClient) *OpenFGAAuthorizer {
	return &OpenFGAAuthorizer{client: client}
}

// Check satisfies the Authorizer interface by asking OpenFGA to evaluate the
// relationship graph remotely.
func (o *OpenFGAAuthorizer) Check(ctx context.Context, req CheckRequest) (CheckResult, error) {
	resp, err := o.client.Check(ctx, fgaclient.ClientCheckRequest{
		User:     string(req.User),
		Relation: string(req.Relation),
		Object:   string(req.Object),
	})
	if err != nil {
		return CheckResult{}, fmt.Errorf("openfga: check %s has %s on %s: %w", req.User, req.Relation, req.Object, err)
	}
	if resp == nil {
		return CheckResult{}, fmt.Errorf("openfga: check %s has %s on %s: empty response", req.User, req.Relation, req.Object)
	}

	return CheckResult{
		Allowed: resp.GetAllowed(),
		Trace:   []string{"OpenFGA evaluated the relationship graph remotely"},
	}, nil
}

// WriteTuples writes relationship facts to OpenFGA without exposing SDK tuple
// types to callers.
func (o *OpenFGAAuthorizer) WriteTuples(ctx context.Context, tuples []TupleKey) error {
	sdkTuples := make([]fgaclient.ClientTupleKey, 0, len(tuples))
	for _, tuple := range tuples {
		sdkTuples = append(sdkTuples, *openfga.NewTupleKey(
			string(tuple.User),
			string(tuple.Relation),
			string(tuple.Object),
		))
	}

	if err := o.client.WriteTuples(ctx, sdkTuples); err != nil {
		return fmt.Errorf("openfga: write tuples: %w", err)
	}
	return nil
}

// Ensure OpenFGAAuthorizer satisfies the Authorizer interface at compile time.
var _ Authorizer = (*OpenFGAAuthorizer)(nil)
