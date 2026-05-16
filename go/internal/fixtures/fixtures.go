// Package fixtures contains the shared test data used by graph authorizer tests,
// service tests, and handler tests.
//
// The tuples model this access scenario:
//
//	workspaceEditor → member of platformTeam
//	platformTeam#member → editor of productWorkspace
//	workspaceViewer → viewer of productWorkspace
//	roadmapDocument → lives in productWorkspace
//
// From these four tuples, the graph authorizer can derive:
//
//	workspaceEditor can_edit roadmapDocument  (via team → workspace editor → document)
//	workspaceViewer can_read roadmapDocument  (via workspace viewer → document viewer → can_read)
//	outsideCollaborator cannot access roadmapDocument (no path in the graph)
package fixtures

import (
	"rebac-primer/internal/authz"
)

// Named objects — use these in tests instead of raw strings.
var (
	WorkspaceEditor     = authz.User("workspaceEditor")
	WorkspaceViewer     = authz.User("workspaceViewer")
	OutsideCollaborator = authz.User("outsideCollaborator")
	PlatformTeam        = authz.Team("platformTeam")
	ProductWorkspace    = authz.Workspace("productWorkspace")
	RoadmapDocument     = authz.Document("roadmapDocument")
)

// SeedRelationshipTuples returns the four base tuples for the demo scenario.
func SeedRelationshipTuples() []authz.TupleKey {
	return []authz.TupleKey{
		// workspaceEditor is a member of platformTeam
		authz.Tuple(PlatformTeam, authz.RelationTeamMember, authz.Subject(WorkspaceEditor)),
		// platformTeam#member are editors of productWorkspace
		authz.Tuple(ProductWorkspace, authz.RelationWorkspaceEditor, authz.SubjectSet(PlatformTeam, authz.RelationTeamMember)),
		// workspaceViewer is a viewer of productWorkspace
		authz.Tuple(ProductWorkspace, authz.RelationWorkspaceViewer, authz.Subject(WorkspaceViewer)),
		// roadmapDocument lives in productWorkspace
		authz.Tuple(RoadmapDocument, authz.RelationDocumentWorkspace, authz.Subject(ProductWorkspace)),
	}
}
