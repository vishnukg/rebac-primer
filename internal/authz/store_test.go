package authz_test

import (
	"sync"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/rebac"
)

// These tests cover the in-memory tuple-store adapter. The store is a
// self-contained stateful unit with no collaborators, so no test doubles are
// needed: each test arranges real tuples, acts on the store, and asserts on its
// observable state.
//
// The store's methods take a context.Context and return an error to satisfy the
// port. Tests check those errors even though this in-memory implementation does
// not currently fail, keeping the examples safe to copy to real adapters.

func TestStore_GivenSeededTuple_WhenHas_ThenReportsTrue(t *testing.T) {
	// Arrange
	tuple := rebac.Tuple(rebac.Team("platformTeam"), rebac.RelationTeamMember, rebac.Subject(rebac.User("alice")))
	store := authz.NewInMemoryStore(tuple)

	// Act
	got, err := store.Has(t.Context(), tuple.Object, tuple.Relation, tuple.User)

	// Assert
	if err != nil {
		t.Fatalf("Has returned unexpected error: %v", err)
	}
	if !got {
		t.Errorf("Has(%+v) = false, want true", tuple)
	}
}

func TestStore_GivenEmptyStore_WhenHas_ThenReportsFalse(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore()
	tuple := rebac.Tuple(rebac.Team("platformTeam"), rebac.RelationTeamMember, rebac.Subject(rebac.User("alice")))

	// Act
	got, err := store.Has(t.Context(), tuple.Object, tuple.Relation, tuple.User)

	// Assert
	if err != nil {
		t.Fatalf("Has returned unexpected error: %v", err)
	}
	if got {
		t.Errorf("Has on empty store = true, want false")
	}
}

func TestStore_GivenWrittenTuple_WhenHas_ThenReportsTrue(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore()
	tuple := rebac.Tuple(rebac.Team("platformTeam"), rebac.RelationTeamMember, rebac.Subject(rebac.User("alice")))

	// Act
	if err := store.Write(t.Context(), tuple); err != nil {
		t.Fatalf("Write returned unexpected error: %v", err)
	}

	// Assert
	got, err := store.Has(t.Context(), tuple.Object, tuple.Relation, tuple.User)
	if err != nil {
		t.Fatalf("Has returned unexpected error: %v", err)
	}
	if !got {
		t.Errorf("Has after Write = false, want true")
	}
}

func TestStore_GivenDuplicateWrites_WhenFindAll_ThenTupleStoredOnce(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore()
	tuple := rebac.Tuple(rebac.Team("platformTeam"), rebac.RelationTeamMember, rebac.Subject(rebac.User("alice")))

	// Act: writing the same tuple twice must be idempotent.
	if err := store.Write(t.Context(), tuple); err != nil {
		t.Fatalf("first Write returned unexpected error: %v", err)
	}
	if err := store.Write(t.Context(), tuple); err != nil {
		t.Fatalf("second Write returned unexpected error: %v", err)
	}

	// Assert
	got, err := store.FindAll(t.Context())
	if err != nil {
		t.Fatalf("FindAll returned unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("FindAll length = %d, want 1 (writes must be idempotent)", len(got))
	}
}

func TestStore_GivenStoredTuple_WhenDeleted_ThenHasReportsFalse(t *testing.T) {
	// Arrange
	tuple := rebac.Tuple(rebac.Team("platformTeam"), rebac.RelationTeamMember, rebac.Subject(rebac.User("alice")))
	store := authz.NewInMemoryStore(tuple)

	// Act
	if err := store.Delete(t.Context(), tuple); err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	// Assert
	got, err := store.Has(t.Context(), tuple.Object, tuple.Relation, tuple.User)
	if err != nil {
		t.Fatalf("Has returned unexpected error: %v", err)
	}
	if got {
		t.Errorf("Has after Delete = true, want false")
	}
}

func TestStore_GivenMissingTuple_WhenDeleted_ThenNoOp(t *testing.T) {
	// Arrange
	aliceMember := rebac.Tuple(
		rebac.Team("platformTeam"),
		rebac.RelationTeamMember,
		rebac.Subject(rebac.User("alice")),
	)
	bobViewer := rebac.Tuple(
		rebac.Workspace("productWorkspace"),
		rebac.RelationWorkspaceViewer,
		rebac.Subject(rebac.User("bob")),
	)
	store := authz.NewInMemoryStore(aliceMember)

	// Act: deleting a tuple that was never written must not affect the store.
	if err := store.Delete(t.Context(), bobViewer); err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	// Assert
	got, err := store.FindAll(t.Context())
	if err != nil {
		t.Fatalf("FindAll returned unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("FindAll length = %d, want 1 (deleting a missing tuple is a no-op)", len(got))
	}
}

func TestStore_GivenMixedTuples_WhenFindByObjectRelation_ThenReturnsOnlyMatches(t *testing.T) {
	// Arrange
	match := rebac.Tuple(rebac.Workspace("productWorkspace"), rebac.RelationWorkspaceViewer, rebac.Subject(rebac.User("bob")))
	nonMatch := rebac.Tuple(rebac.Team("platformTeam"), rebac.RelationTeamMember, rebac.Subject(rebac.User("alice")))
	store := authz.NewInMemoryStore(match, nonMatch)

	// Act
	got, err := store.FindByObjectRelation(t.Context(), match.Object, match.Relation)

	// Assert
	if err != nil {
		t.Fatalf("FindByObjectRelation returned unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != match {
		t.Errorf("FindByObjectRelation = %+v, want [%+v]", got, match)
	}
}

func TestStore_GivenFilter_WhenFindAll_ThenReturnsMatchingTuples(t *testing.T) {
	// Arrange
	cases := map[string]struct {
		filter authz.TupleFilter
		want   int
	}{
		"no filter matches all":        {authz.TupleFilter{}, 2},
		"by object":                    {authz.TupleFilter{Object: rebac.Team("platformTeam")}, 1},
		"by relation":                  {authz.TupleFilter{Relation: rebac.RelationWorkspaceViewer}, 1},
		"by object and relation":       {authz.TupleFilter{Object: rebac.Team("platformTeam"), Relation: rebac.RelationTeamMember}, 1},
		"non-matching filter is empty": {authz.TupleFilter{Object: rebac.Team("noSuchTeam")}, 0},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Arrange: every subtest owns its store and seed data.
			aliceMember := rebac.Tuple(rebac.Team("platformTeam"), rebac.RelationTeamMember, rebac.Subject(rebac.User("alice")))
			bobViewer := rebac.Tuple(rebac.Workspace("productWorkspace"), rebac.RelationWorkspaceViewer, rebac.Subject(rebac.User("bob")))
			store := authz.NewInMemoryStore(aliceMember, bobViewer)

			// Act
			got, err := store.FindAll(t.Context(), tc.filter)

			// Assert
			if err != nil {
				t.Fatalf("FindAll returned unexpected error: %v", err)
			}
			if len(got) != tc.want {
				t.Errorf("FindAll(%+v) length = %d, want %d", tc.filter, len(got), tc.want)
			}
		})
	}
}

func TestStore_GivenTuples_WhenFindAll_ThenReturnsDeterministicOrder(t *testing.T) {
	// Arrange: write in reverse lexical order.
	aliceMember := rebac.Tuple(rebac.Team("platformTeam"), rebac.RelationTeamMember, rebac.Subject(rebac.User("alice")))
	bobViewer := rebac.Tuple(rebac.Workspace("productWorkspace"), rebac.RelationWorkspaceViewer, rebac.Subject(rebac.User("bob")))
	store := authz.NewInMemoryStore(bobViewer, aliceMember)

	// Act
	got, err := store.FindAll(t.Context())

	// Assert: responses should not depend on Go's randomized map iteration order.
	if err != nil {
		t.Fatalf("FindAll returned unexpected error: %v", err)
	}
	want := []rebac.TupleKey{aliceMember, bobViewer}
	if len(got) != len(want) {
		t.Fatalf("FindAll length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("FindAll[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestStore_GivenConcurrentWrites_WhenFindAll_ThenAllTuplesStored(t *testing.T) {
	// Arrange: distinct tuples written from many goroutines. With -race this
	// exercises the store's mutex.
	store := authz.NewInMemoryStore()
	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)

	// Act
	for i := range n {
		go func(i int) {
			defer wg.Done()
			id := string(rune('A'+i%26)) + string(rune('0'+i/26))
			if err := store.Write(t.Context(), rebac.Tuple(rebac.Team(id), rebac.RelationTeamMember, rebac.Subject(rebac.User("alice")))); err != nil {
				t.Errorf("Write returned unexpected error: %v", err)
			}
		}(i)
	}
	wg.Wait()

	// Assert
	got, err := store.FindAll(t.Context())
	if err != nil {
		t.Fatalf("FindAll returned unexpected error: %v", err)
	}
	if len(got) != n {
		t.Errorf("FindAll length = %d, want %d", len(got), n)
	}
}
