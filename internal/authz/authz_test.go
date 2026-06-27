package authz_test

import (
	"context"
	"errors"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/rebac"
)

// This file unit-tests the *authz.Service returned by [authz.New] in
// isolation from any real adapter. The Service is a thin orchestrator over two
// driven ports — TupleRepository and Evaluator — which makes it the right place
// to demonstrate the difference between stubs and mocks:
//
//   - A STUB stands in for a collaborator and returns canned answers. It is used
//     for STATE verification: "given the evaluator says allowed, does Check
//     return allowed?" The test never inspects how the stub was called.
//
//   - A MOCK also stands in for a collaborator but, in addition, records the
//     calls it received. It is used for BEHAVIOUR verification: "does WriteTuples
//     call repository.Write once per tuple, with the exact tuples, in order?"
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

// stubRepository is a STUB TupleRepository whose reads return canned data and
// whose writes are no-ops. Tests that exercise the evaluator path pass this so
// the Service has a collaborator without caring how it is used.
type stubRepository struct {
	all []rebac.TupleKey
}

func (s stubRepository) Has(context.Context, rebac.Object, rebac.Relation, rebac.Subject) (bool, error) {
	return false, nil
}
func (s stubRepository) FindByObjectRelation(context.Context, rebac.Object, rebac.Relation) ([]rebac.TupleKey, error) {
	return nil, nil
}
func (s stubRepository) FindAll(context.Context, ...authz.TupleFilter) ([]rebac.TupleKey, error) {
	return s.all, nil
}
func (s stubRepository) Write(context.Context, rebac.TupleKey) error  { return nil }
func (s stubRepository) Delete(context.Context, rebac.TupleKey) error { return nil }

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

// mockRepository is a MOCK TupleRepository: it records the Write/Delete calls and
// the filters passed to FindAll, so tests can verify the Service's interactions
// with persistence.
type mockRepository struct {
	writes      []rebac.TupleKey
	deletes     []rebac.TupleKey
	findFilters [][]authz.TupleFilter
	findResult  []rebac.TupleKey
}

func (m *mockRepository) Has(context.Context, rebac.Object, rebac.Relation, rebac.Subject) (bool, error) {
	return false, nil
}
func (m *mockRepository) FindByObjectRelation(context.Context, rebac.Object, rebac.Relation) ([]rebac.TupleKey, error) {
	return nil, nil
}
func (m *mockRepository) FindAll(_ context.Context, filter ...authz.TupleFilter) ([]rebac.TupleKey, error) {
	m.findFilters = append(m.findFilters, filter)
	return m.findResult, nil
}
func (m *mockRepository) Write(_ context.Context, t rebac.TupleKey) error {
	m.writes = append(m.writes, t)
	return nil
}
func (m *mockRepository) Delete(_ context.Context, t rebac.TupleKey) error {
	m.deletes = append(m.deletes, t)
	return nil
}

// Compile-time checks that the doubles satisfy the ports they stand in for.
var (
	_ authz.Evaluator       = stubEvaluator{}
	_ authz.Evaluator       = (*mockEvaluator)(nil)
	_ authz.TupleRepository = stubRepository{}
	_ authz.TupleRepository = (*mockRepository)(nil)
)

func sampleRequest() rebac.CheckRequest {
	return rebac.CheckRequest{
		User:     rebac.User("alice"),
		Relation: rebac.RelationDocumentCanEdit,
		Object:   rebac.Document("roadmapDocument"),
	}
}

// ── Check ───────────────────────────────────────────────────────────────────

func TestService_GivenEvaluatorAllows_WhenCheck_ThenReturnsEvaluatorResult(t *testing.T) {
	// Arrange: a STUB evaluator pinned to an allowed result.
	evaluator := stubEvaluator{result: rebac.CheckResult{Allowed: true, Trace: []string{"Result: allowed"}}}
	svc := authz.New(stubRepository{}, evaluator)

	// Act
	result, err := svc.Check(context.Background(), sampleRequest())

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

	// Act
	_, err := svc.Check(context.Background(), sampleRequest())

	// Assert (state): the error is passed through unchanged.
	if !errors.Is(err, wantErr) {
		t.Errorf("Check error = %v, want %v", err, wantErr)
	}
}

func TestService_GivenCheckRequest_WhenCheck_ThenDelegatesExactRequestToEvaluator(t *testing.T) {
	// Arrange: a MOCK evaluator so we can verify the delegation, not the result.
	evaluator := &mockEvaluator{result: rebac.CheckResult{Allowed: true}}
	svc := authz.New(stubRepository{}, evaluator)
	req := sampleRequest()

	// Act
	if _, err := svc.Check(context.Background(), req); err != nil {
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

// ── WriteTuples / DeleteTuples ──────────────────────────────────────────────

func TestService_GivenTuples_WhenWriteTuples_ThenWritesEachToRepositoryInOrder(t *testing.T) {
	// Arrange: a MOCK repository to capture the Write interactions.
	repo := &mockRepository{}
	svc := authz.New(repo, stubEvaluator{})
	tuples := []rebac.TupleKey{
		rebac.Tuple(rebac.Document("d1"), rebac.RelationDocumentOwner, rebac.Subject(rebac.User("alice"))),
		rebac.Tuple(rebac.Document("d1"), rebac.RelationDocumentWorkspace, rebac.Subject(rebac.Workspace("ws"))),
	}

	// Act
	if err := svc.WriteTuples(context.Background(), tuples); err != nil {
		t.Fatalf("WriteTuples returned unexpected error: %v", err)
	}

	// Assert (behaviour): one Write per tuple, same values, same order.
	if len(repo.writes) != len(tuples) {
		t.Fatalf("Write called %d times, want %d", len(repo.writes), len(tuples))
	}
	for i, want := range tuples {
		if repo.writes[i] != want {
			t.Errorf("writes[%d] = %+v, want %+v", i, repo.writes[i], want)
		}
	}
	if len(repo.deletes) != 0 {
		t.Errorf("Delete called %d times, want 0", len(repo.deletes))
	}
}

func TestService_GivenNoTuples_WhenWriteTuples_ThenRepositoryIsNotTouched(t *testing.T) {
	// Arrange
	repo := &mockRepository{}
	svc := authz.New(repo, stubEvaluator{})

	// Act
	if err := svc.WriteTuples(context.Background(), nil); err != nil {
		t.Fatalf("WriteTuples returned unexpected error: %v", err)
	}

	// Assert (behaviour): no interaction with the repository.
	if len(repo.writes) != 0 {
		t.Errorf("Write called %d times, want 0", len(repo.writes))
	}
}

func TestService_GivenTuples_WhenDeleteTuples_ThenDeletesEachFromRepository(t *testing.T) {
	// Arrange: a MOCK repository to capture the Delete interactions.
	repo := &mockRepository{}
	svc := authz.New(repo, stubEvaluator{})
	tuples := []rebac.TupleKey{
		rebac.Tuple(rebac.Document("d1"), rebac.RelationDocumentOwner, rebac.Subject(rebac.User("alice"))),
	}

	// Act
	if err := svc.DeleteTuples(context.Background(), tuples); err != nil {
		t.Fatalf("DeleteTuples returned unexpected error: %v", err)
	}

	// Assert (behaviour): one Delete with the exact tuple, and no writes.
	if len(repo.deletes) != 1 || repo.deletes[0] != tuples[0] {
		t.Errorf("deletes = %+v, want [%+v]", repo.deletes, tuples[0])
	}
	if len(repo.writes) != 0 {
		t.Errorf("Write called %d times, want 0", len(repo.writes))
	}
}

// ── ListTuples ──────────────────────────────────────────────────────────────

func TestService_GivenStoredTuples_WhenListTuples_ThenReturnsRepositoryTuples(t *testing.T) {
	// Arrange: a STUB repository with canned contents — we assert on the result.
	stored := []rebac.TupleKey{
		rebac.Tuple(rebac.Team("platformTeam"), rebac.RelationTeamMember, rebac.Subject(rebac.User("alice"))),
	}
	svc := authz.New(stubRepository{all: stored}, stubEvaluator{})

	// Act
	got, err := svc.ListTuples(context.Background())

	// Assert (state)
	if err != nil {
		t.Fatalf("ListTuples returned unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != stored[0] {
		t.Errorf("ListTuples = %+v, want %+v", got, stored)
	}
}

func TestService_GivenFilter_WhenListTuples_ThenPassesFilterToRepository(t *testing.T) {
	// Arrange: a MOCK repository so we can verify the filter is forwarded.
	repo := &mockRepository{}
	svc := authz.New(repo, stubEvaluator{})
	filter := authz.TupleFilter{Object: rebac.Workspace("productWorkspace"), Relation: rebac.RelationWorkspaceEditor}

	// Act
	if _, err := svc.ListTuples(context.Background(), filter); err != nil {
		t.Fatalf("ListTuples returned unexpected error: %v", err)
	}

	// Assert (behaviour): FindAll received exactly the filter we passed.
	if len(repo.findFilters) != 1 {
		t.Fatalf("FindAll called %d times, want 1", len(repo.findFilters))
	}
	if len(repo.findFilters[0]) != 1 || repo.findFilters[0][0] != filter {
		t.Errorf("FindAll filter = %+v, want [%+v]", repo.findFilters[0], filter)
	}
}

// ── Tuple validation ────────────────────────────────────────────────────────

func TestService_GivenInvalidTuple_WhenWriteTuples_ThenReturnsValidationErrorAndWritesNothing(t *testing.T) {
	// Each case is a tuple that is malformed in exactly one field. The Service must
	// reject the whole batch with a *TupleValidationError and never call Write.
	cases := map[string]rebac.TupleKey{
		"object missing type": {Object: "roadmap", Relation: rebac.RelationDocumentOwner, User: rebac.Subject(rebac.User("alice"))},
		"unknown object type": {Object: "widget:1", Relation: rebac.RelationDocumentOwner, User: rebac.Subject(rebac.User("alice"))},
		"empty relation":      {Object: rebac.Document("d1"), Relation: "", User: rebac.Subject(rebac.User("alice"))},
		"user missing type":   {Object: rebac.Document("d1"), Relation: rebac.RelationDocumentOwner, User: "alice"},
		"subject set missing object type": {
			Object:   rebac.Document("d1"),
			Relation: rebac.RelationDocumentOwner,
			User:     "platformTeam#member",
		},
		"unknown relation for object": {
			Object: rebac.Team("platformTeam"), Relation: rebac.RelationDocumentCanRead, User: rebac.Subject(rebac.User("alice")),
		},
		"computed relation cannot be written": {
			Object: rebac.Document("d1"), Relation: rebac.RelationDocumentCanEdit, User: rebac.Subject(rebac.User("alice")),
		},
		"workspace pointer must reference workspace": {
			Object: rebac.Document("d1"), Relation: rebac.RelationDocumentWorkspace, User: rebac.Subject(rebac.User("alice")),
		},
		"workspace owner requires team admin subject set": {
			Object: rebac.Workspace("productWorkspace"), Relation: rebac.RelationWorkspaceOwner,
			User: rebac.SubjectSet(rebac.Team("platformTeam"), rebac.RelationTeamMember),
		},
	}

	for name, tk := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &mockRepository{}
			svc := authz.New(repo, stubEvaluator{})

			err := svc.WriteTuples(context.Background(), []rebac.TupleKey{tk})

			var verr *authz.TupleValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("expected *TupleValidationError, got %v", err)
			}
			if len(repo.writes) != 0 {
				t.Errorf("expected no writes when a tuple is invalid, got %d", len(repo.writes))
			}
		})
	}
}

func TestService_GivenInvalidCheck_WhenCheck_ThenRejectsBeforeEvaluator(t *testing.T) {
	cases := map[string]rebac.CheckRequest{
		"subject must be user": {
			User:     rebac.Team("platformTeam"),
			Relation: rebac.RelationDocumentCanEdit,
			Object:   rebac.Document("d1"),
		},
		"structural relation cannot be checked for user": {
			User:     rebac.User("alice"),
			Relation: rebac.RelationDocumentWorkspace,
			Object:   rebac.Document("d1"),
		},
	}

	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			evaluator := &mockEvaluator{}
			svc := authz.New(stubRepository{}, evaluator)

			_, err := svc.Check(context.Background(), req)

			var validationErr *authz.TupleValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected *TupleValidationError, got %v", err)
			}
			if len(evaluator.calls) != 0 {
				t.Errorf("evaluator called %d times, want 0", len(evaluator.calls))
			}
		})
	}
}

func TestService_GivenValidSubjectSetTuple_WhenWriteTuples_ThenSucceeds(t *testing.T) {
	// A subject-set user ("team:platformTeam#member") is a valid User value and
	// must pass validation.
	repo := &mockRepository{}
	svc := authz.New(repo, stubEvaluator{})
	tuple := rebac.Tuple(
		rebac.Workspace("productWorkspace"),
		rebac.RelationWorkspaceEditor,
		rebac.SubjectSet(rebac.Team("platformTeam"), rebac.RelationTeamMember),
	)

	if err := svc.WriteTuples(context.Background(), []rebac.TupleKey{tuple}); err != nil {
		t.Fatalf("expected subject-set tuple to be valid, got %v", err)
	}
	if len(repo.writes) != 1 {
		t.Errorf("expected 1 write, got %d", len(repo.writes))
	}
}
