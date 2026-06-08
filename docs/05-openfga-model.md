# OpenFGA Model

OpenFGA separates:

```text
store  -> environment namespace
model  -> schema: object types, relations, computed permissions
tuples -> runtime facts: who has what on what
```

The model for this repo lives in:

```text
deployments/openfga/model.fga
```

The Go in-process mirror lives in:

```text
internal/authz/model.go
```

## Types

The model contains:

```text
user
team
workspace
document
```

## Team

```text
type team
  relations
    define admin: [user]
    define member: [user] or admin
```

An admin is also a member.

## Workspace

```text
type workspace
  relations
    define owner: [user, team#admin]
    define editor: [user, team#member] or owner
    define viewer: [user, team#member] or editor
```

An owner is also an editor. An editor is also a viewer. A team subject set can
grant workspace access to everyone in that team relation.

## Document

```text
type document
  relations
    define workspace: [workspace]
    define owner: [user] or owner from workspace
    define editor: [user] or editor from workspace or owner
    define viewer: [user] or viewer from workspace or editor
    define can_read: viewer
    define can_comment: viewer
    define can_edit: editor
    define can_delete: owner
```

The important line shape is:

```text
editor from workspace
```

That means: follow the document's `workspace` relation to a workspace object,
then check whether the user is an editor there.

## Why It Matters

The model stores rules once. Tuples store facts many times.

You do not write a `can_read` tuple for every viewer. The model says viewers can
read, so the evaluator can derive `can_read` from `viewer`.

## Try It

Add a new permission:

```text
define archiver: [user] or owner
define can_archive: archiver
```

Then mirror it in `internal/rebac/rebac.go`, `internal/authz/model.go`, and
the evaluator tests.
