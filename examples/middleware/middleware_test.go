package middleware_test

import (
	"bytes"
	"strings"
	"testing"

	"rebac-primer/examples/middleware"
	"rebac-primer/internal/authz"
	"rebac-primer/internal/fixtures"
	"rebac-primer/internal/rebac"
)

func TestAuditEvaluator_DelegatesResultToInner(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	ev := authz.NewGraphEvaluator(store)
	var buf bytes.Buffer
	audit := middleware.NewAuditEvaluator(ev, &buf)
	req := rebac.CheckRequest{
		User:     fixtures.Alice,
		Relation: rebac.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	}

	// Act
	result, err := audit.Evaluate(t.Context(), req)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed=true but got false")
	}
}

func TestAuditEvaluator_WritesLogLine(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	ev := authz.NewGraphEvaluator(store)
	var buf bytes.Buffer
	audit := middleware.NewAuditEvaluator(ev, &buf)
	req := rebac.CheckRequest{
		User:     fixtures.Bob,
		Relation: rebac.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	}

	// Act
	_, err := audit.Evaluate(t.Context(), req)

	// Assert
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
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	ev := authz.NewGraphEvaluator(store)
	var buf bytes.Buffer

	// Assign to Checker — if the interface is not satisfied, this fails.
	var c middleware.Checker = middleware.NewAuditEvaluator(ev, &buf)

	// Act
	result, err := c.Evaluate(t.Context(), rebac.CheckRequest{
		User:     fixtures.Alice,
		Relation: rebac.RelationDocumentCanRead,
		Object:   fixtures.RoadmapDocument,
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed=true")
	}
}

func TestReadOnlyStore_ExposesReadMethods(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	ro := middleware.NewReadOnlyStore(store)

	// Act
	found, err := ro.Has(
		t.Context(),
		fixtures.PlatformTeam,
		rebac.RelationTeamMember,
		rebac.Subject(fixtures.Alice),
	)

	// Assert
	if err != nil {
		t.Fatalf("Has returned unexpected error: %v", err)
	}
	if !found {
		t.Error("expected ReadOnlyStore to find the member tuple")
	}
}

func TestReadOnlyStore_CanDriveGraphEvaluator(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
	ro := middleware.NewReadOnlyStore(store)

	ev := authz.NewGraphEvaluator(ro)

	// Act
	result, err := ev.Evaluate(t.Context(), rebac.CheckRequest{
		User:     fixtures.Alice,
		Relation: rebac.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected GraphEvaluator driven by ReadOnlyStore to allow editor")
	}
}
