# Start Here

You do **not** need to read all the docs or all the code. This page is the
on-ramp. The rest of the repo is a reference library.

The on-ramp does not replace or abbreviate that library. It only controls when
new ideas are introduced. Every detailed chapter, comparison, implementation
walkthrough, experiment, and production topic remains available through the
[complete course map](docs/00-course-map.md).

## If You Feel Overwhelmed

Do only this first session:

1. Read “The One Sentence” and “Keep It In Your Head” below.
2. Run `make trace` (or `go test -v -run TestTrace ./internal/authz` with local
   Go).
3. Read [the mental-model chapter](docs/rebac-mental-model.md) only through
   “Walk The Alice Decision Backward.”
4. Explain these three facts aloud:
   - a relationship is a changing product fact
   - the model contains reusable rules
   - a check asks whether one subject may perform one action on one resource
5. Stop.

That is enough for one session. Defer—not discard—these topics:

```text
OAuth/OIDC
production migration
Docker operations
concurrency and generics
policy-engine comparisons
```

Those topics remain part of the full course. They matter later, but none is
required to understand why Alice can edit the roadmap document. The course map
tells you when to bring each one back.

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

The arrows show the domain relationship convention:

```text
subject --relation--> resource
```

The Go `Relationship` struct uses those same domain names: `Subject`,
`Relation`, and `Resource`. The OpenFGA adapter translates them to that API's
`user`, `relation`, and `object` fields.

## Keep It In Your Head

Use this three-layer picture:

```text
model + relationships + check
  -> decision (allow or deny), when evaluation succeeds
  -> evaluation error, when it cannot decide
```

- The **model** is the reusable grammar: owners are editors, editors are
  viewers, and document editors can come from the parent workspace.
- **Relationships** are changing product facts: Alice belongs to this team, this team
  edits that workspace, and this document belongs to that workspace.
- A **check** asks whether one subject may perform one action on one resource:
  `Check(subject, action, resource)`.

Read every relationship as one sentence:

```text
<subject> is a <relation> of <resource>
```

The only unusual subject is `team:platformTeam#member`. It means a set—“all
members of the platform team”—rather than one person. The evaluator starts at
the requested action and works backward through allowed rules and facts. If
it cannot prove a path to the subject, the answer is deny.

Authentication supplies “who is asking.” ReBAC answers “may that identity do
this action to this resource?” Keep those as two separate questions.

The stable vocabulary is: actions are attempted by the application, relations
name policy associations, relationships record durable facts, and policy
derives actions such as `can_edit`. Application code checks actions. See
[Authorization Domain Language](docs/authorization-domain-language.md).

The full version of this memory aid—including sets, usersets, roles versus
actions, debugging, and modeling at work—is
[A ReBAC Mental Model You Can Reuse](docs/rebac-mental-model.md).

## Before You Begin

Choose one toolchain:

- Docker Desktop or another working Docker engine, then use the `make` commands
  throughout the course.
- Go 1.26.5 locally, then run the equivalent `go` commands directly.

Check the Docker path with:

```bash
docker version
make test
```

The optional OpenFGA exercises additionally require the `fga` CLI and `jq` on
your host; the migration chapter lists the setup check.

## Choose Your Route Later

You do not need the same route as every other reader.
Do not read files in numeric order; the numbers group related topics, while the
routes below define the learning order.

### Full route: understand ReBAC

If graphs and OpenFGA are completely new, the optional
[graph and OpenFGA notes](notes-graphs-and-openfga.md) provide a short preview.

1. [Reusable ReBAC mental model](docs/rebac-mental-model.md)
2. [Authorization fundamentals](docs/02-authorization-fundamentals.md)
3. [Graph theory for ReBAC](docs/03-graph-theory-for-rebac.md)
4. [ReBAC concepts](docs/04-rebac-concepts.md)
5. [OpenFGA model](docs/05-openfga-model.md)
6. [Designing a ReBAC authorization service](docs/07-rebac-authorization-service-design.md)
7. [Policy-based authorization](docs/08-policy-based-authorization.md)
8. [Graph evaluator walkthrough](docs/27-graph-evaluator-walkthrough.md)

### Go route: understand the implementation

If Go is new to you, start with the self-contained language foundation:

1. [Go learning path and practice plan](docs/09-go-learning-path.md)
2. [Toolchain and core syntax](docs/10-go-toolchain-and-syntax.md)
3. [Values, pointers, collections, and methods](docs/11-go-values-pointers-and-methods.md)
4. [Errors, interfaces, packages, and testing](docs/12-go-errors-interfaces-and-testing.md)
5. [HTTP, JSON, context, and application lifecycle](docs/13-go-http-json-and-context.md)
6. [Go idioms and patterns](docs/14-go-idioms-and-patterns.md)

Then read the full ReBAC route above and continue with:

1. [Go language guide for this repository](docs/20-go-language-guide.md)
2. [Architecture](docs/06-architecture.md)
3. [Go ReBAC implementation](docs/21-go-rebac-implementation.md)
4. [Go concurrency](docs/22-go-concurrency.md)
5. [Go generics](docs/23-go-generics.md)
6. [Go interfaces and embedding](docs/24-go-interfaces-embedding.md)
7. [Go testing](docs/25-go-testing.md)
8. [Go authz call flow](docs/28-go-authz-call-flow.md)
9. [Guided feature lab](docs/29-go-guided-feature-lab.md)

If you already write Go, skip chapters 10–13 and begin at the repository
language guide. If your goal is specifically to learn Go idioms, still read
chapters 09, 14, and 22-25.

### Production route: understand the boundaries

Start with the [reusable mental model](docs/rebac-mental-model.md), then read
[Designing a ReBAC authorization service](docs/07-rebac-authorization-service-design.md),
[OAuth and OIDC](docs/01-oauth-authentication.md), the staged
[in-process-to-OpenFGA migration](docs/26-openfga-migration.md),
[the OpenFGA adapter](docs/34-openfga-adapter-walkthrough.md), and the final
[production gates](docs/40-production-readiness.md). The OAuth chapter is
intentionally substantial; its "core path" markers tell you where a first
reading can stop.

## Four Files, In This Order

Ignore the rest of the code on your first pass:

1. `internal/fixtures/fixtures.go` — the four changing relationship facts.
2. `deployments/openfga/model.fga` — the reusable policy rules.
3. `internal/authz/evaluator_test.go` — start with
   `TestGraphEvaluator_TeamMemberCanEditDocument` to see the question and answer.
4. `internal/authz/evaluator.go` — read `Evaluate`, then `hasRelation`, with
   [doc 27](docs/27-graph-evaluator-walkthrough.md) beside it.

Do not try to understand every helper on the first pass. If you can explain why
the first evaluator test is allowed, you have understood the core system.

## Three Commands

```bash
make test
make trace
make test-action
```

`TestTrace` prints every step the evaluator took. For `alice / can_edit`, the
successful path is:

```text
user:alice -> team membership -> workspace editor -> document
```

## How To Study

1. Run the trace test.
2. Open `internal/fixtures/fixtures.go`.
3. Change one relationship.
4. Predict which checks change.
5. Run the trace test again.

That predict-then-check loop teaches faster than passive reading.

Every core chapter ends with either an experiment or a checkpoint. Do it before
moving on. ReBAC becomes intuitive when you repeatedly predict an answer and
then ask the evaluator to prove you right or wrong.
