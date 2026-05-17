// Package fixtures contains the shared test data used by graph authorizer tests,
// service tests, and handler tests.
//
// The tuples model this access scenario:
//
//	Alice → member of platformTeam
//	platformTeam#member → editor of productWorkspace
//	Bob → viewer of productWorkspace
//	roadmapDocument → lives in productWorkspace
//
// From these four tuples, the graph authorizer can derive:
//
//	Alice can_edit roadmapDocument  (via team → workspace editor → document)
//	Bob can_read roadmapDocument  (via workspace viewer → document viewer → can_read)
//	Casey cannot access roadmapDocument (no path in the graph)
package fixtures

import (
	"rebac-primer/internal/authz"
)

// Named objects — use these in tests instead of raw strings.
var (
	Alice = authz.User("alice")
	Bob   = authz.User("bob")
	Casey = authz.User("casey")

	PlatformTeam     = authz.Team("platformTeam")
	ProductWorkspace = authz.Workspace("productWorkspace")
	RoadmapDocument  = authz.Document("roadmapDocument")
)

// SeedRelationshipTuples returns the four base tuples for the demo scenario.
func SeedRelationshipTuples() []authz.TupleKey {
	return []authz.TupleKey{
		// Alice is a member of platformTeam
		authz.Tuple(PlatformTeam, authz.RelationTeamMember, authz.Subject(Alice)),
		// platformTeam#member are editors of productWorkspace
		authz.Tuple(ProductWorkspace, authz.RelationWorkspaceEditor, authz.SubjectSet(PlatformTeam, authz.RelationTeamMember)),
		// Bob is a viewer of productWorkspace
		authz.Tuple(ProductWorkspace, authz.RelationWorkspaceViewer, authz.Subject(Bob)),
		// roadmapDocument lives in productWorkspace
		authz.Tuple(RoadmapDocument, authz.RelationDocumentWorkspace, authz.Subject(ProductWorkspace)),
	}
}
