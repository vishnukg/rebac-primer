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

## Modeling Thought Process

The model is structured to answer the demo product requirements with the
fewest durable facts:

```text
Team admins are also team members.
Workspace owners are also editors and viewers.
Teams can receive workspace access as a group.
Documents inherit owner/editor/viewer from their workspace.
Application code asks for can_read/can_comment/can_edit/can_delete.
```

The design process is:

1. Choose object types that exist in the product: `user`, `team`, `workspace`,
   `document`.
2. Identify durable facts to store as tuples: team membership, team access to a
   workspace, direct workspace/document ownership, and a document's parent
   workspace.
3. Identify derived relationships: admin implies member, owner implies editor,
   editor implies viewer, document access can come from the parent workspace.
4. Identify application permissions: read, comment, edit, delete.
5. Keep action permissions as computed relations so callers ask for intent
   (`can_edit`) rather than implementation detail (`editor from workspace`).
6. Write contract tests before trusting the model.

That gives this rule of thumb:

```text
Facts that product workflows mutate go in tuples.
Rules that explain what facts mean go in the model.
Operations that code enforces become can_* permissions.
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

Why: team membership is a group fact, and `admin` is a stronger team relation.
If someone administers a team, they should also satisfy checks that only require
team membership. The model captures that once with `member: [user] or admin`
instead of writing both `admin` and `member` tuples for every admin.

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

Why: workspace permissions are role-like, but scoped to one workspace. Direct
users can be owners/editors/viewers, and teams can be granted access through
subject sets:

```text
team:platformTeam#member editor workspace:productWorkspace
```

That single tuple means current and future platform-team members are workspace
editors. Adding a user to the team updates inherited workspace access without
rewriting workspace tuples.

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

`workspace` is a structural relation, not a user permission. The application
writes `document#workspace` tuples so inheritance can work, but user-facing
permission checks ask for relations such as `can_read`, `can_edit`, `owner`,
`editor`, or `viewer`.

Why: documents are children of workspaces. The document's `workspace` tuple is
the bridge that lets a workspace relationship affect a document:

```text
workspace:productWorkspace workspace document:roadmapDocument
```

Then `editor from workspace` says: "a document editor includes anyone who is an
editor of the workspace this document points to." The parent object must be in
the tuple's subject/user field because `from workspace` follows the document's
`workspace` relation to that subject.

The `can_*` relations are action permissions. They are not writable facts:

```text
can_read    = viewer
can_comment = viewer
can_edit    = editor
can_delete  = owner
```

This lets application code ask stable business questions while the model remains
free to change how those permissions are derived.

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

Read each line of `model.fga` as answering one of three questions:

```text
What can be written directly?       [user], [workspace], [team#member]
What is derived on the same object? viewer includes editor, can_edit is editor
What is inherited from a parent?    editor from workspace
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
