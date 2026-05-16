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

The workspace editor can edit the roadmap document because she is in the platform team,
the platform team can edit the product workspace, and the roadmap document belongs to
the product workspace.

That sentence is already a graph.

```text
workspace editor -> platform team -> product workspace -> roadmap document
```

ReBAC makes that graph explicit and asks whether a useful path exists.

## Nodes

A node is a thing in the graph.

In this repo, nodes are OpenFGA objects:

```text
user:workspaceEditor
team:platformTeam
workspace:productWorkspace
document:roadmapDocument
```

Diagram:

```text
┌──────────────────────┐
│ user:workspaceEditor │
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
user:workspaceEditor --member--> team:platformTeam
```

In OpenFGA tuple form, the same fact is stored as:

```text
(team:platformTeam, member, user:workspaceEditor)
```

The tuple is written from object perspective:

```text
team:platformTeam has member user:workspaceEditor
```

When drawing it for intuition, it is often easier to read from user outward:

```text
user:workspaceEditor is member of team:platformTeam
```

Both are the same relationship. Be comfortable flipping the sentence.

## Labels

Edges have labels.

```text
user:workspaceEditor --member--> team:platformTeam
user:workspaceViewer   --viewer--> workspace:productWorkspace
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
document:roadmapDocument --workspace--> workspace:productWorkspace
```

This says the document belongs to the workspace.

It does not automatically say the workspace belongs to the document.

Direction matters because traversal follows model rules.

Diagram:

```text
document:roadmapDocument ──workspace──► workspace:productWorkspace
```

Do not casually reverse arrows unless the model says that reverse relationship
exists.

## Paths

A path is a sequence of connected edges.

The workspace editor's edit path:

```text
user:workspaceEditor
      │ member
      ▼
team:platformTeam
      │ editor (team:platformTeam#member)
      ▼
workspace:productWorkspace
      ▲ workspace
      │
document:roadmapDocument ──► can_edit ✓
```

Reading top to bottom:

- The workspace editor is a member of the platform team.
- Platform team members are editors of the product workspace.
- The roadmap document declares it belongs to that workspace.
- Document editor access is inherited from the workspace, so `can_edit` is granted.

The `workspace` arrow points upward because the tuple is stored on the document:

```text
(document:roadmapDocument, workspace, workspace:productWorkspace)
```

"The roadmap document's workspace is the product workspace."

Authorization succeeds when the model can prove a valid path.

## Reachability

Reachability asks:

```text
Can I get from node A to node B by following allowed edges?
```

ReBAC check asks a reachability question:

```text
Can user:workspaceEditor reach document:roadmapDocument#can_edit?
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
Check(user:workspaceEditor, can_edit, document:roadmapDocument)
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
team:platformTeam member user:workspaceEditor
workspace:productWorkspace editor team:platformTeam#member
document:roadmapDocument workspace workspace:productWorkspace
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

Here is all four nodes and their edges in one diagram:

```text
user:workspaceEditor        user:workspaceViewer
       │                            │
       │ member                     │ viewer
       ▼                            │
team:platformTeam                   │
       │ editor                     │
       │ (team:platformTeam#member) │
       └────────────────┐           │
                        ▼           ▼
               workspace:productWorkspace
                        ▲
                        │ workspace
                        │
               document:roadmapDocument
```

Three things to notice:

- The workspace editor reaches the workspace through the platform team, not directly.
- The workspace viewer has a direct viewer edge to the workspace.
- The outside collaborator has no node in this graph — no path, no access.

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
workspace:productWorkspace editor team:platformTeam#member
```

means:

```text
any user reachable as a member of team:platformTeam is an editor of workspace:productWorkspace
```

Diagram:

```text
user:workspaceEditor ──member──► team:platformTeam
                              │
                              │ team:platformTeam#member
                              ▼
                       workspace:productWorkspace editor
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
The workspace editor can edit the roadmap document because:
  the workspace editor is a member of the platform team.
  the platform team edits the product workspace.
  the roadmap document belongs to the product workspace.
```

If the explanation takes a paragraph, simplify the model.

## Graph theory terms mapped to ReBAC

| Graph term | ReBAC meaning |
|------------|---------------|
| node | object such as `user:workspaceEditor` or `document:roadmapDocument` |
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
Is there a valid path from user:workspaceEditor to document:roadmapDocument#can_edit?
```

Answer:

```text
yes -> allowed
no  -> denied
```

## Exercise

Draw this graph on paper:

```text
team:platformTeam member user:workspaceEditor
workspace:productWorkspace editor team:platformTeam#member
workspace:productWorkspace viewer user:workspaceViewer
document:roadmapDocument workspace workspace:productWorkspace
```

Then answer:

```text
Can the workspace editor edit the roadmap document?
Can the workspace viewer edit the roadmap document?
Can the workspace viewer read the roadmap document?
Can the outside collaborator read the roadmap document?
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
