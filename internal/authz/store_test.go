package authz_test

import (
	"sync"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/rebac"
)

// These tests cover the in-memory relationship-store adapter. The store is a
// self-contained stateful unit with no collaborators, so no test doubles are
// needed: each test arranges real relationships, acts on the store, and asserts on its
// observable state.
//
// The store's methods take a context.Context and return an error to satisfy the
// port. Tests check those errors even though this in-memory implementation does
// not currently fail, keeping the examples safe to copy to real adapters.

func TestStore_GivenSeededRelationship_WhenHas_ThenReportsTrue(t *testing.T) {
	// Arrange
	relationship := rebac.NewRelationship(rebac.Subject(rebac.User("alice")), rebac.RelationTeamMember, rebac.Team("platformTeam"))
	store := authz.NewInMemoryStore(relationship)

	// Act
	got, err := store.Has(t.Context(), relationship.Subject, relationship.Relation, relationship.Resource)

	// Assert
	if err != nil {
		t.Fatalf("Has returned unexpected error: %v", err)
	}
	if !got {
		t.Errorf("Has(%+v) = false, want true", relationship)
	}
}

func TestStore_GivenEmptyStore_WhenHas_ThenReportsFalse(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore()
	relationship := rebac.NewRelationship(rebac.Subject(rebac.User("alice")), rebac.RelationTeamMember, rebac.Team("platformTeam"))

	// Act
	got, err := store.Has(t.Context(), relationship.Subject, relationship.Relation, relationship.Resource)

	// Assert
	if err != nil {
		t.Fatalf("Has returned unexpected error: %v", err)
	}
	if got {
		t.Errorf("Has on empty store = true, want false")
	}
}

func TestStore_GivenWrittenRelationship_WhenHas_ThenReportsTrue(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore()
	relationship := rebac.NewRelationship(rebac.Subject(rebac.User("alice")), rebac.RelationTeamMember, rebac.Team("platformTeam"))

	// Act
	if err := store.Write(t.Context(), relationship); err != nil {
		t.Fatalf("Write returned unexpected error: %v", err)
	}

	// Assert
	got, err := store.Has(t.Context(), relationship.Subject, relationship.Relation, relationship.Resource)
	if err != nil {
		t.Fatalf("Has returned unexpected error: %v", err)
	}
	if !got {
		t.Errorf("Has after Write = false, want true")
	}
}

func TestStore_GivenDuplicateWrites_WhenFindAll_ThenRelationshipStoredOnce(t *testing.T) {
	// Arrange
	store := authz.NewInMemoryStore()
	relationship := rebac.NewRelationship(rebac.Subject(rebac.User("alice")), rebac.RelationTeamMember, rebac.Team("platformTeam"))

	// Act: writing the same relationship twice must be idempotent.
	if err := store.Write(t.Context(), relationship); err != nil {
		t.Fatalf("first Write returned unexpected error: %v", err)
	}
	if err := store.Write(t.Context(), relationship); err != nil {
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

func TestStore_GivenStoredRelationship_WhenDeleted_ThenHasReportsFalse(t *testing.T) {
	// Arrange
	relationship := rebac.NewRelationship(rebac.Subject(rebac.User("alice")), rebac.RelationTeamMember, rebac.Team("platformTeam"))
	store := authz.NewInMemoryStore(relationship)

	// Act
	if err := store.Delete(t.Context(), relationship); err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	// Assert
	got, err := store.Has(t.Context(), relationship.Subject, relationship.Relation, relationship.Resource)
	if err != nil {
		t.Fatalf("Has returned unexpected error: %v", err)
	}
	if got {
		t.Errorf("Has after Delete = true, want false")
	}
}

func TestStore_GivenMissingRelationship_WhenDeleted_ThenNoOp(t *testing.T) {
	// Arrange
	aliceMember := rebac.NewRelationship(
		rebac.Subject(rebac.User("alice")),
		rebac.RelationTeamMember,
		rebac.Team("platformTeam"),
	)
	bobViewer := rebac.NewRelationship(
		rebac.Subject(rebac.User("bob")),
		rebac.RelationWorkspaceViewer,
		rebac.Workspace("productWorkspace"),
	)
	store := authz.NewInMemoryStore(aliceMember)

	// Act: deleting a relationship that was never written must not affect the store.
	if err := store.Delete(t.Context(), bobViewer); err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	// Assert
	got, err := store.FindAll(t.Context())
	if err != nil {
		t.Fatalf("FindAll returned unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("FindAll length = %d, want 1 (deleting a missing relationship is a no-op)", len(got))
	}
}

func TestStore_GivenMixedRelationships_WhenFindByResourceRelation_ThenReturnsOnlyMatches(t *testing.T) {
	// Arrange
	match := rebac.NewRelationship(rebac.Subject(rebac.User("bob")), rebac.RelationWorkspaceViewer, rebac.Workspace("productWorkspace"))
	nonMatch := rebac.NewRelationship(rebac.Subject(rebac.User("alice")), rebac.RelationTeamMember, rebac.Team("platformTeam"))
	store := authz.NewInMemoryStore(match, nonMatch)

	// Act
	got, err := store.FindByResourceRelation(t.Context(), match.Resource, match.Relation)

	// Assert
	if err != nil {
		t.Fatalf("FindByResourceRelation returned unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != match {
		t.Errorf("FindByResourceRelation = %+v, want [%+v]", got, match)
	}
}

func TestStore_GivenFilter_WhenFindAll_ThenReturnsMatchingRelationships(t *testing.T) {
	// Arrange
	cases := map[string]struct {
		filter authz.RelationshipFilter
		want   int
	}{
		"no filter matches all":        {authz.RelationshipFilter{}, 2},
		"by resource":                  {authz.RelationshipFilter{Resource: rebac.Team("platformTeam")}, 1},
		"by relation":                  {authz.RelationshipFilter{Relation: rebac.RelationWorkspaceViewer}, 1},
		"by resource and relation":     {authz.RelationshipFilter{Resource: rebac.Team("platformTeam"), Relation: rebac.RelationTeamMember}, 1},
		"non-matching filter is empty": {authz.RelationshipFilter{Resource: rebac.Team("noSuchTeam")}, 0},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Arrange: every subtest owns its store and seed data.
			aliceMember := rebac.NewRelationship(rebac.Subject(rebac.User("alice")), rebac.RelationTeamMember, rebac.Team("platformTeam"))
			bobViewer := rebac.NewRelationship(rebac.Subject(rebac.User("bob")), rebac.RelationWorkspaceViewer, rebac.Workspace("productWorkspace"))
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

func TestStore_GivenRelationships_WhenFindAll_ThenReturnsDeterministicOrder(t *testing.T) {
	// Arrange: write in reverse lexical order.
	aliceMember := rebac.NewRelationship(rebac.Subject(rebac.User("alice")), rebac.RelationTeamMember, rebac.Team("platformTeam"))
	bobViewer := rebac.NewRelationship(rebac.Subject(rebac.User("bob")), rebac.RelationWorkspaceViewer, rebac.Workspace("productWorkspace"))
	store := authz.NewInMemoryStore(bobViewer, aliceMember)

	// Act
	got, err := store.FindAll(t.Context())

	// Assert: responses should not depend on Go's randomized map iteration order.
	if err != nil {
		t.Fatalf("FindAll returned unexpected error: %v", err)
	}
	want := []rebac.Relationship{aliceMember, bobViewer}
	if len(got) != len(want) {
		t.Fatalf("FindAll length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("FindAll[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestStore_GivenConcurrentWrites_WhenFindAll_ThenAllRelationshipsStored(t *testing.T) {
	// Arrange: distinct relationships written from many goroutines. With -race this
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
			if err := store.Write(t.Context(), rebac.NewRelationship(rebac.Subject(rebac.User("alice")), rebac.RelationTeamMember, rebac.Team(id))); err != nil {
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
