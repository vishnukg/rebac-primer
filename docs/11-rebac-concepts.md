# ReBAC concepts

Authorization asks one question:

```text
Can this subject perform this action on this object?
```

RBAC usually answers with roles:

```text
alice has role editor
editors can edit documents
therefore alice can edit documents
```

That works until the important question becomes:

```text
Can Alice edit this specific document?
```

That is where relationship-based access control becomes useful.

## Scene

You are building collaborative docs. A global `editor` role is too blunt. Alice
should edit the Acme roadmap because she is on the platform team. Bob should
read it but fail to edit it. Chandra should get nothing unless the graph gives
her a path.

ReBAC is how you model that without creating a new role for every document.

## The ReBAC idea

ReBAC stores authorization as relationships between things.

In this repo:

```text
user:alice is a member of team:platform
team:platform is an editor of workspace:acme
document:roadmap belongs to workspace:acme
```

From those facts, the system can answer:

```text
Can user:alice edit document:roadmap?
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
user:alice
   |
   | member
   v
team:platform
   |
   | editor via team:platform#member
   v
workspace:acme
   ^
   | workspace
   |
document:roadmap
```

The important thing is not the drawing. The important thing is the path:

```text
alice -> platform team -> acme workspace -> roadmap document -> can_edit
```

That path is what authorization checks evaluate.

The same graph as tuples:

```text
┌──────────────────┬──────────┬──────────────────────┐
│ object           │ relation │ user                 │
├──────────────────┼──────────┼──────────────────────┤
│ team:platform    │ member   │ user:alice           │
│ workspace:acme   │ editor   │ team:platform#member │
│ workspace:acme   │ viewer   │ user:bob             │
│ document:roadmap │ workspace│ workspace:acme       │
└──────────────────┴──────────┴──────────────────────┘
```

Tuples are the data. The model explains how to interpret them.

## Objects

Objects are typed ids:

```text
user:alice
team:platform
workspace:acme
document:roadmap
```

The type before the colon matters. `user:alice` and `team:alice` are different
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
team:platform member user:alice
workspace:acme editor team:platform#member
document:roadmap workspace workspace:acme
```

Read each one aloud:

- Alice is a member of the platform team.
- Members of the platform team are editors of the Acme workspace.
- The roadmap document belongs to the Acme workspace.

If you cannot read a tuple aloud, your model is probably unclear.

## Tuples

A tuple is one stored fact:

```text
(object, relation, user)
```

In code:

```ts
tuple(workspace("acme"), "editor", subjectSet(team("platform"), "member"))
```

This means:

```text
workspace:acme has editor team:platform#member
```

That single tuple grants editor access to every current and future member of the
platform team.

This is the practical power of ReBAC. You do not copy permissions to every user.
You model the relationship once.

## Subject sets

This is a direct user:

```text
user:alice
```

This is a subject set:

```text
team:platform#member
```

A subject set means "the set of users who have this relation on this object."

So:

```text
workspace:acme editor team:platform#member
```

means:

```text
anyone who is a member of team:platform is an editor of workspace:acme
```

Subject sets are why team membership changes are powerful. If Alice leaves the
team, remove one tuple:

```text
team:platform member user:alice
```

Alice immediately loses inherited workspace and document access.

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
Check(user:alice, can_edit, document:roadmap)
```

The graph evaluator tries to prove the relation.

In this repo, `GraphAuthorizer` produces a trace:

```text
Check whether user:alice has can_edit on document:roadmap
document.can_edit includes document.editor
document.editor can inherit workspace.editor from workspace:acme
Resolve subject set team:platform#member: does it contain user:alice?
Found direct tuple (team:platform, member, user:alice)
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
      │                    │ workspace:acme    │
      │                    │◄──────────────────┤
      │                    │ resolve editor    │
      │                    ├──────────────────►│
      │                    │ team:platform     │
      │                    │◄──────────────────┤
      │                    │ resolve member    │
      │                    ├──────────────────►│
      │                    │ user:alice found  │
      │                    │◄──────────────────┤
      │ allowed            │                   │
      │◄───────────────────┤                   │
```

## Denial is absence of a path

Bob has viewer access:

```text
workspace:acme viewer user:bob
```

So Bob can read and comment. But Bob cannot edit because there is no path from
Bob to `document:roadmap#editor`.

That "near miss" is important:

```text
Bob can read.
Bob cannot edit.
```

Good authorization tests should include near misses. They prove your model is
not simply too permissive.

## Exercise

Run:

```bash
npm run dev
```

Read the trace for Alice, Bob, and Chandra.

Then change `src/testing/fixtures.ts` so Bob is an editor instead of a viewer:

```ts
tuple(acme, "editor", bob)
```

Predict the new result before running the demo again.

## Checkpoint

Explain why this one tuple is powerful:

```text
workspace:acme editor team:platform#member
```

Good answer: it grants workspace editor access to the set of current and future
platform team members, without writing one tuple per user per document.
