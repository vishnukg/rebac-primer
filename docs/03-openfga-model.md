# OpenFGA model

OpenFGA separates authorization into three layers:

- Store: environment namespace, usually one per dev/staging/prod environment
- Model: schema containing object types, relations, and computed permissions
- Tuples: runtime facts written by your application

The model lives in `src/authz/model.ts`.

```text
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
```

Important mechanics:

- `owner`, `editor`, and `viewer` are relationships.
- `can_read`, `can_edit`, and `can_delete` are permissions.
- `workspace#editor from workspace` means follow the document's `workspace`
  relation, then check whether the user is an editor of that workspace.
- `team#member` is a subject set, not a user. OpenFGA resolves it at check time.

This is the reason ReBAC scales better than copying document permissions to
every user. Team membership changes in one place and every affected check sees
the new graph.
