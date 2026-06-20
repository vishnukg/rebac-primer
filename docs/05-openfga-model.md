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

Read the model as a type system plus set algebra. Type restrictions say what may
be written directly; relation expressions say what may be derived.

Each `object#relation` can be understood as a set of subjects. A Check asks
whether one subject belongs to the effective set for the requested relation.

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

This is the central schema/data split:

```text
relationship tuples  → changing product facts
authorization model  → reusable rules for deriving effective relationships
```

The DSL constructs used here are:

```text
[user]                 direct assignment with a type restriction
can_edit: editor       computed relation on the same object
editor from workspace  inheritance through a related object
or                     union of subject sets
```

OpenFGA also supports intersections, exclusions, conditions, contextual tuples,
and query APIs that the teaching evaluator does not implement.

## Try It

Add a new computed permission:

```text
define can_archive: owner
```

To keep both backends aligned, update:

1. `deployments/openfga/model.fga`
2. the relation constant in `internal/rebac/rebac.go`
3. `documentRules` in `internal/authz/model.go`
4. `relationDefinedFor` and computed-relation validation in
   `internal/authz/validate.go`
5. the shared authorization contract and evaluator tests

The traversal algorithm itself should not change. If adding a simple permission
requires editing DFS code, the model and evaluator are becoming too tightly
coupled.

## Checkpoint

Why is `can_edit` not stored as a tuple? Because it is a computed permission:
the model derives it from `editor`, while tuples store changing relationship
facts.

Next: [Designing a ReBAC authorization service](07-rebac-authorization-service-design.md)
turns this model into a production design and compares adopting OpenFGA with
building the engine yourself.
