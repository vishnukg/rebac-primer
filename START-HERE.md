# Start here (Go track)

You do **not** need to read all the docs or all the code. This one page is the
whole on-ramp. The rest of the repo is a reference library — look things up when
a need arises, not before.

## The one sentence to understand

> Alice can edit the roadmap document **because** she is in the platform team,
> which is an editor of the product workspace, which the document lives in.

That sentence is the entire system. ReBAC just makes a computer prove that
sentence (or fail to) by walking a graph of relationships.

```
user:alice --member--> team:platformTeam --editor--> workspace:productWorkspace <--workspace-- document:roadmapDocument
```

## 6 docs, in order (skip everything else for now)

| # | Doc | What you get |
|---|-----|--------------|
| 1 | `docs/01-oauth-authentication.md` | who is this user? (authn) |
| 2 | `docs/02-authorization-fundamentals.md` | what may they do? RBAC vs ReBAC |
| 3 | `docs/03-graph-theory-for-rebac.md` | nodes, edges, paths (just enough) |
| 4 | `docs/04-rebac-concepts.md` | tuples, subject sets, the model |
| 5 | `docs/05-openfga-model.md` | the permission model as a schema |
| 6 | `docs/27-graph-evaluator-walkthrough.md` | the algorithm, line by line |

Everything else — the TypeScript track, concurrency (22), generics (23),
interfaces (24), Docker (30–33), OpenFGA backend (26, 34), production (40) — is
**optional depth**. Not prerequisites. Don't let their existence stress you.

## 1 file of code to read

Open this with `docs/27` beside it. If you understand `hasRelation` and its four
steps, you understand ReBAC:

```
go/internal/authz/adapters/graph/evaluator.go
```

Ignore `parallel.go`, `result.go`, `middleware.go`, the `openfga/` adapter, and
the HTTP layer on your first pass. They are peripheral.

## 3 commands to run

```bash
cd go

# 1. Prove the setup works
go test ./...

# 2. WATCH the algorithm think (the trace program — see below)
go test -v -run TestTrace ./internal/authz/adapters/graph/

# 3. Run one specific check and read its trace
go test -v -run TestGraphEvaluator_TeamMemberCanEditDocument ./internal/authz/adapters/graph/
```

## The trace program

`go/internal/authz/adapters/graph/trace_example_test.go` runs four checks and
prints every step the evaluator took. Run command #2 above and read the output
top to bottom. For `alice / can_edit`:

```
[0] Check whether user:alice has can_edit on document:roadmapDocument
[1] document:roadmapDocument can_edit includes editor      <- can_edit needs editor
[2] document:roadmapDocument editor includes owner         <- tries owner first (fails)
[3] ...owner can inherit owner from workspace...           <- dead-end branch
[4] ...editor can inherit editor from workspace...         <- the winning branch
[5] Resolve subject set team:platformTeam#member: contains user:alice?
[6] Found direct tuple (team:platformTeam, member, user:alice)   <- the leaf
[7] Found subject-set tuple (workspace..., editor, team...#member)
[8] Result: allowed
```

Each line is one recursive step. Lines 2–3 are a branch that fails; lines 4–8 are
the branch that succeeds. Watching it explore-and-backtrack is the whole lesson.
Try the denied cases too (Bob can't edit, Casey can't read) and notice the
`Already evaluated ...; stop this branch` lines — that is the cycle guard.

## How to study (don't read passively)

1. Run the trace program. Read one trace fully.
2. Open `go/internal/fixtures/fixtures.go`. Delete or change one tuple.
3. **Predict** which checks change before re-running.
4. Run the trace program again and see if you were right.

That predict-then-check loop teaches faster than re-reading notes.

## Today's only goal

Be able to say the one sentence at the top out loud, and point to where each hop
shows up in the trace. If you can do that, today succeeded. Stop there.

## Shaky on the background concepts?

If graph theory or OpenFGA's DSL feel unfamiliar, read `notes-graphs-and-openfga.md`
first — it's a one-page, beginner-level distillation of both, using this exact
example. Then the 6 docs above read much faster.

## When you're ready for more

Go back to `docs/00-course-map.md` and follow the **Go path**. By then it will
feel like a menu, not a mountain.
