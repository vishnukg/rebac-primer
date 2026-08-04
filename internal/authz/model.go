package authz

import "rebac-primer/internal/rebac"

// This file defines the relation hierarchy for each resource type.
//
// # What this file is for
//
// These tables answer one question: "if a subject has relation X on a resource,
// which weaker relations do they automatically also have on that resource?"
//
// For example: a workspace editor can also do everything a viewer can do.
// Rather than writing that rule into every relationship, we declare it once here:
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
// (via the first rule), so owner ⊆ editor ⊆ viewer as sets of subjects.
//
// # Why this is separate from relationships
//
// Relationships store runtime facts: "alice is an editor of productWorkspace".
// These rules store the schema: "editors are also viewers".
//
// Mixing them would mean writing a separate "viewer" relationship for every
// subject who is already an editor — duplicating data that is really a schema
// rule.
// Keeping them apart is the same split OpenFGA makes between its DSL model and
// its relationship store.

// impliedBy maps a relation to the stronger relations that satisfy it.
// "Key relation is implied by any of the value relations."
type impliedBy map[rebac.Relation][]rebac.Relation

// actionRules maps each application action to the base relation that
// satisfies it. Actions are policy results; relations name the stored
// associations and derived role-like sets used to prove those results.
var actionRules = map[rebac.ResourceType]map[rebac.Action][]rebac.Relation{
	rebac.ResourceTypeWorkspace: {
		rebac.ActionWorkspaceCreateDocument: {rebac.RelationWorkspaceEditor},
	},
	rebac.ResourceTypeDocument: {
		rebac.ActionDocumentRead:    {rebac.RelationDocumentViewer},
		rebac.ActionDocumentComment: {rebac.RelationDocumentViewer},
		rebac.ActionDocumentEdit:    {rebac.RelationDocumentEditor},
		rebac.ActionDocumentDelete:  {rebac.RelationDocumentOwner},
	},
}

func actionRelationsFor(
	resourceType rebac.ResourceType,
	action rebac.Action,
) []rebac.Relation {
	return actionRules[resourceType][action]
}

// teamRules — the team relation hierarchy.
//
// The team type has two relations:
//
//	admin  — full control over the team
//	member — read/participate access
//
// Hierarchy: admin ⊆ member as sets of subjects
//
//	(user:alice, member, team:platform) satisfies team membership directly.
//	(user:alice, admin, team:platform) also satisfies membership via this rule.
//
// In OpenFGA DSL this is:
//
//	type team
//	  relations
//	    define admin:  [user]
//	    define member: [user] or admin
var teamRules = impliedBy{
	// "member" is satisfied by anyone who has "admin" on the same team.
	rebac.RelationTeamMember: {rebac.RelationTeamAdmin},
}

// workspaceRules — the workspace relation hierarchy.
//
// The workspace type has three relations:
//
//	owner  — can manage the workspace and all its content
//	editor — can create and edit content
//	viewer — can read content
//
// Hierarchy: owner ⊆ editor ⊆ viewer as sets of subjects
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
	rebac.RelationWorkspaceEditor: {rebac.RelationWorkspaceOwner},
	// "viewer" is satisfied by anyone who has "editor" (or, transitively, "owner").
	rebac.RelationWorkspaceViewer: {rebac.RelationWorkspaceEditor},
}

// documentRules — the document relation hierarchy.
//
// Documents have stored base and structural relations:
//
//	Base relations — stored in relationships, can be granted directly or inherited
//	  from the parent workspace (see expandDocument in evaluator.go):
//	    owner, editor, viewer
//
// Hierarchy of base relations: owner ⊆ editor ⊆ viewer as sets of subjects
//
// Actions are deliberately a separate domain type. Evaluate maps each
// action to its required base relation; ValidateCheckRequest uses the same
// mapping to reject actions that are not valid for the resource type:
//
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
//	    define owner:       [user] or owner from workspace
//	    define editor:      [user] or editor from workspace or owner
//	    define viewer:      [user] or viewer from workspace or editor
//	    define can_read:    viewer
//	    define can_comment: viewer
//	    define can_edit:    editor
//	    define can_delete:  owner
//
// Note: the "from workspace" part of the DSL (workspace inheritance) is handled
// in expandDocument in evaluator.go, not in this table.  These rules only cover
// the same-resource hierarchy.
var documentRules = impliedBy{
	// Base role hierarchy:
	rebac.RelationDocumentViewer: {rebac.RelationDocumentEditor},
	rebac.RelationDocumentEditor: {rebac.RelationDocumentOwner},
}
