package graph

import "rebac-primer/internal/shared"

// permissionmodel.go defines the permission hierarchy for each object type.
//
// # What this file is for
//
// These tables answer one question: "if a user has relation X on an object,
// which weaker relations do they automatically also have on that same object?"
//
// For example: a workspace editor can also do everything a viewer can do.
// Rather than writing that rule into every tuple, we declare it once here:
//
//	workspaceRules["viewer"] = ["editor"]
//	→ "the viewer relation is satisfied by anyone who has the editor relation"
//
// The graph evaluator reads these tables in expandByRules and recursively
// checks the stronger relation when the weaker one is not found directly.
//
// # Reading the tables
//
// Each entry reads: "the key relation is satisfied by any of the value relations."
//
//	rules["viewer"] = ["editor"]     → viewer is satisfied by editor
//	rules["editor"] = ["owner"]      → editor is satisfied by owner
//
// Chained: owner satisfies editor (via the second rule) which satisfies viewer
// (via the first rule), so owner ⊇ editor ⊇ viewer.
//
// # Why this is separate from tuples
//
// Tuples store runtime facts: "alice is an editor of productWorkspace".
// These rules store the schema: "editors are also viewers".
//
// Mixing them would mean writing a separate "viewer" tuple for every user who
// is already an editor — duplicating data that is really a schema rule.
// Keeping them apart is the same split OpenFGA makes between its DSL model and
// its tuple store.
//
// Mirrors typescript/src/authz-service/adapters/graph/permissionModel.ts.

// impliedBy maps a relation to the stronger relations that satisfy it.
// "Key relation is implied by any of the value relations."
type impliedBy map[shared.Relation][]shared.Relation

// teamRules — the team permission hierarchy.
//
// The team type has two relations:
//
//	admin  — full control over the team
//	member — read/participate access
//
// Hierarchy: admin ⊇ member
//
//	(team, member, user:alice) satisfies "is alice a team member?" — direct.
//	(team, admin,  user:alice) also satisfies "is alice a team member?" — via this rule.
//
// In OpenFGA DSL this is:
//
//	type team
//	  relations
//	    define admin:  [user]
//	    define member: [user] or admin
var teamRules = impliedBy{
	// "member" is satisfied by anyone who has "admin" on the same team.
	shared.RelationTeamMember: {shared.RelationTeamAdmin},
}

// workspaceRules — the workspace permission hierarchy.
//
// The workspace type has three relations:
//
//	owner  — can manage the workspace and all its content
//	editor — can create and edit content
//	viewer — can read content
//
// Hierarchy: owner ⊇ editor ⊇ viewer
//
// In OpenFGA DSL:
//
//	type workspace
//	  relations
//	    define owner:  [user, team#admin]
//	    define editor: [user, team#member] or owner
//	    define viewer: [user, team#member] or editor
var workspaceRules = impliedBy{
	// "editor" is satisfied by anyone who has "owner" on the same workspace.
	shared.RelationWorkspaceEditor: {shared.RelationWorkspaceOwner},
	// "viewer" is satisfied by anyone who has "editor" (or, transitively, "owner").
	shared.RelationWorkspaceViewer: {shared.RelationWorkspaceEditor},
}

// documentRules — the document permission hierarchy.
//
// Documents have two kinds of relations:
//
//	Base relations — stored in tuples, can be granted directly or inherited
//	  from the parent workspace (see expandDocument in evaluator.go):
//	    owner, editor, viewer
//
//	Computed permissions — derived from base relations by these rules, never
//	  stored in tuples:
//	    can_read, can_comment, can_edit, can_delete
//
// Hierarchy of base relations: owner ⊇ editor ⊇ viewer
//
// Derived permissions:
//	can_read    ← viewer (and therefore editor and owner)
//	can_comment ← viewer (and therefore editor and owner)
//	can_edit    ← editor (and therefore owner)
//	can_delete  ← owner only
//
// In OpenFGA DSL:
//
//	type document
//	  relations
//	    define workspace:   [workspace]
//	    define owner:       [user] or workspace#owner from workspace
//	    define editor:      [user] or workspace#editor from workspace or owner
//	    define viewer:      [user] or workspace#viewer from workspace or editor
//	    define can_read:    viewer
//	    define can_comment: viewer
//	    define can_edit:    editor
//	    define can_delete:  owner
//
// Note: the "from workspace" part of the DSL (workspace inheritance) is handled
// in expandDocument in evaluator.go, not in this table.  These rules only cover
// the same-object hierarchy.
var documentRules = impliedBy{
	// Computed permissions resolved from base roles:
	shared.RelationDocumentCanRead:    {shared.RelationDocumentViewer},
	shared.RelationDocumentCanComment: {shared.RelationDocumentViewer},
	shared.RelationDocumentCanEdit:    {shared.RelationDocumentEditor},
	shared.RelationDocumentCanDelete:  {shared.RelationDocumentOwner},

	// Base role hierarchy:
	shared.RelationDocumentViewer: {shared.RelationDocumentEditor},
	shared.RelationDocumentEditor: {shared.RelationDocumentOwner},
}
