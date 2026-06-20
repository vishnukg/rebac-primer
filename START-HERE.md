# Start Here

You do **not** need to read all the docs or all the code. This page is the
on-ramp. The rest of the repo is a reference library.

## The One Sentence

> Alice can edit the roadmap document **because** she is in the platform team,
> which is an editor of the product workspace, which the document lives in.

That sentence is the entire system. ReBAC makes the computer prove it by walking
a graph of relationships.

```text
user:alice
  --member of--> team:platformTeam

team:platformTeam#member
  --editor of--> workspace:productWorkspace

workspace:productWorkspace
  --workspace of--> document:roadmapDocument
```

The arrows use the OpenFGA tuple convention:

```text
subject --relation--> object
```

The Go `TupleKey` struct lists its fields as `Object`, `Relation`, `User`, but
that is an internal field order—not a different relationship.

## Before You Begin

Choose one toolchain:

- Docker Desktop or another working Docker engine, then use the `make` commands
  throughout the course.
- Go 1.26.4 locally, then run the equivalent `go` commands directly.

Check the Docker path with:

```bash
docker version
make test
```

The optional OpenFGA exercises additionally require the `fga` CLI and `jq` on
your host; the migration chapter lists the setup check.

## Choose Your Route

You do not need the same route as every other reader.
Do not read files in numeric order; the numbers group related topics, while the
routes below define the learning order.

### Fast route: understand ReBAC

If graphs and OpenFGA are completely new, the optional
[graph and OpenFGA notes](notes-graphs-and-openfga.md) provide a short preview.

1. [Authorization fundamentals](docs/02-authorization-fundamentals.md)
2. [Graph theory for ReBAC](docs/03-graph-theory-for-rebac.md)
3. [ReBAC concepts](docs/04-rebac-concepts.md)
4. [OpenFGA model](docs/05-openfga-model.md)
5. [Designing a ReBAC authorization service](docs/07-rebac-authorization-service-design.md)
6. [Graph evaluator walkthrough](docs/27-graph-evaluator-walkthrough.md)

### Go route: understand the implementation

If Go is new to you, start with the self-contained language foundation:

1. [Toolchain and core syntax](docs/10-go-toolchain-and-syntax.md)
2. [Values, pointers, collections, and methods](docs/11-go-values-pointers-and-methods.md)
3. [Errors, interfaces, packages, and testing](docs/12-go-errors-interfaces-and-testing.md)
4. [HTTP, JSON, context, and application lifecycle](docs/13-go-http-json-and-context.md)

Then read the fast ReBAC route and continue with:

1. [Go language guide for this repository](docs/20-go-language-guide.md)
2. [Architecture](docs/06-architecture.md)
3. [Go ReBAC implementation](docs/21-go-rebac-implementation.md)
4. [Go authz call flow](docs/28-go-authz-call-flow.md)
5. [Go testing](docs/25-go-testing.md)
6. [Guided feature lab](docs/29-go-guided-feature-lab.md)

If you already write Go, skip chapters 10–13 and begin at the repository
language guide.

### Production route: understand the boundaries

Read [Designing a ReBAC authorization service](docs/07-rebac-authorization-service-design.md),
then [OAuth and OIDC](docs/01-oauth-authentication.md),
[migration](docs/26-openfga-migration.md),
[the OpenFGA adapter](docs/34-openfga-adapter-walkthrough.md), and
[production readiness](docs/40-production-readiness.md). The OAuth chapter is
intentionally substantial; its "core path" markers tell you where a first
reading can stop.

## One File To Read

Open this with `docs/27-graph-evaluator-walkthrough.md` beside it:

```text
internal/authz/evaluator.go
```

If you understand `hasRelation` and its four steps, you understand the core
ReBAC algorithm.

## Three Commands

```bash
make test
make trace
make test-permission
```

`TestTrace` prints every step the evaluator took. For `alice / can_edit`, the
successful path is:

```text
user:alice -> team membership -> workspace editor -> document
```

## How To Study

1. Run the trace test.
2. Open `internal/fixtures/fixtures.go`.
3. Change one tuple.
4. Predict which checks change.
5. Run the trace test again.

That predict-then-check loop teaches faster than passive reading.

Every core chapter ends with either an experiment or a checkpoint. Do it before
moving on. ReBAC becomes intuitive when you repeatedly predict an answer and
then ask the evaluator to prove you right or wrong.
