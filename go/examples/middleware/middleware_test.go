package middleware_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"rebac-primer/examples/middleware"
	authzdb "rebac-primer/internal/authz/adapters/db"
	"rebac-primer/internal/authz/adapters/graph"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/shared"
)

// seedStore builds a tuple store from the standard fixture tuples.
func seedStore(extra ...shared.TupleKey) *authzdb.InMemoryTupleStore {
	all := append(fixtures.SeedRelationshipTuples(), extra...)
	return authzdb.New(all...)
}

// newEvaluator wraps seedStore + the real graph evaluator.
func newEvaluator(extra ...shared.TupleKey) *graph.GraphEvaluator {
	return graph.NewGraphEvaluator(seedStore(extra...))
}

func TestAuditEvaluator_DelegatesResultToInner(t *testing.T) {
	ev := newEvaluator()
	var buf bytes.Buffer
	audit := middleware.NewAuditEvaluator(ev, &buf)
	req := shared.CheckRequest{
		User:     fixtures.Alice,
		Relation: shared.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	}

	result, err := audit.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed=true but got false")
	}
}

func TestAuditEvaluator_WritesLogLine(t *testing.T) {
	ev := newEvaluator()
	var buf bytes.Buffer
	audit := middleware.NewAuditEvaluator(ev, &buf)
	req := shared.CheckRequest{
		User:     fixtures.Bob,
		Relation: shared.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	}

	_, err := audit.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	line := buf.String()
	if !strings.Contains(line, "can_edit") {
		t.Errorf("expected log to mention relation can_edit, got: %s", line)
	}
	if !strings.Contains(line, "denied") {
		t.Errorf("expected log to mention denied, got: %s", line)
	}
}

func TestAuditEvaluator_SatisfiesCheckerInterface(t *testing.T) {
	ev := newEvaluator()
	var buf bytes.Buffer

	// Assign to Checker (= authz.Evaluator) — if the interface is not satisfied, this fails.
	var c middleware.Checker = middleware.NewAuditEvaluator(ev, &buf)

	result, err := c.Evaluate(context.Background(), shared.CheckRequest{
		User:     fixtures.Alice,
		Relation: shared.RelationDocumentCanRead,
		Object:   fixtures.RoadmapDocument,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed=true")
	}
}

func TestReadOnlyStore_ExposesReadMethods(t *testing.T) {
	store := seedStore()
	ro := middleware.NewReadOnlyStore(store)

	found, err := ro.Has(
		context.Background(),
		fixtures.PlatformTeam,
		shared.RelationTeamMember,
		shared.Subject(fixtures.Alice),
	)
	if err != nil {
		t.Fatalf("Has returned unexpected error: %v", err)
	}
	if !found {
		t.Error("expected ReadOnlyStore to find the member tuple")
	}
}

func TestReadOnlyStore_CanDriveGraphEvaluator(t *testing.T) {
	store := seedStore()
	ro := middleware.NewReadOnlyStore(store)

	ev := graph.NewGraphEvaluator(ro)
	result, err := ev.Evaluate(context.Background(), shared.CheckRequest{
		User:     fixtures.Alice,
		Relation: shared.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected GraphEvaluator driven by ReadOnlyStore to allow editor")
	}
}
