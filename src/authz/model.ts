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
  "team:platform#member contains user:alice",
  "workspace:acme#editor contains team:platform#member",
  "document:roadmap#workspace points at workspace:acme",
  "therefore user:alice can_edit document:roadmap"
] as const;
