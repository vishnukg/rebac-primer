package graph

import "rebac-primer/internal/shared"

// permissionmodel.go defines which relations are implied by others on the SAME
// object.  It mirrors typescript/src/authz-service/adapters/graph/permissionModel.ts.
//
// Reading the tables: key = relation being checked, value = stronger relations
// that satisfy it.
//
//	workspaceRules[viewer] = [editor]
//	→ "workspace viewer is satisfied by workspace editor"
//
// These are package-level vars allocated once, not on every call.

// impliedBy maps a relation to the set of stronger relations that imply it.
type impliedBy map[shared.Relation][]shared.Relation

// teamRules: team.admin implies team.member
var teamRules = impliedBy{
	shared.RelationTeamMember: {shared.RelationTeamAdmin},
}

// workspaceRules: workspace.owner ⊇ editor ⊇ viewer
var workspaceRules = impliedBy{
	shared.RelationWorkspaceEditor: {shared.RelationWorkspaceOwner},
	shared.RelationWorkspaceViewer: {shared.RelationWorkspaceEditor},
}

// documentRules: document role hierarchy and computed permissions.
// Note: owner/editor/viewer can also be inherited from the parent workspace —
// that logic lives in evaluator.go because it requires a tuple lookup.
var documentRules = impliedBy{
	shared.RelationDocumentCanRead:    {shared.RelationDocumentViewer},
	shared.RelationDocumentCanComment: {shared.RelationDocumentViewer},
	shared.RelationDocumentCanEdit:    {shared.RelationDocumentEditor},
	shared.RelationDocumentCanDelete:  {shared.RelationDocumentOwner},
	shared.RelationDocumentViewer:     {shared.RelationDocumentEditor},
	shared.RelationDocumentEditor:     {shared.RelationDocumentOwner},
}
