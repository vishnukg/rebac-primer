package db_test

import (
	"sync"
	"testing"

	"rebac-primer/internal/authz"
	authzdb "rebac-primer/internal/authz/adapters/db"
	"rebac-primer/internal/shared"
)

// These tests cover the in-memory TupleRepository adapter. The store is a
// self-contained stateful unit with no collaborators, so no test doubles are
// needed: each test arranges real tuples, acts on the store, and asserts on its
// observable state.

func aliceMember() shared.TupleKey {
	return shared.Tuple(shared.Team("platformTeam"), shared.RelationTeamMember, shared.Subject(shared.User("alice")))
}

func bobViewer() shared.TupleKey {
	return shared.Tuple(shared.Workspace("productWorkspace"), shared.RelationWorkspaceViewer, shared.Subject(shared.User("bob")))
}

func TestStore_GivenSeededTuple_WhenHas_ThenReportsTrue(t *testing.T) {
	// Arrange
	tuple := aliceMember()
	store := authzdb.New(tuple)

	// Act
	got := store.Has(tuple.Object, tuple.Relation, tuple.User)

	// Assert
	if !got {
		t.Errorf("Has(%+v) = false, want true", tuple)
	}
}

func TestStore_GivenEmptyStore_WhenHas_ThenReportsFalse(t *testing.T) {
	// Arrange
	store := authzdb.New()
	tuple := aliceMember()

	// Act
	got := store.Has(tuple.Object, tuple.Relation, tuple.User)

	// Assert
	if got {
		t.Errorf("Has on empty store = true, want false")
	}
}

func TestStore_GivenWrittenTuple_WhenHas_ThenReportsTrue(t *testing.T) {
	// Arrange
	store := authzdb.New()
	tuple := aliceMember()

	// Act
	store.Write(tuple)

	// Assert
	if !store.Has(tuple.Object, tuple.Relation, tuple.User) {
		t.Errorf("Has after Write = false, want true")
	}
}

func TestStore_GivenDuplicateWrites_WhenFindAll_ThenTupleStoredOnce(t *testing.T) {
	// Arrange
	store := authzdb.New()
	tuple := aliceMember()

	// Act: writing the same tuple twice must be idempotent.
	store.Write(tuple)
	store.Write(tuple)

	// Assert
	if got := store.FindAll(); len(got) != 1 {
		t.Errorf("FindAll length = %d, want 1 (writes must be idempotent)", len(got))
	}
}

func TestStore_GivenStoredTuple_WhenDeleted_ThenHasReportsFalse(t *testing.T) {
	// Arrange
	tuple := aliceMember()
	store := authzdb.New(tuple)

	// Act
	store.Delete(tuple)

	// Assert
	if store.Has(tuple.Object, tuple.Relation, tuple.User) {
		t.Errorf("Has after Delete = true, want false")
	}
}

func TestStore_GivenMissingTuple_WhenDeleted_ThenNoOp(t *testing.T) {
	// Arrange
	store := authzdb.New(aliceMember())

	// Act: deleting a tuple that was never written must not affect the store.
	store.Delete(bobViewer())

	// Assert
	if got := store.FindAll(); len(got) != 1 {
		t.Errorf("FindAll length = %d, want 1 (deleting a missing tuple is a no-op)", len(got))
	}
}

func TestStore_GivenMixedTuples_WhenFindByObjectRelation_ThenReturnsOnlyMatches(t *testing.T) {
	// Arrange
	match := bobViewer()
	store := authzdb.New(match, aliceMember())

	// Act
	got := store.FindByObjectRelation(match.Object, match.Relation)

	// Assert
	if len(got) != 1 || got[0] != match {
		t.Errorf("FindByObjectRelation = %+v, want [%+v]", got, match)
	}
}

func TestStore_GivenFilter_WhenFindAll_ThenReturnsMatchingTuples(t *testing.T) {
	// Arrange
	store := authzdb.New(aliceMember(), bobViewer())

	cases := map[string]struct {
		filter authz.TupleFilter
		want   int
	}{
		"no filter matches all":        {authz.TupleFilter{}, 2},
		"by object":                    {authz.TupleFilter{Object: shared.Team("platformTeam")}, 1},
		"by relation":                  {authz.TupleFilter{Relation: shared.RelationWorkspaceViewer}, 1},
		"by object and relation":       {authz.TupleFilter{Object: shared.Team("platformTeam"), Relation: shared.RelationTeamMember}, 1},
		"non-matching filter is empty": {authz.TupleFilter{Object: shared.Team("noSuchTeam")}, 0},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			got := store.FindAll(tc.filter)

			// Assert
			if len(got) != tc.want {
				t.Errorf("FindAll(%+v) length = %d, want %d", tc.filter, len(got), tc.want)
			}
		})
	}
}

func TestStore_GivenConcurrentWrites_WhenFindAll_ThenAllTuplesStored(t *testing.T) {
	// Arrange: distinct tuples written from many goroutines. With -race this
	// exercises the store's mutex.
	store := authzdb.New()
	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)

	// Act
	for i := range n {
		go func(i int) {
			defer wg.Done()
			id := string(rune('A'+i%26)) + string(rune('0'+i/26))
			store.Write(shared.Tuple(shared.Team(id), shared.RelationTeamMember, shared.Subject(shared.User("alice"))))
		}(i)
	}
	wg.Wait()

	// Assert
	if got := store.FindAll(); len(got) != n {
		t.Errorf("FindAll length = %d, want %d", len(got), n)
	}
}
