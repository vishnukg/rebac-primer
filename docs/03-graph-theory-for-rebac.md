# Graph theory for ReBAC

You do not need a computer science degree to understand ReBAC.

You do need a few graph ideas:

- nodes
- edges
- labels
- paths
- directed relationships
- traversal
- reachability
- cycles

This chapter teaches only the graph theory needed for authorization.

Keep a pencil nearby. The chapter works better if you redraw the five-node graph
yourself instead of only reading the diagrams.

## One Sentence Version

A graph is just things connected to other things.

```text
thing --relationship--> thing
```

ReBAC asks whether a user is connected to a resource through relationships that
the model says are useful.

```text
user:alice                 --member of--> team:platformTeam
team:platformTeam#member   --editor of--> workspace:productWorkspace
workspace:productWorkspace --workspace of--> document:roadmapDocument
```

The hard part is not graph theory. The hard part is being precise about what
each connection means.

## Graph Vocabulary Cheat Sheet

| Word | Meaning in plain English | Example in this repo |
|------|--------------------------|----------------------|
| Node | A thing | `user:alice` |
| Edge | A connection between two nodes | `user:alice → team:platformTeam` |
| Label | The name on an edge | `member`, `editor`, `viewer`, `workspace` |
| Direction | Which way the relationship is written | subject points to object |
| Path | A chain of relationships | user -> team -> workspace -> document |
| Reachability | Whether a valid policy path exists | is Alice in the document's editor set? |
| Cycle | A path that loops back | A points to B, B points to A |

When you read ReBAC code, keep asking:

```text
What node am I on?
What relation am I checking?
Which edge can I follow next?
Is this place already on my current recursion path?
```

## Scene

Alice can edit the roadmap document because she is in the platform team, the
platform team can edit the product workspace, and the roadmap document belongs
to the product workspace.

That sentence is already a graph.

In ordinary language and OpenFGA tuple diagrams:

```text
Alice -> team membership -> workspace editor -> document
```

ReBAC makes that graph explicit and asks whether a useful path exists.

## A Map Analogy

Think of the graph like a transit map.

```text
station = node
line    = relation
route   = path
```

Question:

```text
Can Alice get from Station A to Station D?
```

ReBAC question:

```text
Is user:alice in document:roadmapDocument#can_edit?
```

The model is the transit rulebook. It says which lines count for which trip.

```text
viewer line -> can_read
editor line -> can_edit and can_read
owner line  -> can_delete, can_edit, and can_read
```

Tuples are the current map data. If a tuple is removed, a route may disappear.

## Nodes

A node is a thing in the graph.

In this repo, nodes are OpenFGA objects:

```text
user:alice
team:platformTeam
workspace:productWorkspace
document:roadmapDocument
```

Diagram:

```text
┌──────────────────────┐
│ user:alice │
└──────────────────────┘

┌───────────────────┐
│ team:platformTeam │
└───────────────────┘

┌────────────────────────────┐
│ workspace:productWorkspace │
└────────────────────────────┘

┌──────────────────────────┐
│ document:roadmapDocument │
└──────────────────────────┘
```

Nodes are the nouns.

## Edges

An edge connects two nodes.

```text
user:alice --member of--> team:platformTeam
```

In OpenFGA tuple form:

```text
user:alice  member  team:platformTeam
```

Read it as:

```text
user:alice is a member of team:platformTeam
```

The repository's internal Go struct lists the same values differently:

```text
Object=team:platformTeam, Relation=member, User=user:alice
```

That is a struct field order, not a different graph relationship.

## Labels

Edges have labels.

```text
user:alice --member of--> team:platformTeam
user:bob   --viewer of--> workspace:productWorkspace
```

The label matters. A `viewer` edge does not mean the same thing as an `editor`
edge.

In ReBAC, labels are relations:

```text
member
admin
owner
editor
viewer
workspace
can_edit
```

## Directed edges

Most ReBAC relationships are directed.

```text
workspace:productWorkspace --workspace of--> document:roadmapDocument
```

This says the document belongs to the workspace.

It does not automatically say the workspace belongs to the document.

Direction matters because traversal follows model rules.

Diagram:

```text
workspace:productWorkspace ──workspace of──► document:roadmapDocument
```

Do not casually reverse arrows unless the model says that reverse relationship
exists.

## Paths

A path is a sequence of connected edges.

Alice's edit path:

```text
user:alice
      │ member of
      ▼
team:platformTeam

team:platformTeam#member
      │ editor of
      ▼
workspace:productWorkspace
      │ workspace of
      ▼
document:roadmapDocument ✓
```

Reading top to bottom:

- Alice is a member of the platform team.
- That team's member set is an editor of the product workspace.
- The product workspace is the workspace of the roadmap document.
- Document editor access is inherited from the workspace, so `can_edit` is granted.

The OpenFGA tuple for the final relationship is:

```text
workspace:productWorkspace  workspace  document:roadmapDocument
```

"The roadmap document's workspace is the product workspace."

Authorization succeeds when the model can prove a valid path.

## Reachability

Reachability asks:

```text
Can I get from node A to node B by following allowed edges?
```

The evaluator asks a reachability question:

```text
Does the model place user:alice in document:roadmapDocument#can_edit?
```

If yes:

```text
allowed
```

If no:

```text
denied
```

That is the core mental model.

## Graph traversal

Traversal means walking the graph.

The authorizer starts with a question:

```text
Check(user:alice, can_edit, document:roadmapDocument)
```

Then it expands relations using the model:

```text
can_edit -> editor
editor   -> direct editor OR workspace editor OR owner
owner    -> direct owner OR workspace owner
```

Traversal is not random. It follows the model definitions.

## Worked Traversal

Here is the same check as a slow trace:

```text
Question:
  Can user:alice edit document:roadmapDocument?

Step 1:
  document can_edit means document editor.

Step 2:
  document editor can come from workspace editor.

Step 3:
  document:roadmapDocument has workspace workspace:productWorkspace.

Step 4:
  workspace:productWorkspace has editor team:platformTeam#member.

Step 5:
  team:platformTeam#member asks:
  is user:alice a member of team:platformTeam?

Step 6:
  yes, tuple exists:
  user:alice member team:platformTeam

Result:
  allowed
```

The denial case is the same process with a missing path:

```text
Can user:bob edit document:roadmapDocument?

document can_edit -> editor
document editor -> workspace editor
workspace editor -> team:platformTeam#member
team member path does not contain user:bob
Bob only has viewer, not editor

Result: denied
```

## The model is the map

Tuples are facts:

```text
user:alice                  member     team:platformTeam
team:platformTeam#member    editor     workspace:productWorkspace
workspace:productWorkspace  workspace  document:roadmapDocument
```

The model is the map that says how facts can be combined:

```text
document can_edit = document editor
document editor includes workspace editor from workspace
workspace editor accepts team member subject sets
```

Diagram:

```text
Tuples                      Model
──────                      ─────
raw edges       +           traversal rules
who has what                what implies what

             = authorization decision
```

If tuples are data, the model is logic.

## The complete tutorial graph

Here are the five connected objects in the seeded graph:

```text
user:alice ──member of──► team:platformTeam

team:platformTeam#member ──editor of──► workspace:productWorkspace
user:bob ──────────────────viewer of──► workspace:productWorkspace

workspace:productWorkspace ──workspace of──► document:roadmapDocument
```

Three things to notice:

- Alice reaches the workspace through the platform team, not directly.
- Bob has a direct viewer edge to the workspace.
- Casey has no relationship edge in this graph—no path, no access.

Keep this diagram in mind. Every check in this repo is a reachability question against it.

## Subject sets as graph shortcuts

This is a subject set:

```text
team:platformTeam#member
```

It means:

```text
the set of users reachable through team:platformTeam member
```

So this tuple:

```text
team:platformTeam#member  editor  workspace:productWorkspace
```

means:

```text
any user reachable as a member of team:platformTeam is an editor of workspace:productWorkspace
```

Diagram:

```text
user:alice
  └─member of─► team:platformTeam

team:platformTeam#member
  └─editor of─► workspace:productWorkspace
```

Subject sets let one tuple represent many users.

## Cycles

A cycle is a path that can come back to where it started.

```text
A -> B -> C -> A
```

Cycles can happen in relationship graphs. A traversal algorithm must avoid
walking forever.

The Go evaluator keeps an **active-path set** containing each
`(object, relation)` pair in the current recursion chain. If the same pair
appears before the earlier call has returned, the traversal found a cycle and
stops that branch.

Go (`evaluator.go`):

```go
visitKey := relationVisit{object: object, relation: relation}
if r.visiting[visitKey] {
    return false
}
r.visiting[visitKey] = true
defer delete(r.visiting, visitKey)
```

The `defer delete` matters. It removes the pair when that recursive call
finishes, so another independent branch may legitimately evaluate the same node.
A global "seen forever" set would prevent cycles, but it could also cause false
denials in a graph where two branches converge.

## Depth and complexity

Relationship graphs can get deep:

```text
user -> team membership -> workspace -> folder -> project -> document
```

Deep graphs are not automatically bad, but they are harder to debug.

Good ReBAC model design keeps common authorization paths explainable:

```text
Alice can edit the roadmap document because:
  Alice is a member of the platform team.
  the platform team edits the product workspace.
  the roadmap document belongs to the product workspace.
```

If the explanation takes a paragraph, simplify the model.

## Graph theory terms mapped to ReBAC

| Graph term | ReBAC meaning |
|------------|---------------|
| node | object such as `user:alice` or `document:roadmapDocument` |
| edge | tuple relationship |
| label | relation name such as `member` or `editor` |
| path | chain of relationships proving access |
| traversal | checking relation definitions and tuples |
| reachability | whether access can be proven |
| cycle | relationship loop that traversal must avoid |

## The key ReBAC question

Every ReBAC check is this:

```text
Does the subject belong to the requested relation's effective set through valid
model expansions and stored tuples?
```

Example:

```text
Is user:alice in document:roadmapDocument#can_edit?
```

Answer:

```text
yes -> allowed
no  -> denied
```

## Exercise

Draw this graph on paper:

```text
user:alice                  member     team:platformTeam
team:platformTeam#member    editor     workspace:productWorkspace
user:bob                    viewer     workspace:productWorkspace
workspace:productWorkspace  workspace  document:roadmapDocument
```

Then answer:

```text
Can Alice edit the roadmap document?
Can Bob edit the roadmap document?
Can Bob read the roadmap document?
Can Casey read the roadmap document?
```

Do not run the tests first. Predict from the graph.

## Going deeper: the Go implementation

If you want to see how these graph concepts map line-by-line to running code,
read `docs/27-graph-evaluator-walkthrough.md`.

It walks through the complete `alice / can_edit / roadmapDocument` check step
by step — every recursive call, every cycle-guard lookup, every subject-set
resolution — against the actual Go evaluator source.

No graph theory experience required; the walkthrough is designed for readers
coming from this chapter.

## Checkpoint

Explain ReBAC using graph words:

```text
ReBAC stores authorization facts as labeled edges between nodes. A check asks
whether a subject belongs to a relation's effective set through a policy-valid
chain of relationships.
```

If that sentence makes sense, you have enough graph theory to continue.

Next: [ReBAC concepts](04-rebac-concepts.md) gives the graph pieces their ReBAC
names: object, relation, tuple, subject set, and check.
