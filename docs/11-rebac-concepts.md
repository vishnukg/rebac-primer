# ReBAC concepts

Authorization asks one question:

```text
Can this subject perform this action on this object?
```

RBAC usually answers with roles:

```text
workspace editor has role editor
editors can edit documents
therefore the workspace editor can edit documents
```

That works until the important question becomes:

```text
Can the workspace editor edit this specific document?
```

That is where relationship-based access control becomes useful.

## Scene

You are building collaborative docs. A global `editor` role is too blunt. The workspace
editor should edit the roadmap document because she is on the platform team. The
workspace viewer should read it but fail to edit it. The outside collaborator should get
nothing unless the graph gives her a path.

ReBAC is how you model that without creating a new role for every document.

## The ReBAC idea

ReBAC stores authorization as relationships between things.

In this repo:

```text
user:workspaceEditor is a member of team:platformTeam
team:platformTeam is an editor of workspace:productWorkspace
document:roadmapDocument belongs to workspace:productWorkspace
```

From those facts, the system can answer:

```text
Can user:workspaceEditor edit document:roadmapDocument?
```

Yes, because a path exists through the graph.

## Architecture view

In an application, ReBAC usually sits behind a small authorization interface:

```text
┌──────────────┐
│ Client       │ terminal app, browser, API consumer
└──────┬───────┘
       │ request: actor wants action on object
       ▼
┌──────────────┐
│ HTTP Server  │ parse request, identify actor
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Domain       │ knows when authorization is required
│ Service      │
└──────┬───────┘
       │ Check(user, relation, object)
       ▼
┌──────────────┐
│ Authorizer   │ ReBAC graph traversal
└──────┬───────┘
       │ reads tuples
       ▼
┌──────────────┐
│ Tuple Store  │ relationship facts
└──────────────┘
```

In this repo:

```text
HTTP handler -> DocumentService -> Authorizer -> MemoryTupleStore/OpenFGA
```

That separation matters. The HTTP layer should not know graph traversal rules.
The domain service should not know SDK details. The authorizer should answer one
question: allowed or denied.

## The graph

Here is the tutorial graph:

```text
user:workspaceEditor
   |
   | member
   v
team:platformTeam
   |
   | editor via team:platformTeam#member
   v
workspace:productWorkspace
   ^
   | workspace
   |
document:roadmapDocument
```

The important thing is not the drawing. The important thing is the path:

```text
workspaceEditor -> platform team -> product workspace -> roadmap document -> can_edit
```

That path is what authorization checks evaluate.

The same graph as tuples:

```text
┌──────────────────┬──────────┬──────────────────────┐
│ object           │ relation │ user                 │
├──────────────────┼──────────┼──────────────────────┤
│ team:platformTeam    │ member   │ user:workspaceEditor           │
│ workspace:productWorkspace   │ editor   │ team:platformTeam#member │
│ workspace:productWorkspace   │ viewer   │ user:workspaceViewer             │
│ document:roadmapDocument │ workspace│ workspace:productWorkspace       │
└──────────────────┴──────────┴──────────────────────┘
```

Tuples are the data. The model explains how to interpret them.

## Objects

Objects are typed ids:

```text
user:workspaceEditor
team:platformTeam
workspace:productWorkspace
document:roadmapDocument
```

The type before the colon matters. `user:workspaceEditor` and `team:workspaceEditor` are different
objects.

This repo models object ids in TypeScript:

```ts
export type RebacObject<TType extends ObjectType = ObjectType> =
  `${TType}:${string}`;
```

That is a small example of TypeScript supporting the authorization model.

## Relations

Relations are named edges.

Examples:

```text
team:platformTeam member user:workspaceEditor
workspace:productWorkspace editor team:platformTeam#member
document:roadmapDocument workspace workspace:productWorkspace
```

Read each one aloud:

- The workspace editor is a member of the platform team.
- Members of the platform team are editors of the product workspace.
- The roadmap document belongs to the product workspace.

If you cannot read a tuple aloud, your model is probably unclear.

## Tuples

A tuple is one stored fact:

```text
(object, relation, user)
```

In code:

```ts
tuple(workspace("productWorkspace"), "editor", subjectSet(team("platformTeam"), "member"))
```

This means:

```text
workspace:productWorkspace has editor team:platformTeam#member
```

That single tuple grants editor access to every current and future member of the
platform team.

This is the practical power of ReBAC. You do not copy permissions to every user.
You model the relationship once.

## Subject sets

This is a direct user:

```text
user:workspaceEditor
```

This is a subject set:

```text
team:platformTeam#member
```

A subject set means "the set of users who have this relation on this object."

So:

```text
workspace:productWorkspace editor team:platformTeam#member
```

means:

```text
anyone who is a member of team:platformTeam is an editor of workspace:productWorkspace
```

Subject sets are why team membership changes are powerful. If the workspace editor leaves the
team, remove one tuple:

```text
team:platformTeam member user:workspaceEditor
```

The workspace editor immediately loses inherited workspace and document access.

## Permissions vs relationships

Relationships describe facts:

```text
owner
editor
viewer
workspace
member
```

Permissions describe actions:

```text
can_read
can_comment
can_edit
can_delete
```

The OpenFGA model connects them:

```text
define can_edit: editor
```

That line says editors can edit. It keeps action names separate from relationship
names, which makes the model easier to evolve.

## How Check works

A check asks:

```text
Check(user, relation, object)
```

Example:

```text
Check(user:workspaceEditor, can_edit, document:roadmapDocument)
```

The graph evaluator tries to prove the relation.

In this repo, `GraphAuthorizer` produces a trace:

```text
Check whether user:workspaceEditor has can_edit on document:roadmapDocument
document.can_edit includes document.editor
document.editor can inherit workspace.editor from workspace:productWorkspace
Resolve subject set team:platformTeam#member: does it contain user:workspaceEditor?
Found direct tuple (team:platformTeam, member, user:workspaceEditor)
Result: allowed
```

This trace is deliberately educational. Real OpenFGA performs the check
remotely, but the mental model is the same.

Check as a sequence diagram:

```text
DocumentService        Authorizer          Tuple graph
      │                    │                   │
      │ can_edit?          │                   │
      ├───────────────────►│                   │
      │                    │ find document     │
      │                    │ workspace         │
      │                    ├──────────────────►│
      │                    │ workspace:productWorkspace    │
      │                    │◄──────────────────┤
      │                    │ resolve editor    │
      │                    ├──────────────────►│
      │                    │ team:platformTeam     │
      │                    │◄──────────────────┤
      │                    │ resolve member    │
      │                    ├──────────────────►│
      │                    │ user:workspaceEditor found  │
      │                    │◄──────────────────┤
      │ allowed            │                   │
      │◄───────────────────┤                   │
```

## Denial is absence of a path

The workspace viewer has viewer access:

```text
workspace:productWorkspace viewer user:workspaceViewer
```

So the workspace viewer can read and comment. But the workspace viewer cannot edit because
there is no path from `user:workspaceViewer` to `document:roadmapDocument#editor`.

That "near miss" is important:

```text
The workspace viewer can read.
The workspace viewer cannot edit.
```

Good authorization tests should include near misses. They prove your model is
not simply too permissive.

## Exercise

Run:

```bash
npm run dev
```

Read the trace for the workspace editor, the workspace viewer, and the outside collaborator.

Then change `src/testing/fixtures.ts` so the workspace viewer is an editor instead of a viewer:

```ts
tuple(productWorkspace, "editor", workspaceViewer)
```

Predict the new result before running the demo again.

## Checkpoint

Explain why this one tuple is powerful:

```text
workspace:productWorkspace editor team:platformTeam#member
```

Good answer: it grants workspace editor access to the set of current and future
platform team members, without writing one tuple per user per document.
