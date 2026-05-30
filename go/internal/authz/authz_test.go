package authz_test

import (
	"context"
	"errors"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/shared"
)

// This file unit-tests the authz [authz.Service] returned by [authz.New] in
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
	result shared.CheckResult
	err    error
}

func (s stubEvaluator) Evaluate(context.Context, shared.CheckRequest) (shared.CheckResult, error) {
	return s.result, s.err
}

// stubRepository is a STUB TupleRepository whose reads return canned data and
// whose writes are no-ops. Tests that exercise the evaluator path pass this so
// the Service has a collaborator without caring how it is used.
type stubRepository struct {
	all []shared.TupleKey
}

func (s stubRepository) Has(shared.Object, shared.Relation, shared.Subject) bool { return false }
func (s stubRepository) FindByObjectRelation(shared.Object, shared.Relation) []shared.TupleKey {
	return nil
}
func (s stubRepository) FindAll(...authz.TupleFilter) []shared.TupleKey { return s.all }
func (s stubRepository) Write(shared.TupleKey)                          {}
func (s stubRepository) Delete(shared.TupleKey)                         {}

// ── Mocks (behaviour verification) ──────────────────────────────────────────

// mockEvaluator is a MOCK: it records every request it is asked to evaluate so a
// test can assert the Service delegated the exact CheckRequest unchanged.
type mockEvaluator struct {
	calls  []shared.CheckRequest
	result shared.CheckResult
}

func (m *mockEvaluator) Evaluate(_ context.Context, req shared.CheckRequest) (shared.CheckResult, error) {
	m.calls = append(m.calls, req)
	return m.result, nil
}

// mockRepository is a MOCK TupleRepository: it records the Write/Delete calls and
// the filters passed to FindAll, so tests can verify the Service's interactions
// with persistence.
type mockRepository struct {
	writes      []shared.TupleKey
	deletes     []shared.TupleKey
	findFilters [][]authz.TupleFilter
	findResult  []shared.TupleKey
}

func (m *mockRepository) Has(shared.Object, shared.Relation, shared.Subject) bool { return false }
func (m *mockRepository) FindByObjectRelation(shared.Object, shared.Relation) []shared.TupleKey {
	return nil
}
func (m *mockRepository) FindAll(filter ...authz.TupleFilter) []shared.TupleKey {
	m.findFilters = append(m.findFilters, filter)
	return m.findResult
}
func (m *mockRepository) Write(t shared.TupleKey)  { m.writes = append(m.writes, t) }
func (m *mockRepository) Delete(t shared.TupleKey) { m.deletes = append(m.deletes, t) }

// Compile-time checks that the doubles satisfy the ports they stand in for.
var (
	_ authz.Evaluator       = stubEvaluator{}
	_ authz.Evaluator       = (*mockEvaluator)(nil)
	_ authz.TupleRepository = stubRepository{}
	_ authz.TupleRepository = (*mockRepository)(nil)
)

func sampleRequest() shared.CheckRequest {
	return shared.CheckRequest{
		User:     shared.User("alice"),
		Relation: shared.RelationDocumentCanEdit,
		Object:   shared.Document("roadmapDocument"),
	}
}

// ── Check ───────────────────────────────────────────────────────────────────

func TestService_GivenEvaluatorAllows_WhenCheck_ThenReturnsEvaluatorResult(t *testing.T) {
	// Arrange: a STUB evaluator pinned to an allowed result.
	evaluator := stubEvaluator{result: shared.CheckResult{Allowed: true, Trace: []string{"Result: allowed"}}}
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
	evaluator := &mockEvaluator{result: shared.CheckResult{Allowed: true}}
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
	tuples := []shared.TupleKey{
		shared.Tuple(shared.Document("d1"), shared.RelationDocumentOwner, shared.Subject(shared.User("alice"))),
		shared.Tuple(shared.Document("d1"), shared.RelationDocumentWorkspace, shared.Subject(shared.Workspace("ws"))),
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
	tuples := []shared.TupleKey{
		shared.Tuple(shared.Document("d1"), shared.RelationDocumentOwner, shared.Subject(shared.User("alice"))),
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
	stored := []shared.TupleKey{
		shared.Tuple(shared.Team("platformTeam"), shared.RelationTeamMember, shared.Subject(shared.User("alice"))),
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
	filter := authz.TupleFilter{Object: shared.Workspace("productWorkspace"), Relation: shared.RelationWorkspaceEditor}

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
