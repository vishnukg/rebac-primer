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

## Scene

Alice can edit the roadmap because she is in the platform team, and the platform
team can edit the Acme workspace, and the roadmap belongs to the Acme workspace.

That sentence is already a graph.

```text
Alice -> Platform Team -> Acme Workspace -> Roadmap
```

ReBAC makes that graph explicit and asks whether a useful path exists.

## Nodes

A node is a thing in the graph.

In this repo, nodes are OpenFGA objects:

```text
user:alice
team:platform
workspace:acme
document:roadmap
```

Diagram:

```text
┌────────────┐   ┌───────────────┐   ┌────────────────┐   ┌──────────────────┐
│ user:alice │   │ team:platform │   │ workspace:acme │   │ document:roadmap │
└────────────┘   └───────────────┘   └────────────────┘   └──────────────────┘
```

Nodes are the nouns.

## Edges

An edge connects two nodes.

```text
user:alice --member--> team:platform
```

In OpenFGA tuple form, the same fact is stored as:

```text
(team:platform, member, user:alice)
```

The tuple is written from object perspective:

```text
team:platform has member user:alice
```

When drawing it for intuition, it is often easier to read from user outward:

```text
user:alice is member of team:platform
```

Both are the same relationship. Be comfortable flipping the sentence.

## Labels

Edges have labels.

```text
user:alice --member--> team:platform
user:bob   --viewer--> workspace:acme
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
document:roadmap --workspace--> workspace:acme
```

This says the document belongs to the workspace.

It does not automatically say the workspace belongs to the document.

Direction matters because traversal follows model rules.

Diagram:

```text
document:roadmap ──workspace──► workspace:acme
```

Do not casually reverse arrows unless the model says that reverse relationship
exists.

## Paths

A path is a sequence of connected edges.

Alice's edit path:

```text
user:alice
  ──member──► team:platform
  ──editor──► workspace:acme
  ◄─workspace── document:roadmap
  ──editor/can_edit──► allowed
```

The drawing mixes intuitive direction with OpenFGA's object-centric relations.
The important point is the chain of facts:

```text
Alice is in team.
Team edits workspace.
Document belongs to workspace.
Workspace editor implies document editor.
Document editor implies can_edit.
```

Authorization succeeds when the model can prove a valid path.

## Reachability

Reachability asks:

```text
Can I get from node A to node B by following allowed edges?
```

ReBAC check asks a reachability question:

```text
Can user:alice reach document:roadmap#can_edit?
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
Check(user:alice, can_edit, document:roadmap)
```

Then it expands relations using the model:

```text
can_edit -> editor
editor   -> direct editor OR workspace editor OR owner
owner    -> direct owner OR workspace owner
```

Traversal is not random. It follows the model definitions.

## The model is the map

Tuples are facts:

```text
team:platform member user:alice
workspace:acme editor team:platform#member
document:roadmap workspace workspace:acme
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

## Subject sets as graph shortcuts

This is a subject set:

```text
team:platform#member
```

It means:

```text
the set of users reachable through team:platform member
```

So this tuple:

```text
workspace:acme editor team:platform#member
```

means:

```text
any user reachable as a member of team:platform is an editor of workspace:acme
```

Diagram:

```text
user:alice ──member──► team:platform
                              │
                              │ team:platform#member
                              ▼
                       workspace:acme editor
```

Subject sets let one tuple represent many users.

## Cycles

A cycle is a path that can come back to where it started.

```text
A -> B -> C -> A
```

Cycles can happen in relationship graphs. A traversal algorithm must avoid
walking forever.

This repo's `GraphAuthorizer` keeps a `visited` set:

```ts
const visitKey: VisitKey = `${object}#${relation}`;
if (visited.has(visitKey)) {
  return false;
}
visited.add(visitKey);
```

That is basic cycle protection.

## Depth and complexity

Relationship graphs can get deep:

```text
user -> team -> workspace -> folder -> project -> document
```

Deep graphs are not automatically bad, but they are harder to debug.

Good ReBAC model design keeps common authorization paths explainable:

```text
Alice can edit roadmap because:
  Alice is member of platform.
  Platform edits Acme.
  Roadmap belongs to Acme.
```

If the explanation takes a paragraph, simplify the model.

## Graph theory terms mapped to ReBAC

| Graph term | ReBAC meaning |
|------------|---------------|
| node | object such as `user:alice` or `document:roadmap` |
| edge | tuple relationship |
| label | relation name such as `member` or `editor` |
| path | chain of relationships proving access |
| traversal | checking relation definitions and tuples |
| reachability | whether access can be proven |
| cycle | relationship loop that traversal must avoid |

## The key ReBAC question

Every ReBAC check is this:

```text
Is there a valid path from the user to the requested relation on the object?
```

Example:

```text
Is there a valid path from user:alice to document:roadmap#can_edit?
```

Answer:

```text
yes -> allowed
no  -> denied
```

## Exercise

Draw this graph on paper:

```text
team:platform member user:alice
workspace:acme editor team:platform#member
workspace:acme viewer user:bob
document:roadmap workspace workspace:acme
```

Then answer:

```text
Can Alice edit roadmap?
Can Bob edit roadmap?
Can Bob read roadmap?
Can Chandra read roadmap?
```

Do not run the tests first. Predict from the graph.

## Checkpoint

Explain ReBAC using graph words:

```text
ReBAC stores authorization facts as labeled edges between nodes. A check asks
whether the model can traverse a valid path from a user to a requested relation
on an object.
```

If that sentence makes sense, you have enough graph theory to continue.
