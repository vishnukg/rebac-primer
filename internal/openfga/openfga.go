// Package openfga adapts a real OpenFGA server to the authorization operations
// consumed by the application. It can replace the in-process service with a
// wiring change in cmd/server/main.go (selected by AUTHZ_BACKEND=openfga).
//
// Why this implements the whole application-facing capability (not Evaluator):
// The graph build swaps the Evaluator port (Evaluate has ctx + error — a good
// network seam) and keeps the in-memory TupleRepository for writes. But
// TupleRepository.Write is sync and has no ctx/error, which does not fit a
// network backend. The public operations here all carry ctx + error, so checks
// and tuple writes can both go to OpenFGA and remain consistent.
//
// The model and the workspace/team policy tuples are seeded into the store out
// of band (deployments/openfga/seed.sh). Document-level tuples are still written
// at runtime by the documents service via WriteTuples — they just land in
// OpenFGA instead of the in-memory store.
package openfga

import (
	"context"
	"fmt"
	"slices"

	openfga "github.com/openfga/go-sdk/client"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/rebac"
)

// Config points the adapter at a store + pinned model on an OpenFGA server.
type Config struct {
	APIURL  string // e.g. http://127.0.0.1:8080
	StoreID string
	ModelID string
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
		User:     string(req.User),
		Relation: string(req.Relation),
		Object:   string(req.Object),
	}).Execute()
	if err != nil {
		return rebac.CheckResult{}, fmt.Errorf("openfga: check: %w", err)
	}
	allowed := resp.GetAllowed()
	return rebac.CheckResult{
		Allowed: allowed,
		Trace:   []string{fmt.Sprintf("OpenFGA: %s %s %s -> %t", req.User, req.Relation, req.Object, allowed)},
	}, nil
}

// WriteTuples persists relationship facts to the OpenFGA store.
//
// Tuples that already exist are skipped before the write. The OpenFGA Write API
// rejects a duplicate tuple as an error, but the application contract (and the
// in-memory store) treats writes as idempotent — re-running the startup seed
// against a populated store must be a no-op, not a crash. The read-then-write is
// not atomic, so a concurrent writer can still race in a duplicate; that
// best-effort idempotency is fine for this primer.
func (s *Service) WriteTuples(ctx context.Context, tuples []rebac.TupleKey) error {
	writes := make([]openfga.ClientTupleKey, 0, len(tuples))
	for _, t := range tuples {
		if err := authz.ValidateTuple(t); err != nil {
			return err
		}
		exists, err := s.tupleExists(ctx, t)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		writes = append(writes, openfga.ClientTupleKey{
			User:     string(t.User),
			Relation: string(t.Relation),
			Object:   string(t.Object),
		})
	}
	if len(writes) == 0 {
		return nil
	}
	if _, err := s.client.Write(ctx).Body(openfga.ClientWriteRequest{Writes: writes}).Execute(); err != nil {
		return fmt.Errorf("openfga: write tuples: %w", err)
	}
	return nil
}

// tupleExists reports whether the exact tuple is already stored.
func (s *Service) tupleExists(ctx context.Context, t rebac.TupleKey) (bool, error) {
	stored, err := s.ListTuples(ctx, authz.TupleFilter{Object: t.Object, Relation: t.Relation})
	if err != nil {
		return false, err
	}
	return slices.Contains(stored, t), nil
}

// DeleteTuples removes relationship facts from the OpenFGA store.
func (s *Service) DeleteTuples(ctx context.Context, tuples []rebac.TupleKey) error {
	if len(tuples) == 0 {
		return nil
	}
	deletes := make([]openfga.ClientTupleKeyWithoutCondition, 0, len(tuples))
	for _, t := range tuples {
		deletes = append(deletes, openfga.ClientTupleKeyWithoutCondition{
			User:     string(t.User),
			Relation: string(t.Relation),
			Object:   string(t.Object),
		})
	}
	if _, err := s.client.Write(ctx).Body(openfga.ClientWriteRequest{Deletes: deletes}).Execute(); err != nil {
		return fmt.Errorf("openfga: delete tuples: %w", err)
	}
	return nil
}

// ListTuples reads tuples back from the OpenFGA store, optionally filtered by
// object and/or relation.
func (s *Service) ListTuples(ctx context.Context, filter ...authz.TupleFilter) ([]rebac.TupleKey, error) {
	body := openfga.ClientReadRequest{}
	if len(filter) > 0 {
		if filter[0].Object != "" {
			object := string(filter[0].Object)
			body.Object = &object
		}
		if filter[0].Relation != "" {
			relation := string(filter[0].Relation)
			body.Relation = &relation
		}
	}
	var out []rebac.TupleKey
	var continuationToken *string
	for {
		resp, err := s.client.Read(ctx).
			Body(body).
			Options(openfga.ClientReadOptions{ContinuationToken: continuationToken}).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("openfga: read tuples: %w", err)
		}
		for _, t := range resp.GetTuples() {
			key := t.GetKey()
			out = append(out, rebac.TupleKey{
				Object:   rebac.Object(key.GetObject()),
				Relation: rebac.Relation(key.GetRelation()),
				User:     rebac.Subject(key.GetUser()),
			})
		}

		token := resp.GetContinuationToken()
		if token == "" {
			return out, nil
		}
		continuationToken = &token
	}
}
