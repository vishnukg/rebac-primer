package authz_test

import (
	"context"
	"sync"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/rebac"
)

// These tests cover the in-memory TupleRepository adapter. The store is a
// self-contained stateful unit with no collaborators, so no test doubles are
// needed: each test arranges real tuples, acts on the store, and asserts on its
// observable state.
//
// The store's methods take a context.Context and return an error to satisfy the
// port (a real backend can fail or be cancelled). The in-memory store never
// fails, so these tests pass context.Background() and ignore the nil error.

func aliceMember() rebac.TupleKey {
	return rebac.Tuple(rebac.Team("platformTeam"), rebac.RelationTeamMember, rebac.Subject(rebac.User("alice")))
}

func bobViewer() rebac.TupleKey {
	return rebac.Tuple(rebac.Workspace("productWorkspace"), rebac.RelationWorkspaceViewer, rebac.Subject(rebac.User("bob")))
}

func TestStore_GivenSeededTuple_WhenHas_ThenReportsTrue(t *testing.T) {
	// Arrange
	tuple := aliceMember()
	store := authz.NewInMemoryStore(tuple)

	// Act
	got, _ := store.Has(context.Background(), tuple.Object, tuple.Relation, tuple.User)

	// Assert
	if !got {
		t.Errorf("Has(%+v) = false, want true", tuple)
	}
}

func TestStore_GivenEmptyStore_WhenHas_ThenReportsFalse(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore()
	tuple := aliceMember()

	// Act
	got, _ := store.Has(context.Background(), tuple.Object, tuple.Relation, tuple.User)

	// Assert
	if got {
		t.Errorf("Has on empty store = true, want false")
	}
}

func TestStore_GivenWrittenTuple_WhenHas_ThenReportsTrue(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore()
	tuple := aliceMember()

	// Act
	store.Write(context.Background(), tuple)

	// Assert
	if got, _ := store.Has(context.Background(), tuple.Object, tuple.Relation, tuple.User); !got {
		t.Errorf("Has after Write = false, want true")
	}
}

func TestStore_GivenDuplicateWrites_WhenFindAll_ThenTupleStoredOnce(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore()
	tuple := aliceMember()

	// Act: writing the same tuple twice must be idempotent.
	store.Write(context.Background(), tuple)
	store.Write(context.Background(), tuple)

	// Assert
	if got, _ := store.FindAll(context.Background()); len(got) != 1 {
		t.Errorf("FindAll length = %d, want 1 (writes must be idempotent)", len(got))
	}
}

func TestStore_GivenStoredTuple_WhenDeleted_ThenHasReportsFalse(t *testing.T) {
	// Arrange
	tuple := aliceMember()
	store := authz.NewInMemoryStore(tuple)

	// Act
	store.Delete(context.Background(), tuple)

	// Assert
	if got, _ := store.Has(context.Background(), tuple.Object, tuple.Relation, tuple.User); got {
		t.Errorf("Has after Delete = true, want false")
	}
}

func TestStore_GivenMissingTuple_WhenDeleted_ThenNoOp(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(aliceMember())

	// Act: deleting a tuple that was never written must not affect the store.
	store.Delete(context.Background(), bobViewer())

	// Assert
	if got, _ := store.FindAll(context.Background()); len(got) != 1 {
		t.Errorf("FindAll length = %d, want 1 (deleting a missing tuple is a no-op)", len(got))
	}
}

func TestStore_GivenMixedTuples_WhenFindByObjectRelation_ThenReturnsOnlyMatches(t *testing.T) {
	// Arrange
	match := bobViewer()
	store := authz.NewInMemoryStore(match, aliceMember())

	// Act
	got, _ := store.FindByObjectRelation(context.Background(), match.Object, match.Relation)

	// Assert
	if len(got) != 1 || got[0] != match {
		t.Errorf("FindByObjectRelation = %+v, want [%+v]", got, match)
	}
}

func TestStore_GivenFilter_WhenFindAll_ThenReturnsMatchingTuples(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore(aliceMember(), bobViewer())

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
			// Act
			got, _ := store.FindAll(context.Background(), tc.filter)

			// Assert
			if len(got) != tc.want {
				t.Errorf("FindAll(%+v) length = %d, want %d", tc.filter, len(got), tc.want)
			}
		})
	}
}

func TestStore_GivenTuples_WhenFindAll_ThenReturnsDeterministicOrder(t *testing.T) {
	// Arrange: write in reverse lexical order.
	store := authz.NewInMemoryStore(bobViewer(), aliceMember())

	// Act
	got, _ := store.FindAll(context.Background())

	// Assert: responses should not depend on Go's randomized map iteration order.
	want := []rebac.TupleKey{aliceMember(), bobViewer()}
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
			store.Write(context.Background(), rebac.Tuple(rebac.Team(id), rebac.RelationTeamMember, rebac.Subject(rebac.User("alice"))))
		}(i)
	}
	wg.Wait()

	// Assert
	if got, _ := store.FindAll(context.Background()); len(got) != n {
		t.Errorf("FindAll length = %d, want %d", len(got), n)
	}
}
