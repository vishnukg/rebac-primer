package authz

// OpenFGAModel is the authorization model in OpenFGA DSL format.
// It mirrors the TypeScript openFgaModel constant in typescript/src/authz/model.ts.
//
// This model drives both the graph authorizer (which implements its rules in Go)
// and the OpenFGA adapter (which uploads it to an OpenFGA server).
const OpenFGAModel = `
model
  schema 1.1

type user

type team
  relations
    define admin: [user]
    define member: [user] or admin

type workspace
  relations
    define owner: [user, team#admin]
    define editor: [user, team#member] or owner
    define viewer: [user, team#member] or editor

type document
  relations
    define workspace: [workspace]
    define owner: [user] or workspace#owner from workspace
    define editor: [user] or workspace#editor from workspace or owner
    define viewer: [user] or workspace#viewer from workspace or editor
    define can_read: viewer
    define can_comment: viewer
    define can_edit: editor
    define can_delete: owner
`

// RelationshipGraphExample illustrates the tuple chain that grants
// workspaceEditor can_edit access to roadmapDocument.
var RelationshipGraphExample = []string{
	"team:platformTeam#member contains user:workspaceEditor",
	"workspace:productWorkspace#editor contains team:platformTeam#member",
	"document:roadmapDocument#workspace points at workspace:productWorkspace",
	"therefore user:workspaceEditor can_edit document:roadmapDocument",
}
