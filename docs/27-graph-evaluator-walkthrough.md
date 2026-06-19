# How the graph evaluator works

This document explains, step by step, how the in-process graph evaluator
answers a permission check.  No prior graph theory knowledge required.

The relevant source files are:

- `internal/authz/evaluator.go` — the traversal algorithm
- `internal/authz/model.go` — the rule tables
- `internal/authz/store.go` — the in-memory tuple store

---

## What a graph is

A **graph** is a set of **nodes** connected by **edges**.

That is the whole definition.  In our system:

- **Nodes** are entities: a user, a team, a workspace, a document.
- **Edges** are relationship tuples: a named connection between two nodes.

A tuple like:

```
(team:platformTeam, member, user:alice)
```

is a directed edge that reads: "there is a `member` edge pointing from
`team:platformTeam` to `user:alice`."

The arrow has a direction — it points *from* the object (`team:platformTeam`)
*to* the subject (`user:alice`).  The relation (`member`) is the label on the
edge.

---

## The four fixture tuples as a graph

The project seeds four tuples. The table stores them in this repository's
`(object, relation, subject)` order, then paraphrases them from the user's point
of view:

```
Stored tuple                                             Human-readable relationship
───────────────────────────────────────────────────────────────────────────────
(team:platformTeam,       member,    user:alice)         alice──[member]──►team
(workspace:productWS,     editor,    team:platTeam#mbr)  team─[editor via #member]──►workspace
(workspace:productWS,     viewer,    user:bob)           bob──[viewer]──►workspace
(document:roadmapDoc,     workspace, workspace:productWS)  doc──[workspace]──►workspace
```

Drawn as one picture:

```

The picture is for human explanation and draws Alice outward toward the resource.
The evaluator itself starts at the requested object and follows stored
object-to-subject edges until it can prove the user is in the resulting set.
Both views describe the same facts; do not mix up the storage order.
user:alice ──[member]──────────────────────► team:platformTeam
                                                     │
                                    [editor] (via team:platformTeam#member)
                                                     │
                                                     ▼
user:bob ──[viewer]────────────────► workspace:productWorkspace
                                                     ▲
                                              [workspace]
                                                     │
                                       document:roadmapDocument
```

The second edge is special.  Instead of pointing to a single user, it points to
`team:platformTeam#member` — a **subject set**.  That means "everyone who has
the `member` relation on `team:platformTeam`".  Right now, that is just alice.
If you added carol to the team, she would automatically get workspace editor
access without any new workspace tuple.

---

## What a permission check is asking

A check question is: **"starting at `<object>`, can I reach `<user>` by
following edges that satisfy `<relation>`?"**

Concretely:

```
Does user:alice have can_edit on document:roadmapDocument?
```

Means: is there a path from `document:roadmapDocument` through the graph that
eventually touches `user:alice`, via relations that together satisfy `can_edit`?

The answer is yes.  The path is:

```
document:roadmapDocument
  ──[workspace]──► workspace:productWorkspace
  ──[editor via team:platformTeam#member]──► team:platformTeam
  ──[member]──► user:alice
```

The evaluator finds this path by traversing the graph.

---

## The traversal algorithm

The evaluator uses **depth-first search (DFS)**: it picks one branch and follows
it all the way to the end before trying another.

For each `(object, relation)` pair it visits, it tries four things:

| Step | Name | What it does |
|---|---|---|
| 1 | Direct lookup | Is there a tuple `(object, relation, user)` in the store? |
| 2 | Subject-set | Is there a tuple `(object, relation, group#rel)` where user is a member of that group? |
| 3 | Rule expansion | Does the permission model say this relation is implied by a stronger one? Recurse. |
| 4 | Workspace inherit | (documents only) Follow the `workspace` pointer to the parent and check there. |

If any step returns `true`, the whole branch is `true`.  If all four are
exhausted, backtrack and try a different branch.  If every branch is exhausted,
the check is denied.

---

## Full walkthrough: alice / can_edit / roadmapDocument

Let's trace every step the evaluator takes.

### Starting point

```
hasRelation(alice, document:roadmapDocument, can_edit)
```

**Step 1 — direct lookup:** Is there a tuple `(document:roadmapDocument, can_edit, user:alice)` in the store?
No.  The four fixture tuples don't include that.

**Step 3 — rule expansion:** Consult `documentRules`:

```go
documentRules["can_edit"] = ["editor"]
```

This says: "`can_edit` is satisfied by anyone who has `editor`".  So recurse
with `editor` instead:

---

### Recursion 1: alice / editor / roadmapDocument

```
hasRelation(alice, document:roadmapDocument, editor)
```

**Step 1:** Is there a tuple `(document:roadmapDocument, editor, user:alice)`? No.

**Step 3:** Consult `documentRules`:

```go
documentRules["editor"] = ["owner"]
```

Recurse with `owner`:

---

### Recursion 2: alice / owner / roadmapDocument (dead end)

```
hasRelation(alice, document:roadmapDocument, owner)
```

**Step 1:** No tuple. **Step 3:** `documentRules["owner"]` is not in the table →
nothing to expand.

**Step 4 — workspace inheritance:** The relation is `owner`, which is one of the
inheritable base relations (`owner`, `editor`, `viewer`).  Follow the
`workspace` tuple on `document:roadmapDocument`:

```
(document:roadmapDocument, workspace, workspace:productWorkspace)
```

Now check: does alice have `owner` on `workspace:productWorkspace`?

```
hasRelation(alice, workspace:productWorkspace, owner)
```

**Step 1:** No tuple. **Step 3:** `workspaceRules["owner"]` is not in the table.
Dead end — return `false`.

Back up to recursion 1 (`editor` on `roadmapDocument`).  The `owner` branch
failed.  Try step 4 for `editor`.

---

### Recursion 3: alice / editor / workspace:productWorkspace (success)

**Step 4 — workspace inheritance for `editor`:** Follow the `workspace` tuple
again.  Check: does alice have `editor` on `workspace:productWorkspace`?

```
hasRelation(alice, workspace:productWorkspace, editor)
```

**Step 1 — direct:** Is there a tuple `(workspace:productWorkspace, editor, user:alice)`?
No direct alice tuple.

**Step 2 — subject-set:** Scan all tuples for `(workspace:productWorkspace, editor, *)`:

```
(workspace:productWorkspace, editor, team:platformTeam#member)
```

Found one.  The subject is `team:platformTeam#member` — that is a subject set
(it contains `#`).  Is alice in it?

```
subjectSetContains(alice, team:platformTeam#member)
  → split: object=team:platformTeam, relation=member
  → hasRelation(alice, team:platformTeam, member)
```

**Step 1 — direct:** Is there a tuple `(team:platformTeam, member, user:alice)`?

```
YES — (team:platformTeam, member, user:alice)  ✓
```

Return `true` all the way back up the call stack.

---

### How the result propagates back

```
(team:platformTeam, member, user:alice)            → true ✓
  subjectSetContains → true ✓
    hasTuple on workspace:productWorkspace/editor  → true ✓
      hasRelation on workspace:productWorkspace/editor → true ✓
        workspace inheritance for document/editor  → true ✓
          hasRelation on document:roadmapDocument/editor → true ✓
            expandByRules: can_edit includes editor → true ✓
              hasRelation on document:roadmapDocument/can_edit → true ✓
```

**Result: allowed.**

---

## The trace output

The evaluator records every step it takes in a `Trace` slice.  For the alice /
`can_edit` / `roadmapDocument` check, the trace looks like this:

```
Check whether user:alice has can_edit on document:roadmapDocument
document:roadmapDocument can_edit includes editor
document:roadmapDocument editor includes owner
document:roadmapDocument owner can inherit owner from workspace:productWorkspace
document:roadmapDocument editor can inherit editor from workspace:productWorkspace
Resolve subject set team:platformTeam#member: does it contain user:alice?
Found direct tuple (team:platformTeam, member, user:alice)
Found subject-set tuple (workspace:productWorkspace, editor, team:platformTeam#member)
Result: allowed
```

Read it top to bottom: each line is one step, in the order the evaluator visited
it.  Notice that lines 3–4 show the failed `owner` branch, and lines 5–8 show
the successful `editor` branch.  The evaluator explored both before finding the
winning path.

You can print the trace yourself from a test:

```go
result, _ := evaluator.Evaluate(ctx, rebac.CheckRequest{
    User:     fixtures.Alice,
    Relation: rebac.RelationDocumentCanEdit,
    Object:   fixtures.RoadmapDocument,
})
for _, line := range result.Trace {
    fmt.Println(line)
}
```

---

## Walkthrough: casey / can_read / roadmapDocument (denied)

Casey has no tuples.  The evaluator exhausts every branch and finds nothing.

```
hasRelation(casey, document:roadmapDocument, can_read)
  step 1: no direct tuple
  step 3: can_read → viewer (documentRules)
    hasRelation(casey, document:roadmapDocument, viewer)
      step 1: no direct tuple
      step 3: viewer → editor → owner (documentRules, chained)
        ... all return false (no tuples for casey on roadmapDocument)
      step 4: workspace inherit for viewer
        hasRelation(casey, workspace:productWorkspace, viewer)
          step 1: no direct tuple
          step 3: viewer → editor → owner (workspaceRules, chained)
            ... all return false
          → false
        → false
      → false
    → false
  → false
→ false
```

The last trace line is: `Result: denied`.

---

## Subject sets explained

A **subject set** is a tuple whose "user" field is `object#relation` instead of
`user:someone`.  Example:

```
(workspace:productWorkspace, editor, team:platformTeam#member)
```

It means: "the `editor` relation on `productWorkspace` is held by *all members*
of `platformTeam`."

When the evaluator sees a subject set in step 2, it splits the string at `#` and
asks: "does the user hold `member` on `team:platformTeam`?"  That is just
another call to `hasRelation` — the same algorithm, applied to the team.

This is powerful because a single tuple grants access to a whole group.  Add a
new member to the team and they instantly have workspace editor access — no new
workspace tuple needed.

---

## Cycle detection

What happens if the graph has a loop?  For example:

```
(document:cyclicDoc, workspace, document:cyclicDoc)   ← points to itself
```

That tuple is intentionally malformed and would be rejected by
`ValidateTuple`; the cycle test inserts it directly into the low-level store to
prove the traversal remains safe even if corrupted data bypasses normal writes.

Without a guard, `hasRelation` would recurse forever:

```
hasRelation(bob, document:cyclicDoc, can_read)
  → workspace inherit → hasRelation(bob, document:cyclicDoc, viewer)
      → workspace inherit → hasRelation(bob, document:cyclicDoc, viewer)
          → ... forever
```

The **active-path set** prevents this. At the start of every `hasRelation` call,
the evaluator checks whether the `(object, relation)` pair is already in the
current recursion chain:

```go
visitKey := relationVisit{object: object, relation: relation}
if r.visiting[visitKey] {
    // This pair is already active — stop the cycle.
    return false
}
r.visiting[visitKey] = true
defer delete(r.visiting, visitKey)
```

The second time `hasRelation(bob, document:cyclicDoc, viewer)` is called, the
pair is already active, so it returns `false` immediately instead of recursing
again. When a call returns, `defer delete` removes its pair. That allows a
different branch to revisit the same graph node without being incorrectly
denied.

---

## The permission model rules

`internal/authz/model.go` holds three tables—one per object type. Each table maps
a relation to the *stronger* relations that imply it.

```
workspaceRules["viewer"] = ["editor"]   → viewer is satisfied by editor
workspaceRules["editor"] = ["owner"]    → editor is satisfied by owner
```

These are not tuples — they are schema rules.  Tuples say "alice is an editor
of productWorkspace".  Rules say "editors are also viewers".

The evaluator consults the rules in step 3.  It looks up the current relation,
then recurses for each stronger relation that could satisfy it.  If a stronger
relation is found, the weaker one is satisfied automatically.

```
Check "viewer" on workspace:productWorkspace for alice:
  workspaceRules["viewer"] = ["editor"]
  → check "editor" instead
    workspaceRules["editor"] = ["owner"]
    → check "owner" instead
      (no tuple, no rules) → false
    hasTuple "editor": found via team subject-set → true ✓
  → true ✓ (editor satisfies viewer)
```

---

## How the code maps to these steps

| Concept | Code location |
|---|---|
| Entry point for a check | `GraphEvaluator.Evaluate()` (builds a per-request `resolution`) |
| The recursive traversal | `resolution.hasRelation()` |
| Step 1: direct lookup | `hasTuple()` — first `if` block |
| Step 2: subject-set | `hasTuple()` — the `for` loop |
| Subject-set recursion | `subjectSetContains()` |
| Step 3: rule expansion | `expandByRules()` |
| The rule tables | `internal/authz/model.go` |
| Step 4: workspace inherit | `expandDocument()` — the second `if` block |
| Cycle detection | `hasRelation()` — the `visiting` block at the top |
| Depth + cancellation guard | `hasRelation()` — the `depth`/`ctx.Err()` checks at the top |
| Trace output | `r.trace = append(r.trace, ...)` calls scattered through all functions |

---

## Exercise: add a new permission

Add a `can_share` permission: only document owners can share.

**1. Add the relation constant** in `internal/rebac/rebac.go`:

```go
RelationDocumentCanShare Relation = "can_share"
```

**2. Add the rule** in `internal/authz/model.go`:

```go
var documentRules = impliedBy{
    // ... existing rules ...
    rebac.RelationDocumentCanShare: {rebac.RelationDocumentOwner},
}
```

**3. Add the relation to model validation** in `internal/authz/validate.go`.
`relationDefinedFor` is the in-process guard that rejects unknown checks and
tuple writes. Computed permissions such as `can_share` belong in
`relationDefinedFor`, but remain unwritable through `isComputedRelation`.

**4. Mirror the rule in OpenFGA** in `deployments/openfga/model.fga`:

```text
define can_share: owner
```

**5. Add a test** in `internal/authz/evaluator_test.go`:

```go
func TestGraphEvaluator_OnlyOwnerCanShare(t *testing.T) {
    // Make alice a direct owner of the document
    extra := rebac.Tuple(
        fixtures.RoadmapDocument,
        rebac.RelationDocumentOwner,
        rebac.Subject(fixtures.Alice),
    )
    ev := newEvaluator(extra)
    ctx := context.Background()

    // alice (owner) can share
    got, _ := ev.Evaluate(ctx, rebac.CheckRequest{
        User:     fixtures.Alice,
        Relation: rebac.RelationDocumentCanShare,
        Object:   fixtures.RoadmapDocument,
    })
    if !got.Allowed {
        t.Error("expected owner can_share=true")
    }

    // bob (viewer) cannot share
    got, _ = ev.Evaluate(ctx, rebac.CheckRequest{
        User:     fixtures.Bob,
        Relation: rebac.RelationDocumentCanShare,
        Object:   fixtures.RoadmapDocument,
    })
    if got.Allowed {
        t.Error("expected viewer can_share=false")
    }
}
```

No changes to the traversal algorithm are needed—the rule tables drive it. The
extra model and validation edits are important because this repository keeps a
teaching evaluator and an OpenFGA model intentionally aligned.

Next: choose `20-go-language-guide.md` and `21-go-rebac-implementation.md` to
study the Go design, or docs 26 and 34 to replace the teaching evaluator with
OpenFGA.
