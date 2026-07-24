package authz_test

import (
	"context"
	"errors"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/rebac"
)

// This file unit-tests the *authz.Service returned by [authz.New] in
// isolation from any real adapter. The Service is a thin orchestrator over relationship
// storage ports and an Evaluator, which makes it the right place
// to demonstrate the difference between stubs and mocks:
//
//   - A STUB stands in for a collaborator and returns canned answers. It is used
//     for STATE verification: "given the evaluator says allowed, does Check
//     return allowed?" The test never inspects how the stub was called.
//
//   - A MOCK also stands in for a collaborator but, in addition, records the
//     calls it received. It is used for BEHAVIOUR verification: "does WriteRelationships
//     call repository.Write once per relationship, with the exact relationships, in order?"
//     The assertions are about the interaction, not a returned value.
//
// Both kinds implement the same port interface; the difference is what the test
// asserts on, not the type.

// ── Stubs (state verification) ──────────────────────────────────────────────

// stubEvaluator is a STUB: it returns a fixed CheckResult/error and records
// nothing.
type stubEvaluator struct {
	result rebac.CheckResult
	err    error
}

func (s stubEvaluator) Evaluate(context.Context, rebac.CheckRequest) (rebac.CheckResult, error) {
	return s.result, s.err
}

// stubRepository is a STUB RelationshipRepository whose reads return canned data and
// whose writes are no-ops. Tests that exercise the evaluator path pass this so
// the Service has a collaborator without caring how it is used.
type stubRepository struct {
	all []rebac.Relationship
}

func (s stubRepository) Has(context.Context, rebac.Subject, rebac.Relation, rebac.Resource) (bool, error) {
	return false, nil
}
func (s stubRepository) FindByResourceRelation(context.Context, rebac.Resource, rebac.Relation) ([]rebac.Relationship, error) {
	return nil, nil
}
func (s stubRepository) FindAll(context.Context, ...authz.RelationshipFilter) ([]rebac.Relationship, error) {
	return s.all, nil
}
func (s stubRepository) Write(context.Context, rebac.Relationship) error  { return nil }
func (s stubRepository) Delete(context.Context, rebac.Relationship) error { return nil }

// ── Mocks (behaviour verification) ──────────────────────────────────────────

// mockEvaluator is a MOCK: it records every request it is asked to evaluate so a
// test can assert the Service delegated the exact CheckRequest unchanged.
type mockEvaluator struct {
	calls  []rebac.CheckRequest
	result rebac.CheckResult
}

func (m *mockEvaluator) Evaluate(_ context.Context, req rebac.CheckRequest) (rebac.CheckResult, error) {
	m.calls = append(m.calls, req)
	return m.result, nil
}

// mockRepository is a MOCK RelationshipRepository: it records the Write/Delete calls and
// the filters passed to FindAll, so tests can verify the Service's interactions
// with persistence.
type mockRepository struct {
	writes      []rebac.Relationship
	deletes     []rebac.Relationship
	findFilters [][]authz.RelationshipFilter
	findResult  []rebac.Relationship
}

func (m *mockRepository) Has(context.Context, rebac.Subject, rebac.Relation, rebac.Resource) (bool, error) {
	return false, nil
}
func (m *mockRepository) FindByResourceRelation(context.Context, rebac.Resource, rebac.Relation) ([]rebac.Relationship, error) {
	return nil, nil
}
func (m *mockRepository) FindAll(_ context.Context, filter ...authz.RelationshipFilter) ([]rebac.Relationship, error) {
	m.findFilters = append(m.findFilters, filter)
	return m.findResult, nil
}
func (m *mockRepository) Write(_ context.Context, t rebac.Relationship) error {
	m.writes = append(m.writes, t)
	return nil
}
func (m *mockRepository) Delete(_ context.Context, t rebac.Relationship) error {
	m.deletes = append(m.deletes, t)
	return nil
}

// Compile-time checks that the doubles satisfy the ports they stand in for.
var (
	_ authz.Evaluator              = stubEvaluator{}
	_ authz.Evaluator              = (*mockEvaluator)(nil)
	_ authz.RelationshipRepository = stubRepository{}
	_ authz.RelationshipRepository = (*mockRepository)(nil)
)

// ── Check ───────────────────────────────────────────────────────────────────

func TestService_GivenEvaluatorAllows_WhenCheck_ThenReturnsEvaluatorResult(t *testing.T) {
	// Arrange: a STUB evaluator pinned to an allowed result.
	evaluator := stubEvaluator{result: rebac.CheckResult{Allowed: true, Trace: []string{"Result: allowed"}}}
	svc := authz.New(stubRepository{}, evaluator)
	req := rebac.CheckRequest{Subject: rebac.User("alice"), Permission: rebac.PermissionDocumentEdit, Resource: rebac.Document("roadmapDocument")}

	// Act
	result, err := svc.Check(t.Context(), req)

	// Assert (state): the Service returns whatever the evaluator produced.
	if err != nil {
		t.Fatalf("Check returned unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("Allowed = false, want true")
	}
}

func TestService_GivenEvaluatorFails_WhenCheck_ThenPropagatesError(t *testing.T) {
	// Arrange: a STUB evaluator that fails.
	wantErr := errors.New("evaluator exploded")
	svc := authz.New(stubRepository{}, stubEvaluator{err: wantErr})
	req := rebac.CheckRequest{Subject: rebac.User("alice"), Permission: rebac.PermissionDocumentEdit, Resource: rebac.Document("roadmapDocument")}

	// Act
	_, err := svc.Check(t.Context(), req)

	// Assert (state): the error is passed through unchanged.
	if !errors.Is(err, wantErr) {
		t.Errorf("Check error = %v, want %v", err, wantErr)
	}
}

func TestService_GivenCheckRequest_WhenCheck_ThenDelegatesExactRequestToEvaluator(t *testing.T) {
	// Arrange: a MOCK evaluator so we can verify the delegation, not the result.
	evaluator := &mockEvaluator{result: rebac.CheckResult{Allowed: true}}
	svc := authz.New(stubRepository{}, evaluator)
	req := rebac.CheckRequest{Subject: rebac.User("alice"), Permission: rebac.PermissionDocumentEdit, Resource: rebac.Document("roadmapDocument")}

	// Act
	if _, err := svc.Check(t.Context(), req); err != nil {
		t.Fatalf("Check returned unexpected error: %v", err)
	}

	// Assert (behaviour): exactly one delegation, with the request unchanged.
	if len(evaluator.calls) != 1 {
		t.Fatalf("evaluator called %d times, want 1", len(evaluator.calls))
	}
	if evaluator.calls[0] != req {
		t.Errorf("evaluator received %+v, want %+v", evaluator.calls[0], req)
	}
}

// ── WriteRelationships / DeleteRelationships ──────────────────────────────────────────────

func TestService_GivenRelationships_WhenWriteRelationships_ThenWritesEachToRepositoryInOrder(t *testing.T) {
	// Arrange: a MOCK repository to capture the Write interactions.
	repo := &mockRepository{}
	svc := authz.New(repo, stubEvaluator{})
	relationships := []rebac.Relationship{
		rebac.NewRelationship(rebac.Subject(rebac.User("alice")), rebac.RelationDocumentOwner, rebac.Document("d1")),
		rebac.NewRelationship(rebac.Subject(rebac.Workspace("ws")), rebac.RelationDocumentWorkspace, rebac.Document("d1")),
	}

	// Act
	if err := svc.WriteRelationships(t.Context(), relationships); err != nil {
		t.Fatalf("WriteRelationships returned unexpected error: %v", err)
	}

	// Assert (behaviour): one Write per relationship, same values, same order.
	if len(repo.writes) != len(relationships) {
		t.Fatalf("Write called %d times, want %d", len(repo.writes), len(relationships))
	}
	for i, want := range relationships {
		if repo.writes[i] != want {
			t.Errorf("writes[%d] = %+v, want %+v", i, repo.writes[i], want)
		}
	}
	if len(repo.deletes) != 0 {
		t.Errorf("Delete called %d times, want 0", len(repo.deletes))
	}
}

func TestService_GivenNoRelationships_WhenWriteRelationships_ThenRepositoryIsNotTouched(t *testing.T) {
	// Arrange
	repo := &mockRepository{}
	svc := authz.New(repo, stubEvaluator{})

	// Act
	if err := svc.WriteRelationships(t.Context(), nil); err != nil {
		t.Fatalf("WriteRelationships returned unexpected error: %v", err)
	}

	// Assert (behaviour): no interaction with the repository.
	if len(repo.writes) != 0 {
		t.Errorf("Write called %d times, want 0", len(repo.writes))
	}
}

func TestService_GivenRelationships_WhenDeleteRelationships_ThenDeletesEachFromRepository(t *testing.T) {
	// Arrange: a MOCK repository to capture the Delete interactions.
	repo := &mockRepository{}
	svc := authz.New(repo, stubEvaluator{})
	relationships := []rebac.Relationship{
		rebac.NewRelationship(rebac.Subject(rebac.User("alice")), rebac.RelationDocumentOwner, rebac.Document("d1")),
	}

	// Act
	if err := svc.DeleteRelationships(t.Context(), relationships); err != nil {
		t.Fatalf("DeleteRelationships returned unexpected error: %v", err)
	}

	// Assert (behaviour): one Delete with the exact relationship, and no writes.
	if len(repo.deletes) != 1 || repo.deletes[0] != relationships[0] {
		t.Errorf("deletes = %+v, want [%+v]", repo.deletes, relationships[0])
	}
	if len(repo.writes) != 0 {
		t.Errorf("Write called %d times, want 0", len(repo.writes))
	}
}

// ── ListRelationships ──────────────────────────────────────────────────────────────

func TestService_GivenStoredRelationships_WhenListRelationships_ThenReturnsRepositoryRelationships(t *testing.T) {
	// Arrange: a STUB repository with canned contents — we assert on the result.
	stored := []rebac.Relationship{
		rebac.NewRelationship(rebac.Subject(rebac.User("alice")), rebac.RelationTeamMember, rebac.Team("platformTeam")),
	}
	svc := authz.New(stubRepository{all: stored}, stubEvaluator{})

	// Act
	got, err := svc.ListRelationships(t.Context())

	// Assert (state)
	if err != nil {
		t.Fatalf("ListRelationships returned unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != stored[0] {
		t.Errorf("ListRelationships = %+v, want %+v", got, stored)
	}
}

func TestService_GivenFilter_WhenListRelationships_ThenPassesFilterToRepository(t *testing.T) {
	// Arrange: a MOCK repository so we can verify the filter is forwarded.
	repo := &mockRepository{}
	svc := authz.New(repo, stubEvaluator{})
	filter := authz.RelationshipFilter{Resource: rebac.Workspace("productWorkspace"), Relation: rebac.RelationWorkspaceEditor}

	// Act
	if _, err := svc.ListRelationships(t.Context(), filter); err != nil {
		t.Fatalf("ListRelationships returned unexpected error: %v", err)
	}

	// Assert (behaviour): FindAll received exactly the filter we passed.
	if len(repo.findFilters) != 1 {
		t.Fatalf("FindAll called %d times, want 1", len(repo.findFilters))
	}
	if len(repo.findFilters[0]) != 1 || repo.findFilters[0][0] != filter {
		t.Errorf("FindAll filter = %+v, want [%+v]", repo.findFilters[0], filter)
	}
}

// ── Relationship validation ────────────────────────────────────────────────

func TestService_GivenInvalidRelationship_WhenWriteRelationships_ThenReturnsValidationErrorAndWritesNothing(t *testing.T) {
	// Arrange
	// Each case is a relationship malformed in exactly one field. The Service must
	// reject the whole batch with a *RelationshipValidationError and never call Write.
	cases := map[string]rebac.Relationship{
		"resource missing type": {Resource: "roadmap", Relation: rebac.RelationDocumentOwner, Subject: rebac.Subject(rebac.User("alice"))},
		"unknown resource type": {Resource: "widget:1", Relation: rebac.RelationDocumentOwner, Subject: rebac.Subject(rebac.User("alice"))},
		"empty relation":        {Resource: rebac.Document("d1"), Relation: "", Subject: rebac.Subject(rebac.User("alice"))},
		"subject missing type":  {Resource: rebac.Document("d1"), Relation: rebac.RelationDocumentOwner, Subject: "alice"},
		"subject set missing resource type": {
			Resource: rebac.Document("d1"),
			Relation: rebac.RelationDocumentOwner,
			Subject:  "platformTeam#member",
		},
		"unknown relation for resource": {
			Resource: rebac.Team("platformTeam"), Relation: rebac.Relation(rebac.PermissionDocumentRead), Subject: rebac.Subject(rebac.User("alice")),
		},
		"permission cannot be written as relation": {
			Resource: rebac.Document("d1"), Relation: rebac.Relation(rebac.PermissionDocumentEdit), Subject: rebac.Subject(rebac.User("alice")),
		},
		"workspace pointer must reference workspace": {
			Resource: rebac.Document("d1"), Relation: rebac.RelationDocumentWorkspace, Subject: rebac.Subject(rebac.User("alice")),
		},
		"workspace owner requires team admin subject set": {
			Resource: rebac.Workspace("productWorkspace"), Relation: rebac.RelationWorkspaceOwner,
			Subject: rebac.SubjectSet(rebac.Team("platformTeam"), rebac.RelationTeamMember),
		},
	}

	for name, relationship := range cases {
		t.Run(name, func(t *testing.T) {
			// Arrange
			repo := &mockRepository{}
			svc := authz.New(repo, stubEvaluator{})

			// Act
			err := svc.WriteRelationships(t.Context(), []rebac.Relationship{relationship})

			// Assert
			var verr *authz.RelationshipValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("expected *RelationshipValidationError, got %v", err)
			}
			if len(repo.writes) != 0 {
				t.Errorf("expected no writes when a relationship is invalid, got %d", len(repo.writes))
			}
		})
	}
}

func TestService_GivenInvalidCheck_WhenCheck_ThenRejectsBeforeEvaluator(t *testing.T) {
	// Arrange
	cases := map[string]rebac.CheckRequest{
		"subject must be user": {
			Subject:    rebac.Team("platformTeam"),
			Permission: rebac.PermissionDocumentEdit,
			Resource:   rebac.Document("d1"),
		},
		"relation cannot be supplied as permission": {
			Subject:    rebac.User("alice"),
			Permission: rebac.Permission(rebac.RelationDocumentWorkspace),
			Resource:   rebac.Document("d1"),
		},
	}

	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			// Arrange
			evaluator := &mockEvaluator{}
			svc := authz.New(stubRepository{}, evaluator)

			// Act
			_, err := svc.Check(t.Context(), req)

			// Assert
			var validationErr *authz.CheckValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected *CheckValidationError, got %v", err)
			}
			if len(evaluator.calls) != 0 {
				t.Errorf("evaluator called %d times, want 0", len(evaluator.calls))
			}
		})
	}
}

func TestService_GivenValidSubjectSetRelationship_WhenWriteRelationships_ThenSucceeds(t *testing.T) {
	// Arrange
	// A subject set ("team:platformTeam#member") is a valid relationship subject.
	repo := &mockRepository{}
	svc := authz.New(repo, stubEvaluator{})
	relationship := rebac.NewRelationship(
		rebac.SubjectSet(rebac.Team("platformTeam"), rebac.RelationTeamMember),
		rebac.RelationWorkspaceEditor,
		rebac.Workspace("productWorkspace"),
	)

	// Act
	err := svc.WriteRelationships(t.Context(), []rebac.Relationship{relationship})

	// Assert
	if err != nil {
		t.Fatalf("expected subject-set relationship to be valid, got %v", err)
	}
	if len(repo.writes) != 1 {
		t.Errorf("expected 1 write, got %d", len(repo.writes))
	}
}
