# Start Here

You do **not** need to read all the docs or all the code. This page is the
on-ramp. The rest of the repo is a reference library.

## The One Sentence

> Alice can edit the roadmap document **because** she is in the platform team,
> which is an editor of the product workspace, which the document lives in.

That sentence is the entire system. ReBAC makes the computer prove it by walking
a graph of relationships.

```text
user:alice --member--> team:platformTeam --editor--> workspace:productWorkspace <--workspace-- document:roadmapDocument
```

## Six Docs, In Order

| # | Doc | What you get |
|---|---|---|
| 1 | `docs/01-oauth-authentication.md` | who is this user? |
| 2 | `docs/02-authorization-fundamentals.md` | what may they do? |
| 3 | `docs/03-graph-theory-for-rebac.md` | nodes, edges, paths |
| 4 | `docs/04-rebac-concepts.md` | tuples, subject sets, checks |
| 5 | `docs/05-openfga-model.md` | the policy model as schema |
| 6 | `docs/27-graph-evaluator-walkthrough.md` | the evaluator, line by line |

Everything else is optional depth: OpenFGA migration, Docker, production
readiness, and Go examples.

## One File To Read

Open this with `docs/27` beside it:

```text
internal/authz/evaluator.go
```

If you understand `hasRelation` and its four steps, you understand the core
ReBAC algorithm.

## Three Commands

```bash

go test ./...
go test -v -run TestTrace ./internal/authz
go test -v -run TestGraphEvaluator_TeamMemberCanEditDocument ./internal/authz
```

`TestTrace` prints every step the evaluator took. For `alice / can_edit`, the
successful path is:

```text
document -> workspace -> team#member -> user:alice
```

## How To Study

1. Run the trace test.
2. Open `internal/fixtures/fixtures.go`.
3. Change one tuple.
4. Predict which checks change.
5. Run the trace test again.

That predict-then-check loop teaches faster than passive reading.
