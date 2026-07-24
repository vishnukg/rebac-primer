// Package openfga adapts a real OpenFGA server to the authorization operations
// consumed by the application. It can replace the in-process service with a
// wiring change in cmd/server/main.go (selected by AUTHZ_BACKEND=openfga).
//
// Why this implements the whole application-facing capability (not Evaluator):
// The graph build swaps the Evaluator port, but writes still go through the
// in-process relationship store. The public operations here all carry ctx + error, so
// checks and relationship writes can both go to OpenFGA and remain consistent.
//
// The model and the workspace/team relationships are seeded into the store out
// of band (deployments/openfga/seed.sh). Document relationships are still written
// at runtime by the documents service via WriteRelationships — they just land in
// OpenFGA instead of the in-memory store.
package openfga

import (
	"context"
	"fmt"
	"net/http"

	openfga "github.com/openfga/go-sdk/client"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/rebac"
)

// Config points the adapter at a store + pinned model on an OpenFGA server.
type Config struct {
	APIURL     string // e.g. http://127.0.0.1:8080
	StoreID    string
	ModelID    string
	HTTPClient *http.Client // optional; defaults to the SDK's HTTP client
}

// Service delegates authorization operations to an OpenFGA server. Consumers
// accept it through interfaces declared at their point of use.
type Service struct {
	client *openfga.OpenFgaClient
}

// New builds an OpenFGA-backed authz service.
func New(cfg Config) (*Service, error) {
	if cfg.APIURL == "" {
		return nil, fmt.Errorf("openfga: APIURL is required")
	}
	if cfg.StoreID == "" {
		return nil, fmt.Errorf("openfga: StoreID is required")
	}
	if cfg.ModelID == "" {
		return nil, fmt.Errorf("openfga: ModelID is required")
	}

	client, err := openfga.NewSdkClient(&openfga.ClientConfiguration{
		ApiUrl:               cfg.APIURL,
		StoreId:              cfg.StoreID,
		AuthorizationModelId: cfg.ModelID,
		HTTPClient:           cfg.HTTPClient,
	})
	if err != nil {
		return nil, fmt.Errorf("openfga: new client: %w", err)
	}
	return &Service{client: client}, nil
}

// Check delegates to the OpenFGA Check API. OpenFGA does the graph traversal
// our evaluator.go does in process; it returns only allow/deny, so the trace is
// a single synthetic line rather than the step-by-step trace the graph produces.
func (s *Service) Check(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error) {
	if err := authz.ValidateCheckRequest(req); err != nil {
		return rebac.CheckResult{}, err
	}
	resp, err := s.client.Check(ctx).Body(openfga.ClientCheckRequest{
		// OpenFGA's transport vocabulary is user/relation/object. In the
		// application domain these values are subject/permission/resource.
		User:     string(req.Subject),
		Relation: string(req.Permission),
		Object:   string(req.Resource),
	}).Execute()
	if err != nil {
		return rebac.CheckResult{}, fmt.Errorf("openfga: check: %w", err)
	}
	allowed := resp.GetAllowed()
	return rebac.CheckResult{
		Allowed: allowed,
		Trace: []string{fmt.Sprintf(
			"OpenFGA: subject=%s permission=%s resource=%s -> %t",
			req.Subject, req.Permission, req.Resource, allowed,
		)},
	}, nil
}

// WriteRelationships persists relationship facts to the OpenFGA tuple store.
//
// The conflict option makes duplicate writes atomic no-ops. That matches the
// application contract and the in-memory store without a racy read-before-write
// round trip.
func (s *Service) WriteRelationships(ctx context.Context, relationships []rebac.Relationship) error {
	writes := make([]openfga.ClientTupleKey, 0, len(relationships))
	for _, relationship := range relationships {
		if err := authz.ValidateRelationship(relationship); err != nil {
			return err
		}
		writes = append(writes, openfga.ClientTupleKey{
			User:     string(relationship.Subject),
			Relation: string(relationship.Relation),
			Object:   string(relationship.Resource),
		})
	}
	if len(writes) == 0 {
		return nil
	}
	options := openfga.ClientWriteOptions{
		Conflict: openfga.ClientWriteConflictOptions{
			OnDuplicateWrites: openfga.CLIENT_WRITE_REQUEST_ON_DUPLICATE_WRITES_IGNORE,
		},
	}
	if _, err := s.client.Write(ctx).
		Body(openfga.ClientWriteRequest{Writes: writes}).
		Options(options).
		Execute(); err != nil {
		return fmt.Errorf("openfga: write tuples: %w", err)
	}
	return nil
}

// DeleteRelationships removes relationship facts from the OpenFGA tuple store.
func (s *Service) DeleteRelationships(ctx context.Context, relationships []rebac.Relationship) error {
	if len(relationships) == 0 {
		return nil
	}
	deletes := make([]openfga.ClientTupleKeyWithoutCondition, 0, len(relationships))
	for _, relationship := range relationships {
		if err := authz.ValidateRelationship(relationship); err != nil {
			return err
		}
		deletes = append(deletes, openfga.ClientTupleKeyWithoutCondition{
			User:     string(relationship.Subject),
			Relation: string(relationship.Relation),
			Object:   string(relationship.Resource),
		})
	}
	options := openfga.ClientWriteOptions{
		Conflict: openfga.ClientWriteConflictOptions{
			OnMissingDeletes: openfga.CLIENT_WRITE_REQUEST_ON_MISSING_DELETES_IGNORE,
		},
	}
	if _, err := s.client.Write(ctx).
		Body(openfga.ClientWriteRequest{Deletes: deletes}).
		Options(options).
		Execute(); err != nil {
		return fmt.Errorf("openfga: delete tuples: %w", err)
	}
	return nil
}

// ListRelationships reads relationships back from OpenFGA, optionally filtered
// by resource and/or relation.
//
// A relation-only filter is rejected up front: the OpenFGA Read API requires at
// least an object type alongside a relation, so forwarding that filter would
// fail server-side with a less helpful error. The in-memory store supports it;
// this is one of the small capability differences between the two backends.
func (s *Service) ListRelationships(
	ctx context.Context,
	filter ...authz.RelationshipFilter,
) ([]rebac.Relationship, error) {
	body := openfga.ClientReadRequest{}
	if len(filter) > 0 {
		if filter[0].Resource == "" && filter[0].Relation != "" {
			return nil, fmt.Errorf("openfga: list relationships: the OpenFGA Read API cannot filter by relation alone; set Resource too or drop the Relation filter")
		}
		if filter[0].Resource != "" {
			object := string(filter[0].Resource)
			body.Object = &object
		}
		if filter[0].Relation != "" {
			relation := string(filter[0].Relation)
			body.Relation = &relation
		}
	}
	var out []rebac.Relationship
	var continuationToken *string
	for {
		resp, err := s.client.Read(ctx).
			Body(body).
			Options(openfga.ClientReadOptions{ContinuationToken: continuationToken}).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("openfga: read relationships: %w", err)
		}
		for _, t := range resp.GetTuples() {
			key := t.GetKey()
			out = append(out, rebac.Relationship{
				Subject:  rebac.Subject(key.GetUser()),
				Relation: rebac.Relation(key.GetRelation()),
				Resource: rebac.Resource(key.GetObject()),
			})
		}

		token := resp.GetContinuationToken()
		if token == "" {
			return out, nil
		}
		continuationToken = &token
	}
}
