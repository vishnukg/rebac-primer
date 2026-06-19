# Notes: graph theory + OpenFGA (the minimum that matters)

Companion to `START-HERE.md`. Two topics, beginner level, grounded in this repo's
one example. Depth versions live in `docs/03-graph-theory-for-rebac.md` and
`docs/05-openfga-model.md` — read those when these notes feel too short.

---

## Part 1 — Graph theory (only what ReBAC needs)

You need ~6 ideas. That's it. Skip everything else you've heard about graphs.

### 1. A graph is just things connected to things

- **Node** = a thing. Here: `user:alice`, `team:platformTeam`, `workspace:productWorkspace`, `document:roadmapDocument`.
- **Edge** = a connection between two things.

```
user:alice  ──────►  team:platformTeam
  (node)     (edge)       (node)
```

### 2. Edges have a direction and a label

- **Directed**: the arrow points one way. "Alice → team" (Alice is in the team) does **not** automatically mean "team → Alice."
- **Label** = the *kind* of connection. In ReBAC the label is the **relation**: `member`, `editor`, `viewer`, `workspace`.

```
user:alice ──member──► team:platformTeam
user:bob   ──viewer──► workspace:productWorkspace
```

A stored fact (a **tuple**) is exactly one labeled edge.

### 3. Our whole graph is 4 edges

These are the four fixture tuples (`internal/fixtures/fixtures.go`):

```
user:alice ─[member]─►  team:platformTeam
team:platformTeam (its #member set) ─[editor]─► workspace:productWorkspace
user:bob ─[viewer]─►   workspace:productWorkspace
document:roadmapDocument ─[workspace]─► workspace:productWorkspace
```

Drawn together:

```
user:alice ──member──► team:platformTeam
                              │ editor (via team:platformTeam#member)
                              ▼
user:bob ──viewer──► workspace:productWorkspace
                              ▲ workspace
                              │
                     document:roadmapDocument
```

### 4. A path is a chain of edges; reachability is "does a path exist?"

A **path** is hops you can follow end to end:

```
alice ─member─► team ─editor─► workspace ◄─workspace─ document
```

**Reachability** is the core question behind a ReBAC permission check:

> Starting from the document, following allowed edges, can I reach the user?

`Can alice edit the roadmap?` = `Is there a path from document:roadmapDocument
(via edges that count as "can_edit") to user:alice?` Yes → allowed. No → denied.
**That is the entire mental model.**

### 5. Traversal = walking the graph to find a path (DFS)

The evaluator uses **depth-first search**: pick one branch, follow it all the way
down; if it dead-ends, back up and try the next branch. You already watched this
in the trace program — the `owner` branch fails, it backs out, the `editor`
branch succeeds. That explore-and-backtrack *is* traversal.

### 6. Cycles, and why there is an active-path set

A **cycle** is a path that loops back on itself (A → B → A). If a document's
workspace pointed at itself, naive traversal would recurse forever. The guard
remembers every `(object, relation)` pair in the current recursion chain. If it
sees the same pair before the earlier call returns, it stops that cycle:

```
Cycle detected at workspace:productWorkspace#owner; stop this branch
```

The pair is removed when the call returns. That detail lets another independent
branch evaluate the same node without causing a false denial.

### Graph word ↔ ReBAC word

| Graph | ReBAC |
|-------|-------|
| node | object (`user:alice`, `document:x`) |
| edge | tuple |
| label | relation (`member`, `editor`) |
| path | chain of relationships that proves access |
| reachability | "is access provable?" |
| cycle | relationship loop the traversal must not get stuck in |

---

## Part 2 — OpenFGA and its DSL

### What OpenFGA is

A dedicated **authorization service**. Instead of scattering `if` checks across
your app, you ask one service: *"can user X do Y on object Z?"* It is open
source and inspired by Google's **Zanzibar** authorization-system paper.
This repo first *builds the idea from scratch* (the graph evaluator) so OpenFGA
stops looking like magic — `evaluator.go` does in-process what OpenFGA does as a
service.

### The three layers (each changes on a different clock)

```
store   →  a namespace / environment        (almost never changes)
model   →  the schema: types + relations     (changes rarely — it's "the rules")
tuples  →  the facts: who relates to what     (change constantly — it's "the data")
```

Keeping the **rules** (model) separate from the **data** (tuples) is the whole
point. You don't write "Alice is a viewer" for every editor — you say once in the
model "editors are also viewers," and it applies to everyone forever.

### Tuples = the facts (the edges from Part 1)

OpenFGA commonly displays tuple keys as `(user, relation, object)`:

```
user:alice              member   team:platformTeam
team:platformTeam#member editor   workspace:productWorkspace
document:roadmapDocument workspace workspace:productWorkspace
```

This repository's Go `TupleKey` struct orders the fields as
`(object, relation, user)`. The values mean the same thing; only the display
order differs. Always read the field names rather than relying on position.

The `team:platformTeam#member` form is a **subject set**: "everyone who has
`member` on `team:platformTeam`," not one person. One tuple grants a whole group —
add someone to the team and they inherit access with no new tuple.

### The model = the DSL. Walk `deployments/openfga/model.fga` line by line

```
model
  schema 1.1
```
Version header. Always there.

```
type user
```
A `user` is a plain subject — no relations of its own. It's a leaf.

```
type team
  relations
    define admin:  [user]
    define member: [user] or admin
```
- `define admin: [user]` — the `[user]` is a **type restriction**: only a
  `user:*` can be written directly as an admin. (`team:x` as admin → rejected.)
- `define member: [user] or admin` — a member is anyone written directly **or**
  anyone who is `admin`. `or` is set union. This is how "admins are also members"
  is stated once, as a rule.

```
type workspace
  relations
    define owner:  [user, team#admin]
    define editor: [user, team#member] or owner
    define viewer: [user, team#member] or editor
```
- `[user, team#admin]` — two kinds of subject may be written directly: a literal
  `user:*`, **or** a subject set `team:*#admin` (all admins of some team).
- `or owner` / `or editor` builds the hierarchy: **owner ⊇ editor ⊇ viewer**.
  Owners can do anything editors can; editors anything viewers can.

```
type document
  relations
    define workspace:   [workspace]
    define owner:       [user] or owner from workspace
    define editor:      [user] or editor from workspace or owner
    define viewer:      [user] or viewer from workspace or editor
    define can_read:    viewer
    define can_comment: viewer
    define can_edit:    editor
    define can_delete:  owner
```
Three new things here:

- `define workspace: [workspace]` — a document points at its parent workspace.
  This is the structural edge that enables inheritance.
- **`X from Y`** (the key construct, "tuple-to-userset") —
  `editor from workspace` means: *follow this document's `workspace` edge to the
  workspace object, then check `editor` there.* This is how permission flows from
  parent to child. It is graph traversal expressed in one line.
- **Computed relations** — `define can_edit: editor` means `can_edit` is not
  stored anywhere; it's *computed* as "whoever is `editor`." This separates
  **permissions** (`can_edit`, action words) from **relations** (`editor`,
  fact words), so you can rename actions without touching the tuples.

DSL constructs you'll see, summarized:

| Syntax | Name | Meaning |
|--------|------|---------|
| `[user]`, `[user, team#member]` | type restriction | which subject types may be assigned directly |
| `or` | union | satisfied by either side |
| `and` | intersection | must satisfy both (not used here) |
| `but not` | exclusion | satisfied unless the right side holds (not used here) |
| `X from Y` | tuple-to-userset | follow relation `Y` to a parent, check `X` there |
| `define can_edit: editor` | computed userset | this relation = that relation, same object |

### The Check API = asking the reachability question

```
Check(user:alice, can_edit, document:roadmapDocument)  ->  allowed / denied
```

OpenFGA does the graph traversal (Part 1) over the model + tuples and returns a
boolean. Other APIs you'll meet later: `ListObjects` ("which docs can Alice
read?"), `BatchCheck` (many checks at once), `Expand` (debug a relation).

### How this repo maps to OpenFGA (the payoff)

| Concept | From-scratch code | OpenFGA |
|---------|-------------------|---------|
| the rules | `internal/authz/model.go` tables | `model.fga` DSL |
| the facts | in-memory tuple store (`internal/authz/store.go`) | OpenFGA's datastore |
| `X from Y` inheritance | `expandDocument` in `internal/authz/evaluator.go` | the `from` keyword |
| computed permission | `documentRules[can_edit] = {editor}` | `define can_edit: editor` |
| the Check | `GraphEvaluator.Evaluate` | OpenFGA `/check` |

The repo can run **either** backend behind the same interface — set
`AUTHZ_BACKEND=openfga` (see `docs/26-openfga-migration.md` and
`docs/34-openfga-adapter-walkthrough.md`). Same questions, same answers; one is
in your process, one is a real service.

### Play with it (optional, fun)

- [play.fga.dev](https://play.fga.dev) — paste a model + tuples, run checks in a
  browser, see the resolution visually. Try pasting `model.fga`.
- Then `make openfga/up && make openfga/seed && make server-openfga`.

---

## Glossary

- **Object / node** — a typed thing, `type:id`.
- **Relation / label** — a named edge: `member`, `editor`, `can_edit`.
- **Tuple / edge / fact** — one stored relationship `(user, relation, object)`.
- **Subject set** — `team:x#member`, "everyone with `member` on `team:x`."
- **Model / schema** — the rules: types, relations, how they imply each other.
- **Check** — the allow/deny question; under the hood, graph reachability.
- **Tuple-to-userset (`from`)** — inheritance: follow an edge to a parent and
  check there.
- **Zanzibar** — Google's authorization-system paper that inspired OpenFGA's
  relationship-based model.

## Checkpoint

Read this line aloud and explain it as a graph path:

```
define editor: [user] or editor from workspace or owner
```

> "A document editor is: someone written directly as an editor, **or** an editor
> of the workspace this document belongs to, **or** an owner of the document."

If that makes sense, you've got both topics. Go run the trace program again and
watch `editor from workspace` actually happen (the `can inherit ... from
workspace` lines).
