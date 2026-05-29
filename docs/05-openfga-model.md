# OpenFGA model

Imagine the product manager walks over and says: "Alice should be able to edit
the roadmap doc because she's on the platform team." You could code that rule
directly into `documents.update`. But then the next request — "engineers
on this workspace can comment but not edit" — needs another `if`. And the one
after that needs another. Soon the rules are scattered across handlers and
nobody can answer "who can edit this document, and why?"

OpenFGA replaces that pile of `if`s with three layers you can reason about
independently:

```text
store  -> environment namespace
model  -> schema: object types, relations, computed permissions
tuples -> runtime facts: who has what on what
```

The split matters because each layer changes on a different clock. Your model
should change rarely (the *shape* of the rules). Your tuples change constantly
(the *data*). Your store almost never changes (one per environment).

Reference: [OpenFGA concepts](https://openfga.dev/docs/concepts) explains the
same split between authorization models, relationship tuples, users, objects,
and checks.

## Scene

The product rule sounds simple: workspace editors can edit workspace documents.
The model is where that sentence becomes executable. If the model is clear, the
application code can stay boring.

## The model in this repo

The model is defined in both implementations — the DSL is identical:

- TypeScript: `typescript/src/authz-service/adapters/graph/permissionModel.ts`
- Go: `go/internal/authz/adapters/graph/permissionmodel.go`

Open either one.

The model contains four types:

```text
user
team
workspace
document
```

Read those as the nouns in the system.

Architecture diagram:

```text
┌─────────────────────────────────────────────┐
│ OpenFGA Store                               │
│                                             │
│  ┌──────────────┐     ┌──────────────────┐  │
│  │ Model        │     │ Tuples           │  │
│  │ schema       │     │ runtime facts    │  │
│  │ rarely       │     │ change often     │  │
│  │ changes      │     │                  │  │
│  └──────┬───────┘     └────────┬─────────┘  │
│         │                      │            │
│         └──────────┬───────────┘            │
│                    ▼                        │
│              Check evaluation               │
└─────────────────────────────────────────────┘
```

Model and tuples are separate on purpose:

```text
Model:  what relationships can mean
Tuples: which relationships currently exist
```

## Reading the DSL

Before walking through each type, here are the three syntax constructs you will
see throughout the model. Everything is built from these three ideas.

**Type restriction** — square brackets declare which subject types can be
assigned directly to a relation:

```text
define admin: [user]
```

Only a `user:someone` is valid as a direct admin. Writing `team:platformTeam` as
an admin would be rejected by the model.

**Subject set** — the `type#relation` form means "everyone who holds that
relation on that type":

```text
define editor: [user, team#member]
```

Both a literal `user:alice` and a `team:platformTeam#member` (all members of
that team) are valid workspace editors. One tuple grants access to an entire
team.

**Graph traversal** — the `X from Y` form means "follow relation Y to a parent
object, then check X there":

```text
define editor: editor from workspace
```

Find the object this document points to via its `workspace` relation, then check
whether the user is an `editor` on that workspace. This is how permissions flow
from parent to child.

These three constructs are worth re-reading when any DSL line looks confusing.
Most lines are just combinations of them.

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
    define admin: [user]
    define member: [user] or admin
```

The square brackets are **type restrictions**. They declare which subject types
can appear as direct values in a tuple for this relation.

`[user]` means only a `user:someone` value is valid. You cannot write a
`team:platformTeam` directly as a team admin — the model rejects it.

This says:

- only users can be direct team admins
- users can be direct team members
- admins are also members

Do not reverse this hierarchy unless the product truly means it.

```text
admin: [user] or member
```

That would mean every member is an admin, which is usually too powerful. The
important lesson is that relation definitions encode hierarchy, and hierarchy
direction matters.

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

This says two kinds of subjects are valid as direct workspace editors:

- a `user:someone` literal
- a `team:someTeam#member` subject set (everyone who is a member of that team)

The `team#member` form is what makes one tuple grant access to an entire team.
Without it, you would have to write one tuple per user.

## Documents

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

This is the most important part of the model.

A document can have direct owners, editors, and viewers. It can also inherit
access from its parent workspace.

## The `from` keyword

This line deserves attention:

```text
define editor: [user] or editor from workspace or owner
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
editor from workspace
```

means:

1. find the document's `workspace` relation
2. follow it to a workspace object
3. check whether the user has `editor` on that workspace

That is graph traversal.

Diagram:

```text
document:roadmapDocument
      │
      │ workspace
      ▼
workspace:productWorkspace
      │
      │ editor
      ▼
team:platformTeam#member
      │
      │ member
      ▼
user:alice
```

The `from` keyword is what lets document access flow from the parent workspace.

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
Alice can edit the roadmap document.
Alice can read the roadmap document.
Alice can comment on the roadmap document.
Alice cannot delete the roadmap document unless she is also an owner.
```

This is the kind of rule you want in the model, not scattered across handlers.

Permission graph:

```text
owner
  ├── can_delete
  └── editor
        ├── can_edit
        └── viewer
              ├── can_read
              └── can_comment
```

One owner relationship implies several permissions without writing extra tuples.

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
define editor: [user] or editor from workspace or owner
```

Do not start by typing DSL. Start with the rule.

## Tuple examples

These tuples are enough to grant Alice edit access:

```text
(team:platformTeam, member, user:alice)
(workspace:productWorkspace, editor, team:platformTeam#member)
(document:roadmapDocument, workspace, workspace:productWorkspace)
```

No tuple says:

```text
(document:roadmapDocument, editor, user:alice)
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

**TypeScript** — mirror the model change in code:

1. Add `"archiver"` and `"can_archive"` to `DocumentRelation` in `typescript/src/shared/rebac.ts`
2. Add the expansion rule to `makeGraphEvaluator` in `typescript/src/authz-service/adapters/graph/makeGraphEvaluator.ts`
3. Add tests: owner is allowed, viewer is denied

**Go** — mirror the same change:

1. Add `RelationDocumentArchiver` and `RelationDocumentCanArchive` constants to `go/internal/shared/rebac.go`
2. Add the expansion rule in `expandDocument` in `go/internal/authz/adapters/graph/evaluator.go`
3. Add a test in `go/internal/authz/adapters/graph/evaluator_test.go` following the AAA pattern

This exercise forces the model, the TypeScript vocabulary, and the Go constants to
stay in sync — which is the real cost of running dual implementations.

## Checkpoint

Read this line out loud:

```text
define editor: [user] or editor from workspace or owner
```

If you can explain it as a graph path, you understand the model. If you cannot,
do not add more permissions yet.
