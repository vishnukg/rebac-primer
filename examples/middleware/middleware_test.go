package middleware_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"rebac-primer/examples/middleware"
	"rebac-primer/internal/authz"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/rebac"
)

// seedStore builds a tuple store from the standard fixture tuples.
func seedStore(extra ...rebac.TupleKey) *authz.InMemoryStore {
	all := append(fixtures.SeedRelationshipTuples(), extra...)
	return authz.NewInMemoryStore(all...)
}

// newEvaluator wraps seedStore + the real graph evaluator.
func newEvaluator(extra ...rebac.TupleKey) *authz.GraphEvaluator {
	return authz.NewGraphEvaluator(seedStore(extra...))
}

func TestAuditEvaluator_DelegatesResultToInner(t *testing.T) {
	ev := newEvaluator()
	var buf bytes.Buffer
	audit := middleware.NewAuditEvaluator(ev, &buf)
	req := rebac.CheckRequest{
		User:     fixtures.Alice,
		Relation: rebac.RelationDocumentCanEdit,
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
	req := rebac.CheckRequest{
		User:     fixtures.Bob,
		Relation: rebac.RelationDocumentCanEdit,
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

	// Assign to Checker — if the interface is not satisfied, this fails.
	var c middleware.Checker = middleware.NewAuditEvaluator(ev, &buf)

	result, err := c.Evaluate(context.Background(), rebac.CheckRequest{
		User:     fixtures.Alice,
		Relation: rebac.RelationDocumentCanRead,
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
		rebac.RelationTeamMember,
		rebac.Subject(fixtures.Alice),
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

	ev := authz.NewGraphEvaluator(ro)
	result, err := ev.Evaluate(context.Background(), rebac.CheckRequest{
		User:     fixtures.Alice,
		Relation: rebac.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected GraphEvaluator driven by ReadOnlyStore to allow editor")
	}
}
