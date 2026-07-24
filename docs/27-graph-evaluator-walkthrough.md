# How the graph evaluator works

This document explains, step by step, how the in-process graph evaluator
answers a permission check. Chapter 07 explains how this teaching implementation
fits into a production ReBAC service and an OpenFGA adoption decision.

The relevant source files are:

- `internal/authz/evaluator.go` — the traversal algorithm
- `internal/authz/model.go` — the rule tables
- `internal/authz/store.go` — the in-memory relationship store

---

## What a graph is

A **graph** is a set of **nodes** connected by **edges**.

That is the whole definition.  In our system:

- **Nodes** are entities: a user, a team, a workspace, a document.
- **Edges** are relationships: named connections between two nodes.

A relationship like:

```
user:alice  member  team:platformTeam
```

reads: "`user:alice` is a `member` of `team:platformTeam`."

Domain diagrams use `(subject, relation, resource)`:

```text
user:alice --member of--> team:platformTeam
```

The evaluator searches from the requested resource toward matching subjects.
That reverse lookup direction is an algorithm detail, not a second relationship
convention.

---

## The four fixture relationships as a graph

The project seeds four relationships in
`(subject, relation, resource)` order:

```
Subject                       Relation   Resource
─────────────────────────────────────────────────────────────────────
user:alice                    member     team:platformTeam
team:platformTeam#member      editor     workspace:productWorkspace
user:bob                      viewer     workspace:productWorkspace
workspace:productWorkspace    workspace  document:roadmapDocument
```

Drawn as one picture:

```text
user:alice ──member of──► team:platformTeam

team:platformTeam#member
  └─editor of─► workspace:productWorkspace ◄─viewer of── user:bob
                  │
                  └─workspace of─► document:roadmapDocument
```

The second relationship is special. Its subject is
`team:platformTeam#member`—a **subject set**—instead of one concrete user. It
means "everyone who has the `member` relation on `team:platformTeam`". Right
now, that is just Alice. If you added Carol to the team, she would automatically
get workspace editor access without any new workspace relationship.

---

## What a permission check is asking

A check question is: **"does `<subject>` have `<permission>` on
`<resource>`?"**

Concretely:

```
Does user:alice have can_edit on document:roadmapDocument?
```

It means: do the stored relationships and model rules place `user:alice` in
`document:roadmapDocument#can_edit`?

The answer is yes.  The path is:

```
user:alice
  ──[member of]──► team:platformTeam

team:platformTeam#member
  ──[editor of]──► workspace:productWorkspace
  ──[workspace of]──► document:roadmapDocument
```

The evaluator maps `can_edit` to the `editor` relation required by policy, then
proves the relationship chain by searching in reverse from the requested
document toward candidate subjects.

More precisely, it evaluates whether Alice belongs to
the `can_edit` permission on `document:roadmapDocument`. It does not perform
unrestricted graph reachability: every recursive branch comes from the
permission mapping and relation model.

---

## The traversal algorithm

The evaluator uses **depth-first search (DFS)**: it picks one branch and follows
it all the way to the end before trying another.

Before traversal, the evaluator maps the checked permission to a relation. For
each `(resource, relation)` pair it visits, it tries four things:

| Step | Name | What it does |
|---|---|---|
| 1 | Direct lookup | Is there a relationship `(subject, relation, resource)` in the store? |
| 2 | Subject-set | Is there a relationship `(group#rel, relation, resource)` where the subject is a member of that group? |
| 3 | Rule expansion | Does the policy model say this relation is implied by a stronger one? Recurse. |
| 4 | Workspace inherit | (documents only) Follow the `workspace` pointer to the parent and check there. |

Stored relationships use `(subject, relation, resource)` throughout the domain.
The OpenFGA adapter alone translates them to `user/relation/object`.

If any step returns `true`, the whole branch is `true`.  If all four are
exhausted, backtrack and try a different branch.  If every branch is exhausted,
the check is denied.

---

## Full walkthrough: alice / can_edit / roadmapDocument

Let's trace every step the evaluator takes.

### Starting point

```text
Check(alice, can_edit, document:roadmapDocument)
```

**Permission mapping:** `permissionRelationsFor` maps the `can_edit` permission
to the document's `editor` relation:

```text
can_edit -> editor
```

Traversal therefore begins at:

---

### Recursion 1: alice / editor / roadmapDocument

```
hasRelation(alice, document:roadmapDocument, editor)
```

**Step 1:** Is there a relationship
`(user:alice, editor, document:roadmapDocument)`? No.

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

**Step 1:** No relationship. **Step 3:** `documentRules["owner"]` is not in the table →
nothing to expand.

**Step 4 — workspace inheritance:** The relation is `owner`, which is one of the
inheritable base relations (`owner`, `editor`, `viewer`).  Follow the
`workspace` relationship on `document:roadmapDocument`:

```text
workspace:productWorkspace  workspace  document:roadmapDocument
```

Now check: does alice have `owner` on `workspace:productWorkspace`?

```
hasRelation(alice, workspace:productWorkspace, owner)
```

**Step 1:** No relationship. **Step 3:** `workspaceRules["owner"]` is not in the table.
Dead end — return `false`.

Back up to recursion 1 (`editor` on `roadmapDocument`).  The `owner` branch
failed.  Try step 4 for `editor`.

---

### Recursion 3: alice / editor / workspace:productWorkspace (success)

**Step 4 — workspace inheritance for `editor`:** Follow the `workspace` relationship
again.  Check: does alice have `editor` on `workspace:productWorkspace`?

```
hasRelation(alice, workspace:productWorkspace, editor)
```

**Step 1 — direct:** Is there a relationship
`(user:alice, editor, workspace:productWorkspace)`?
No direct Alice relationship.

**Step 2 — subject-set:** Scan relationships for the requested resource and relation:

```text
team:platformTeam#member  editor  workspace:productWorkspace
```

Found one.  The subject is `team:platformTeam#member` — that is a subject set
(it contains `#`).  Is alice in it?

```
subjectSetContains(alice, team:platformTeam#member)
  → split: resource=team:platformTeam, relation=member
  → hasRelation(alice, team:platformTeam, member)
```

**Step 1 — direct:** Is this relationship present?

```text
YES — user:alice  member  team:platformTeam  ✓
```

Return `true` all the way back up the call stack.

---

### How the result propagates back

```text
user:alice member team:platformTeam                → true ✓
  subjectSetContains → true ✓
    hasRelationship on workspace:productWorkspace/editor  → true ✓
      hasRelation on workspace:productWorkspace/editor → true ✓
        workspace inheritance for document/editor  → true ✓
          hasRelation on document:roadmapDocument/editor → true ✓
            permission can_edit requires editor → true ✓
```

**Result: allowed.**

---

## The trace output

The evaluator records every step it takes in a `Trace` slice. For the Alice /
`can_edit` / `roadmapDocument` check, the trace looks like this:

```
Check whether user:alice has permission can_edit on document:roadmapDocument
Permission can_edit requires relation editor
document:roadmapDocument editor includes owner
document:roadmapDocument owner can inherit owner from workspace:productWorkspace
document:roadmapDocument editor can inherit editor from workspace:productWorkspace
Resolve subject set team:platformTeam#member: does it contain user:alice?
Found direct relationship (user:alice, member, team:platformTeam)
Found subject-set relationship (team:platformTeam#member, editor, workspace:productWorkspace)
Result: allowed
```

Read it top to bottom: each line is one step, in the order the evaluator visited
it.  Notice that lines 3–4 show the failed `owner` branch, and lines 5–8 show
the successful `editor` branch.  The evaluator explored both before finding the
winning path.

You can print the trace yourself from a test:

```go
result, _ := evaluator.Evaluate(ctx, rebac.CheckRequest{
    Subject:    fixtures.Alice,
    Permission: rebac.PermissionDocumentEdit,
    Resource:   fixtures.RoadmapDocument,
})
for _, line := range result.Trace {
    fmt.Println(line)
}
```

---

## Walkthrough: casey / can_read / roadmapDocument (denied)

Casey has no relationships. The evaluator exhausts every branch and finds nothing.

```
permission can_read requires relation viewer
hasRelation(casey, document:roadmapDocument, viewer)
  step 1: no direct relationship
  step 3: viewer → editor → owner (documentRules, chained)
    ... all return false (no relationships for Casey on roadmapDocument)
  step 4: workspace inherit for viewer
    hasRelation(casey, workspace:productWorkspace, viewer)
      step 1: no direct relationship
      step 3: viewer → editor → owner (workspaceRules, chained)
        ... all return false
      → false
    → false
  → false
→ false
```

The last trace line is: `Result: denied`.

---

## Subject sets explained

A **subject set** is a relationship whose subject is `resource#relation` instead of
`user:someone`.  Example:

```text
team:platformTeam#member  editor  workspace:productWorkspace
```

It means: "the `editor` relation on `productWorkspace` is held by *all members*
of `platformTeam`."

When the evaluator sees a subject set in step 2, it splits the string at `#` and
asks: "does the user hold `member` on `team:platformTeam`?"  That is just
another call to `hasRelation` — the same algorithm, applied to the team.

This is powerful because a single relationship grants access to a whole group. Add a
new member to the team and they instantly have workspace editor access — no new
workspace relationship needed.

---

## Cycle detection

What happens if the graph has a loop?  For example:

```
(team:a, member, team:b#member)
(team:b, member, team:a#member)   ← points back to the first userset
```

Those relationships are intentionally invalid for this repository's model—team
membership accepts users, not nested team usersets—and `ValidateRelationship` would
reject them. The cycle test inserts them directly into the low-level store to
prove the traversal remains safe even if corrupted data bypasses normal writes
or a future model introduces recursion.

Without a guard, `hasRelation` would recurse forever:

```
hasRelation(casey, team:a, member)
  → resolve team:b#member → hasRelation(casey, team:b, member)
      → resolve team:a#member → hasRelation(casey, team:a, member)
          → ... forever
```

The **active-path set** prevents this. At the start of every `hasRelation` call,
the evaluator checks whether the `(resource, relation)` pair is already in the
current recursion chain:

```go
visitKey := relationVisit{resource: resource, relation: relation}
if r.visiting[visitKey] {
    // This pair is already active — stop the cycle.
    return false
}
r.visiting[visitKey] = true
defer delete(r.visiting, visitKey)
```

The second `hasRelation(casey, team:a, member)` call finds the pair already
active, so it returns `false` immediately instead of recursing again. When a
call returns, `defer delete` removes its pair. That allows a different branch to
revisit the same graph node without being incorrectly denied.

---

## The permission model rules

`internal/authz/model.go` holds a permission-to-relation mapping and three
relation-hierarchy tables—one per resource type. Each hierarchy table maps a
relation to the *stronger* relations that imply it.

```
workspaceRules["viewer"] = ["editor"]   → viewer is satisfied by editor
workspaceRules["editor"] = ["owner"]    → editor is satisfied by owner
```

These are not relationships—they are schema rules. Relationships say "Alice is an editor
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
      (no relationship, no rules) → false
    hasRelationship "editor": found via team subject-set → true ✓
  → true ✓ (editor satisfies viewer)
```

---

## How the code maps to these steps

| Concept | Code location |
|---|---|
| Entry point for a check | `GraphEvaluator.Evaluate()` (builds a per-request `resolution`) |
| The recursive traversal | `resolution.hasRelation()` |
| Permission-to-relation mapping | `permissionRelationsFor()` |
| Step 1: direct lookup | `hasRelationship()` — direct `store.Has` lookup |
| Step 2: subject-set | `hasRelationship()` — the candidate loop |
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

**1. Add the permission constant** in `internal/rebac/rebac.go`:

```go
PermissionDocumentShare Permission = "can_share"
```

**2. Add the permission mapping** in `internal/authz/model.go`:

```go
var permissionRules = map[rebac.ResourceType]map[rebac.Permission][]rebac.Relation{
    // ... existing rules ...
    rebac.ResourceTypeDocument: {
        rebac.PermissionDocumentShare: {rebac.RelationDocumentOwner},
    },
}
```

`permissionDefinedFor` reads this mapping, while relationship validation accepts
only relations, so `can_share` remains unwritable by construction.

**3. Mirror the rule in OpenFGA** in `deployments/openfga/model.fga`:

```text
define can_share: owner
```

**4. Add a test** in `internal/authz/evaluator_test.go`:

```go
func TestGraphEvaluator_OnlyOwnerCanShare(t *testing.T) {
    // Make alice a direct owner of the document
    extra := rebac.NewRelationship(
        rebac.Subject(fixtures.Alice),
        rebac.RelationDocumentOwner,
        fixtures.RoadmapDocument,
    )
    relationships := append(fixtures.SeedRelationships(), extra)
    store := authz.NewInMemoryStore(relationships...)
    ev := authz.NewGraphEvaluator(store)
    ctx := t.Context()

    // alice (owner) can share
    got, err := ev.Evaluate(ctx, rebac.CheckRequest{
        Subject:    fixtures.Alice,
        Permission: rebac.PermissionDocumentShare,
        Resource:   fixtures.RoadmapDocument,
    })
    if err != nil {
        t.Fatalf("owner check: %v", err)
    }
    if !got.Allowed {
        t.Error("expected owner can_share=true")
    }

    // bob (viewer) cannot share
    got, err = ev.Evaluate(ctx, rebac.CheckRequest{
        Subject:    fixtures.Bob,
        Permission: rebac.PermissionDocumentShare,
        Resource:   fixtures.RoadmapDocument,
    })
    if err != nil {
        t.Fatalf("viewer check: %v", err)
    }
    if got.Allowed {
        t.Error("expected viewer can_share=false")
    }
}
```

No changes to the traversal algorithm are needed—the permission mapping and
relation tables drive it. The model and validation edits are important because this repository keeps a
teaching evaluator and an OpenFGA model intentionally aligned.

Next: choose `20-go-language-guide.md` and `21-go-rebac-implementation.md` to
study the Go design, or read docs 26 and 34 for the staged path from this
teaching evaluator to OpenFGA.
