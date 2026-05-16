# OpenFGA model

OpenFGA separates authorization into three layers:

```text
store  -> environment namespace
model  -> schema: object types, relations, computed permissions
tuples -> runtime facts: who has what on what
```

This separation is one of the best ideas in OpenFGA.

Your model should change rarely. Your tuples change constantly.

## Scene

The product rule sounds simple: workspace editors can edit workspace documents.
The model is where that sentence becomes executable. If the model is clear, the
application code can stay boring.

## The model in this repo

Open `src/authz/model.ts`.

The model contains four types:

```text
user
team
workspace
document
```

Read those as the nouns in the system.

## Users

```text
type user
```

The user type has no relations. It is a leaf subject. Users get access by being
related to teams, workspaces, and documents.

## Teams

```text
type team
  relations
    define member: [user]
    define admin: [user] or member
```

This says:

- only users can be direct team members
- users can be direct admins
- members are also included in admin for this simplified tutorial model

In a production model, you may choose the opposite hierarchy:

```text
member: [user] or admin
admin: [user]
```

That would mean admins are members, but members are not admins. The exact shape
depends on the product rules. The important lesson is that relation definitions
encode hierarchy.

## Workspaces

```text
type workspace
  relations
    define owner: [user, team#admin]
    define editor: [user, team#member] or owner
    define viewer: [user, team#member] or editor
```

This creates a hierarchy:

```text
owner -> editor -> viewer
```

Owners can do everything editors can do. Editors can do everything viewers can
do.

The type restrictions matter:

```text
[user, team#member]
```

This says you can write direct users or team member subject sets as workspace
editors.

## Documents

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

This is the most important part of the model.

A document can have direct owners, editors, and viewers. It can also inherit
access from its parent workspace.

## The `from` keyword

This line deserves attention:

```text
define editor: [user] or workspace#editor from workspace or owner
```

Read it as:

```text
A document editor is:
  a direct user editor, OR
  someone who is an editor of the document's workspace, OR
  a document owner
```

The phrase:

```text
workspace#editor from workspace
```

means:

1. find the document's `workspace` relation
2. follow it to a workspace object
3. check whether the user has `editor` on that workspace

That is graph traversal.

## Relationship hierarchy

For documents:

```text
owner -> editor -> viewer

viewer -> can_read
viewer -> can_comment
editor -> can_edit
owner  -> can_delete
```

So if Alice is a workspace editor, and the roadmap document belongs to that
workspace:

```text
Alice can edit the roadmap.
Alice can read the roadmap.
Alice can comment on the roadmap.
Alice cannot delete the roadmap unless she is also an owner.
```

This is the kind of rule you want in the model, not scattered across handlers.

## Model design habit

Start with the product sentence:

```text
Workspace editors can edit documents in that workspace.
```

Then write the graph sentence:

```text
document editor includes workspace editor from workspace
```

Then write the OpenFGA DSL:

```text
define editor: [user] or workspace#editor from workspace or owner
```

Do not start by typing DSL. Start with the rule.

## Tuple examples

These tuples are enough to grant Alice edit access:

```text
(team:platform, member, user:alice)
(workspace:acme, editor, team:platform#member)
(document:roadmap, workspace, workspace:acme)
```

No tuple says:

```text
(document:roadmap, editor, user:alice)
```

That is the point. The access is inherited.

## Debugging model mistakes

When a check surprises you, ask:

1. Is the object id correct?
2. Is the relation name correct?
3. Does a tuple exist for the first edge?
4. Does a subject set need to be resolved?
5. Does a `from` relation point at the expected parent?
6. Did the model define the permission in terms of the right relationship?

Then write the path in plain English.

If the English path is confusing, the model probably is too.

## Exercise

Add an `archiver` relationship to documents:

```text
define archiver: [user] or owner
define can_archive: archiver
```

Then mirror that in TypeScript:

1. add `"archiver"` and `"can_archive"` to `DocumentRelation`
2. update `GraphAuthorizer`
3. add tests for owner allowed and viewer denied

This exercise forces the OpenFGA model and TypeScript vocabulary to stay in
sync.

## Checkpoint

Read this line out loud:

```text
define editor: [user] or workspace#editor from workspace or owner
```

If you can explain it as a graph path, you understand the model. If you cannot,
do not add more permissions yet.
