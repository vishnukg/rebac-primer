// Package fixtures contains the shared test data used across both services.
//
// The tuples model this access scenario:
//
//	Alice → member of platformTeam
//	platformTeam#member → editor of productWorkspace
//	Bob → viewer of productWorkspace
//	roadmapDocument → lives in productWorkspace
//
// From these four tuples, the graph evaluator can derive:
//
//	Alice can_edit roadmapDocument  (via team → workspace editor → document)
//	Bob can_read roadmapDocument    (via workspace viewer → document viewer → can_read)
//	Casey cannot access roadmapDocument (no path in the graph)
package fixtures

import (
	"rebac-primer/internal/documents"
	"rebac-primer/internal/rebac"
)

// Named objects — use these in tests instead of raw strings.
var (
	Alice = rebac.User("alice")
	Bob   = rebac.User("bob")
	Casey = rebac.User("casey")

	PlatformTeam     = rebac.Team("platformTeam")
	ProductWorkspace = rebac.Workspace("productWorkspace")
	RoadmapDocument  = rebac.Document("roadmapDocument")
)

// DemoTokens maps demo bearer tokens to their claims.
func DemoTokens() map[string]documents.TokenClaims {
	return map[string]documents.TokenClaims{
		"demo-token-alice": {Sub: "alice", Scopes: []string{"documents:read", "documents:write"}},
		"demo-token-bob":   {Sub: "bob", Scopes: []string{"documents:read"}},
		"demo-token-casey": {Sub: "casey", Scopes: []string{"documents:read"}},
	}
}

// SeedRelationshipTuples returns the four base tuples for the demo scenario.
func SeedRelationshipTuples() []rebac.TupleKey {
	return []rebac.TupleKey{
		// Alice is a member of platformTeam
		rebac.Tuple(PlatformTeam, rebac.RelationTeamMember, rebac.Subject(Alice)),
		// platformTeam#member are editors of productWorkspace
		rebac.Tuple(ProductWorkspace, rebac.RelationWorkspaceEditor, rebac.SubjectSet(PlatformTeam, rebac.RelationTeamMember)),
		// Bob is a viewer of productWorkspace
		rebac.Tuple(ProductWorkspace, rebac.RelationWorkspaceViewer, rebac.Subject(Bob)),
		// roadmapDocument lives in productWorkspace
		rebac.Tuple(RoadmapDocument, rebac.RelationDocumentWorkspace, rebac.Subject(ProductWorkspace)),
	}
}
