package authz_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/fixtures"
)

func TestAuditAuthorizer_DelegatesResultToInner(t *testing.T) {
	// Arrange: wrap the real GraphAuthorizer so we can verify the audit layer
	// does not change the result.
	store := seedStore()
	inner := authz.NewGraphAuthorizer(store)
	var buf bytes.Buffer
	audit := authz.NewAuditAuthorizer(inner, &buf)
	req := authz.CheckRequest{
		User:     fixtures.WorkspaceEditor,
		Relation: authz.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	}

	// Act
	result, err := audit.Check(context.Background(), req)

	// Assert: the result must match what the inner authorizer would return directly.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed=true but got false")
	}
}

func TestAuditAuthorizer_WritesLogLine(t *testing.T) {
	// Arrange
	store := seedStore()
	inner := authz.NewGraphAuthorizer(store)
	var buf bytes.Buffer
	audit := authz.NewAuditAuthorizer(inner, &buf)
	req := authz.CheckRequest{
		User:     fixtures.WorkspaceViewer,
		Relation: authz.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	}

	// Act
	_, err := audit.Check(context.Background(), req)

	// Assert: the log must mention the relation and the denied outcome.
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

func TestAuditAuthorizer_SatisfiesAuthorizerInterface(t *testing.T) {
	// Arrange: this test documents the compile-time assertion in middleware.go.
	// If AuditAuthorizer ever stops satisfying Authorizer, the build breaks there.
	// Here we verify the runtime behaviour matches the contract.
	store := seedStore()
	inner := authz.NewGraphAuthorizer(store)
	var buf bytes.Buffer

	// Act: assign to Authorizer — if the interface is not satisfied, this line fails.
	var a authz.Authorizer = authz.NewAuditAuthorizer(inner, &buf)

	// Assert
	result, err := a.Check(context.Background(), authz.CheckRequest{
		User:     fixtures.WorkspaceEditor,
		Relation: authz.RelationDocumentCanRead,
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
	// Arrange: seed a store and wrap it read-only.
	store := seedStore()
	ro := authz.NewReadOnlyStore(store)

	// Act: use the promoted Has method (from the embedded TupleReader).
	found := ro.Has(
		fixtures.PlatformTeam,
		authz.RelationTeamMember,
		authz.Subject(fixtures.WorkspaceEditor),
	)

	// Assert
	if !found {
		t.Error("expected ReadOnlyStore to find the member tuple")
	}
}

func TestReadOnlyStore_CanDriveGraphAuthorizer(t *testing.T) {
	// Arrange: GraphAuthorizer accepts a TupleReader. ReadOnlyStore embeds
	// TupleReader, so it can be passed directly — no conversion needed.
	store := seedStore()
	ro := authz.NewReadOnlyStore(store)

	// Act
	auth := authz.NewGraphAuthorizer(ro)
	result, err := auth.Check(context.Background(), authz.CheckRequest{
		User:     fixtures.WorkspaceEditor,
		Relation: authz.RelationDocumentCanEdit,
		Object:   fixtures.RoadmapDocument,
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected GraphAuthorizer driven by ReadOnlyStore to allow editor")
	}
}
