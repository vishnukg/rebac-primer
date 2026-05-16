export const openFgaModel = `
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
`;

export const relationshipGraphExample = [
  "team:platformTeam#member contains user:workspaceEditor",
  "workspace:productWorkspace#editor contains team:platformTeam#member",
  "document:roadmapDocument#workspace points at workspace:productWorkspace",
  "therefore user:workspaceEditor can_edit document:roadmapDocument"
] as const;
