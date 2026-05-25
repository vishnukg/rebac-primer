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
	"rebac-primer/internal/documents/adapters/authn"
	"rebac-primer/internal/shared"
)

// Named objects — use these in tests instead of raw strings.
var (
	Alice = shared.User("alice")
	Bob   = shared.User("bob")
	Casey = shared.User("casey")

	PlatformTeam     = shared.Team("platformTeam")
	ProductWorkspace = shared.Workspace("productWorkspace")
	RoadmapDocument  = shared.Document("roadmapDocument")
)

// DemoTokens maps demo bearer tokens to their claims.
// Mirrors typescript/src/demo/fixtures.ts demoTokens.
func DemoTokens() map[string]authn.TokenClaims {
	return map[string]authn.TokenClaims{
		"demo-token-alice": {Sub: "alice", Scopes: []string{"documents:read", "documents:write"}},
		"demo-token-bob":   {Sub: "bob", Scopes: []string{"documents:read"}},
		"demo-token-casey": {Sub: "casey", Scopes: []string{"documents:read"}},
	}
}

// SeedRelationshipTuples returns the four base tuples for the demo scenario.
func SeedRelationshipTuples() []shared.TupleKey {
	return []shared.TupleKey{
		// Alice is a member of platformTeam
		shared.Tuple(PlatformTeam, shared.RelationTeamMember, shared.Subject(Alice)),
		// platformTeam#member are editors of productWorkspace
		shared.Tuple(ProductWorkspace, shared.RelationWorkspaceEditor, shared.SubjectSet(PlatformTeam, shared.RelationTeamMember)),
		// Bob is a viewer of productWorkspace
		shared.Tuple(ProductWorkspace, shared.RelationWorkspaceViewer, shared.Subject(Bob)),
		// roadmapDocument lives in productWorkspace
		shared.Tuple(RoadmapDocument, shared.RelationDocumentWorkspace, shared.Subject(ProductWorkspace)),
	}
}
