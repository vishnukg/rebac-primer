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

You are building collaborative docs. A global `editor` role is too blunt.
Alice should edit the roadmap document because she is on the platform team. Bob
should read it but fail to edit it. Casey should get nothing unless the graph
gives them a path.

ReBAC is how you model that without creating a new role for every document.

## Cast Of Characters

The examples use human names for actors and product names for resources:

| Person or object | ReBAC ID | What it means |
|------------------|----------|---------------|
| Alice | `user:alice` | platform team member; can read and edit the roadmap |
| Bob | `user:bob` | direct workspace viewer; can read but cannot edit |
| Casey | `user:casey` | outside collaborator; denied by default |
| Platform Team | `team:platformTeam` | team whose members edit the product workspace |
| Product Workspace | `workspace:productWorkspace` | workspace that owns the roadmap document |
| Roadmap Document | `document:roadmapDocument` | document being protected |

## The ReBAC idea

ReBAC stores authorization as relationships between things.

In this repo:

```text
user:alice is a member of team:platformTeam
team:platformTeam is an editor of workspace:productWorkspace
document:roadmapDocument belongs to workspace:productWorkspace
```

From those facts, the system can answer:

```text
Can user:alice edit document:roadmapDocument?
```

Yes, because a path exists through the graph.

## Build ReBAC From Normal Sentences

Do not start by thinking about OpenFGA syntax. Start with product sentences.

Product sentence:

```text
Workspace editors can edit documents in that workspace.
```

Break it into nouns:

```text
workspace editor
document
workspace
```

Break it into relationships:

```text
user is a member of team
team members are editors of workspace
document belongs to workspace
workspace editors are document editors
document editors can edit documents
```

Then write the graph facts:

```text
team:platformTeam member user:alice
workspace:productWorkspace editor team:platformTeam#member
document:roadmapDocument workspace workspace:productWorkspace
```

Then write the model rules:

```text
document editor can come from workspace editor
document can_edit comes from document editor
```

ReBAC is the combination of:

```text
facts + rules -> decision
```

## Three Layers To Keep Separate

Beginners often mix these together. Keep them separate:

| Layer | Question | Example |
|-------|----------|---------|
| Identity | Who is the user? | `user:alice` |
| Relationship facts | What relationships exist now? | team member, workspace editor |
| Authorization model | How do relationships imply permissions? | editor implies can_edit |

The final check uses all three:

```text
identity: user:alice
facts:    user is member of platform team
model:    team members can be workspace editors, workspace editors can edit docs
result:   allowed
```

## The Ladder Model

For this repo, think of document access as a ladder:

```text
owner
  |
  v
editor
  |
  v
viewer
```

Higher rungs include lower rungs:

```text
owner  -> editor -> viewer
editor -> viewer
viewer -> can_read and can_comment
editor -> can_edit
owner  -> can_delete
```

So the questions become:

```text
can_read?    find viewer
can_comment? find viewer
can_edit?    find editor
can_delete?  find owner
```

The graph traversal is mostly a search for the needed rung.

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
│ Document     │ knows when authorization is required
│ Domain       │
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
HTTP handler -> Documents -> Authorizer -> TupleStore/OpenFGA
```

That separation matters. The HTTP layer should not know graph traversal rules.
The document domain should not know SDK details. The authorizer should answer one
question: allowed or denied.

## End-To-End Request Example

Here is the whole story, from request to decision:

```text
PATCH /documents/roadmapDocument
actorId=alice
body="new roadmap"
```

In a production app, `actorId` would come from an authenticated session or token.
This repo passes it explicitly so the authorization lesson is visible.

```text
HTTP handler
  parses document id and actor id
  |
  v
documents.update
  knows editing a document requires can_edit
  |
  v
Authorizer.Check
  asks: user:alice can_edit document:roadmapDocument?
  |
  v
Tuple store + model rules
  find a valid relationship path
  |
  v
allowed
  |
  v
repository saves updated document
```

If the actor is Bob (`user:bob`), the first half is the same. Only the graph
answer changes:

```text
user:bob can_read document:roadmapDocument  -> allowed
user:bob can_edit document:roadmapDocument  -> denied
```

## The graph

Here is the tutorial graph:

```text
user:alice
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
alice -> platform team -> product workspace -> roadmap document -> can_edit
```

That path is what authorization checks evaluate.

The same graph as tuples:

```text
┌────────────────────────────┬───────────┬────────────────────────────┐
│ object                     │ relation  │ user                       │
├────────────────────────────┼───────────┼────────────────────────────┤
│ team:platformTeam          │ member    │ user:alice                 │
│ workspace:productWorkspace │ editor    │ team:platformTeam#member    │
│ workspace:productWorkspace │ viewer    │ user:bob                   │
│ document:roadmapDocument   │ workspace │ workspace:productWorkspace  │
└────────────────────────────┴───────────┴────────────────────────────┘
```

Tuples are the data. The model explains how to interpret them.

## Objects

Objects are typed ids:

```text
user:alice
team:platformTeam
workspace:productWorkspace
document:roadmapDocument
```

The type before the colon matters. `user:alice` and `team:alice` are different
objects.

This repo models object ids in TypeScript as branded strings:

```ts
// typescript/src/shared/rebac.ts
export type RebacObject<TType extends ObjectType = ObjectType> =
  `${TType}:${string}`;
```

In Go the same idea uses named types — a `string` that the compiler treats as a
distinct type:

```go
// go/internal/authz/types.go
type Object string  // "type:id" — e.g. "workspace:productWorkspace"
```

Both approaches make it a compile error to pass a raw string where a typed object
is expected.

## Relations

Relations are named edges.

Examples:

```text
team:platformTeam member user:alice
workspace:productWorkspace editor team:platformTeam#member
document:roadmapDocument workspace workspace:productWorkspace
```

Read each one aloud:

- Alice is a member of the platform team.
- Members of the platform team are editors of the product workspace.
- The roadmap document belongs to the product workspace.

If you cannot read a tuple aloud, your model is probably unclear.

## Tuples

A tuple is one stored fact:

```text
(object, relation, user)
```

In TypeScript:

```ts
// typescript/test/fixtures.ts
tuple(workspace("productWorkspace"), "editor", subjectSet(team("platformTeam"), "member"))
```

In Go:

```go
// go/internal/fixtures/fixtures.go
authz.Tuple(ProductWorkspace, authz.RelationWorkspaceEditor, authz.SubjectSet(PlatformTeam, authz.RelationTeamMember))
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
user:alice
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

Subject sets are why team membership changes are powerful. If Alice leaves the
platform team, remove one tuple:

```text
team:platformTeam member user:alice
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
Check(user:alice, can_edit, document:roadmapDocument)
```

The graph evaluator tries to prove the relation.

In this repo, `makeGraphEvaluator` produces a trace:

```text
Check whether user:alice has can_edit on document:roadmapDocument
document.can_edit includes document.editor
document.editor can inherit workspace.editor from workspace:productWorkspace
Resolve subject set team:platformTeam#member: does it contain user:alice?
Found direct tuple (team:platformTeam, member, user:alice)
Result: allowed
```

This trace is deliberately educational. Real OpenFGA performs the check
remotely, but the mental model is the same.

Check as a sequence diagram:

```text
Documents              Authorizer          Tuple graph
      │                    │                    │
      │ can_edit?          │                    │
      ├───────────────────►│                    │
      │                    │ find workspace     │
      │                    ├───────────────────►│
      │                    │◄───────────────────┤ workspace:productWorkspace
      │                    │ resolve editor     │
      │                    ├───────────────────►│
      │                    │◄───────────────────┤ team:platformTeam#member
      │                    │ resolve member     │
      │                    ├───────────────────►│
      │                    │◄───────────────────┤ user:alice ✓
      │ allowed            │                    │
      │◄───────────────────┤                    │
```

## Denial is absence of a path

Bob has viewer access:

```text
workspace:productWorkspace viewer user:bob
```

So Bob can read and comment. But Bob cannot edit because
there is no path from `user:bob` to `document:roadmapDocument#editor`.

That "near miss" is important:

```text
Bob can read.
Bob cannot edit.
```

Good authorization tests should include near misses. They prove your model is
not simply too permissive.

## ReBAC For Agentic Tool Calls

In an agentic system, the agent usually does not directly "own" the data. It is
acting for a user, a team, a workflow, or an organization.

That means the authorization question should include delegation:

```text
Can this agent perform this action on this object for this user?
```

You can model that in layers.

Layer 1: user access to the object.

```text
Check(user:alice, can_edit, document:roadmapDocument)
```

Layer 2: agent access to the tool.

```text
Check(agent:docAssistant, can_use, tool:updateDocument)
```

Layer 3: optional delegation from user to agent.

```text
Check(agent:docAssistant, delegate, user:alice)
```

A combined decision might look like:

```text
allow tool call if:
  user can_edit document
  agent can_use updateDocument tool
  agent is delegated by this user or this workspace
```

The graph could contain facts like:

```text
user:alice delegate agent:docAssistant
tool:updateDocument can_use agent:docAssistant
team:platformTeam member user:alice
workspace:productWorkspace editor team:platformTeam#member
document:roadmapDocument workspace workspace:productWorkspace
```

The first tuple reads "the agent is in alice's `delegate` set" — i.e. Alice has
delegated to the agent. That direction matches the layer-3 check
`Check(agent:docAssistant, delegate, user:alice)`, which asks whether the agent
appears in alice's `delegate` relation.

Then a tool call is not just a prompt response. It becomes an authorized action:

```text
agent proposes action
  |
  v
server validates requested object and relation
  |
  v
server runs ReBAC checks
  |
  +-- denied  -> tool call blocked
  |
  +-- allowed -> tool call executes
```

This is especially useful when agents can operate across many resources:

```text
docs
issues
calendar events
source code repositories
customer records
deployment systems
```

ReBAC gives each tool call a precise permission question instead of relying on a
broad "agent is trusted" flag.

## Exercise

**TypeScript:** run the demo and read the trace:

```bash
make ts-server   # start the server
# or inside the container: npm run dev
```

Then change `typescript/test/fixtures.ts` so Bob is an editor instead of
a viewer:

```ts
tuple(productWorkspace, "editor", bob)
```

**Go:** run the graph test with verbose output and read every trace line:

```bash
make go-test
# or: go test -v ./internal/authz/...
```

Open `go/internal/authz/graph_test.go` and change the fixture similarly. Predict
the new result before running the test again.

For both implementations: predict the new result before running anything.

## Checkpoint

Explain why this one tuple is powerful:

```text
workspace:productWorkspace editor team:platformTeam#member
```

Good answer: it grants workspace editor access to the set of current and future
platform team members, without writing one tuple per user per document.
