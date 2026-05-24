package graph_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"rebac-primer/internal/authzservice/adapters/graph"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/shared"
)

func TestAuditEvaluator_DelegatesResultToInner(t *testing.T) {
	ev := newEvaluator()
	var buf bytes.Buffer
	audit := graph.NewAuditEvaluator(ev, &buf)
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
	audit := graph.NewAuditEvaluator(ev, &buf)
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

	// Assign to Checker (= ports.Evaluator) — if the interface is not satisfied, this fails.
	var c graph.Checker = graph.NewAuditEvaluator(ev, &buf)

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
	ro := graph.NewReadOnlyStore(store)

	found := ro.Has(
		fixtures.PlatformTeam,
		shared.RelationTeamMember,
		shared.Subject(fixtures.Alice),
	)
	if !found {
		t.Error("expected ReadOnlyStore to find the member tuple")
	}
}

func TestReadOnlyStore_CanDriveGraphEvaluator(t *testing.T) {
	store := seedStore()
	ro := graph.NewReadOnlyStore(store)

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
