// Package fixtures contains the shared test data used across both services.
//
// The relationships model this access scenario:
//
//	user:alice → member of team:platformTeam
//	team:platformTeam#member → editor of workspace:productWorkspace
//	user:bob → viewer of workspace:productWorkspace
//	workspace:productWorkspace → workspace of document:roadmapDocument
//
// From these four relationships, the graph evaluator can derive:
//
//	Alice can_edit roadmapDocument  (via team → workspace editor → document)
//	Bob can_read roadmapDocument    (via workspace viewer → document viewer → can_read)
//	Casey cannot access roadmapDocument (no path in the graph)
package fixtures

import (
	"rebac-primer/internal/documents"
	"rebac-primer/internal/rebac"
)

// Named resources — use these in tests instead of raw strings.
var (
	Alice = rebac.User("alice")
	Bob   = rebac.User("bob")
	Casey = rebac.User("casey")
	Dana  = rebac.User("dana")
	Erin  = rebac.User("erin")

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

// SeedRelationships returns the four base relationships for the demo scenario.
func SeedRelationships() []rebac.Relationship {
	return []rebac.Relationship{
		// Alice is a member of platformTeam
		rebac.NewRelationship(rebac.Subject(Alice), rebac.RelationTeamMember, PlatformTeam),
		// platformTeam#member are editors of productWorkspace
		rebac.NewRelationship(
			rebac.SubjectSet(PlatformTeam, rebac.RelationTeamMember),
			rebac.RelationWorkspaceEditor,
			ProductWorkspace,
		),
		// Bob is a viewer of productWorkspace
		rebac.NewRelationship(rebac.Subject(Bob), rebac.RelationWorkspaceViewer, ProductWorkspace),
		// roadmapDocument lives in productWorkspace
		rebac.NewRelationship(
			rebac.Subject(ProductWorkspace),
			rebac.RelationDocumentWorkspace,
			RoadmapDocument,
		),
	}
}
