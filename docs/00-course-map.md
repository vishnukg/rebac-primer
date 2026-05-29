# Course map

Welcome to **ReBAC Primer: TypeScript and Go**.

This repo is meant to be more than a code sample. It is a practical ReBAC course
with two implementation tracks: one in TypeScript and one in Go.

You are not reading a manual. You are taking over a small authorization system
and making it easier to understand, test, run, and evolve.

The learning loop is:

```text
read a concept -> inspect real code -> run tests -> change one thing -> explain what changed
```

If a topic does not help you read, write, test, or maintain this codebase, it is
kept out of the main path.

## How each chapter works

Each tutorial chapter is meant to feel like a guided lab:

- **Scene**: the practical problem the chapter is solving
- **Concept**: the TypeScript, Go, ReBAC, Node, or Docker idea
- **Code walk**: where the idea appears in this repo
- **Try it**: a small change that gives you feedback
- **Checkpoint**: a quick question to prove the idea landed

If a chapter starts feeling abstract, jump to the code named in the table and
run the command beside it. The repo is the lesson.

## What you will build

The project domain is collaborative documents:

- users belong to teams
- teams get workspace access
- documents belong to workspaces
- document permissions are inherited through a relationship graph
- both implementations keep the authorization vocabulary explicit

The important idea is that ReBAC gives you a precise domain language. TypeScript
and Go encode that language differently, but both implementations answer the
same authorization questions against the same model.

Here is the story you will keep coming back to:

```text
Alice edits the roadmap document because she is in the platform team.
Bob can read the roadmap document but cannot edit it.
Casey has no path through the graph, so access is denied.
```

That tiny cast of users keeps the examples grounded.

## Shared track: authorization and ReBAC

Read these first. They are language-agnostic and give both the TypeScript and
Go tracks the same foundation.

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 01 | OAuth/OIDC authentication fundamentals | conceptual |
| 02 | Authorization fundamentals: RBAC, ABAC, ReBAC, agentic systems | conceptual |
| 03 | Graph theory needed for ReBAC | conceptual |
| 04 | ReBAC concepts, relationship graphs, agentic tool calls | `typescript/src/authz-service/adapters/graph/makeGraphEvaluator.ts`, `go/internal/authz/adapters/graph/evaluator.go` |
| 05 | OpenFGA model DSL | `typescript/src/authz-service/adapters/graph/permissionModel.ts`, `go/internal/authz/adapters/graph/permissionmodel.go` |
| 06 | Architecture: ports & adapters, dependency direction, intentional caveats (both languages) | `go/internal/authz/ports.go`, `go/internal/documents/ports.go`, `typescript/src/documents-service/core/ports` |

## TypeScript track

Read these after the shared ReBAC track if TypeScript is your implementation language.

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 10 | TypeScript mental model, `strict`, project setup | `typescript/tsconfig.json`, `typescript/package.json` |
| 11 | Types, unions, narrowing, template literal types | `typescript/src/shared/rebac.ts` |
| 12 | Functions, modules, factories, interfaces | `typescript/src/documents-service/core/domain`, `typescript/src/authz-service/adapters` |
| 13 | Async TypeScript, errors, and boundaries | `typescript/src/documents-service/core/domain`, `typescript/src/documents-service/adapters/http` |
| 14 | Testing TypeScript with Vitest | `typescript/test/*.test.ts` |
| 15 | Coding style for maintainable TypeScript | `docs/15-typescript-code-style.md` |
| 16 | Node ESM, module loading, module patterns, singletons | `typescript/package.json`, `typescript/tsconfig.json`, `typescript/src/authz-service/index.ts` |

Read these after the shared OpenFGA model chapter.

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 17 | TypeScript AuthZ adapter pattern | `typescript/src/documents-service/adapters/authz/makeAuthzServiceClient.ts`, `typescript/test/fixtures.ts` |
| 18 | TypeScript ReBAC implementation — theory to code walkthrough | `typescript/src/authz-service/adapters/graph/makeGraphEvaluator.ts`, `typescript/src/documents-service/adapters/http` |
| 19 | Factory function pattern — closure-based DI, pattern names, trade-offs | `typescript/src/documents-service/core/domain`, `typescript/src/authz-service/compose.ts` |
| 29 | Authz call flow: tracing a request through every layer (two HTTP services) | `typescript/src/documents-service/adapters/http`, `typescript/src/documents-service/adapters/authz/makeAuthzServiceClient.ts`, `typescript/src/authz-service/adapters/http` |

## Go track

Read these after the shared ReBAC track if Go is your implementation language.

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 20 | Go language primer | `go/internal/shared/rebac.go`, `go/internal/documents/documents.go` |
| 21 | Go ReBAC implementation walkthrough | `go/internal/authz/adapters/graph/evaluator.go`, `go/internal/documents/adapters/http/handler.go` |
| 22 | Go concurrency: goroutines, channels, WaitGroups | `go/internal/authz/adapters/graph/parallel.go` |
| 23 | Go generics: type parameters, constraints, Result[T] | `go/internal/authz/adapters/graph/result.go` |
| 24 | Go interfaces and embedding: decorator pattern | `go/internal/authz/adapters/graph/middleware.go` |
| 25 | Go testing: AAA, table-driven, benchmarks, fuzz | `go/internal/authz/adapters/graph/evaluator_test.go` |
| 26 | From-scratch ReBAC vs OpenFGA: concept mapping and migration guide | `go/internal/authz/adapters/graph/evaluator.go`, `go/internal/authz/adapters/graph/permissionmodel.go` |
| 27 | Graph evaluator deep dive: step-by-step walkthrough for non-graph-theory readers | `go/internal/authz/adapters/graph/evaluator.go` |
| 28 | Authz call flow: tracing a request through every layer (in-process) | `go/cmd/server/main.go`, `go/internal/documents/read.go`, `go/internal/authz/domain.go` |

## Shared track: Docker and local services

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 30 | Docker fundamentals: images, containers, Dockerfile | `typescript/Dockerfile`, `go/Dockerfile` |
| 31 | Docker networking: host ports, service names, Compose DNS | `deployments/docker-compose.yml` |
| 32 | Docker Compose local services | `deployments/docker-compose.yml` |
| 33 | Client/server ReBAC demo | `typescript/src/authz-service/index.ts`, `typescript/src/documents-service/index.ts`, `typescript/src/cli/index.ts`, `go/cmd/server/main.go` |

## Shared track: going to production

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 40 | Production readiness: turning tutorial patterns into a real service | conceptual |

---

## Reading paths

### TypeScript path

Read if TypeScript or Node.js is your primary goal.

```
01 → 02 → 03 → 04 → 05             Authn/authz + ReBAC + OpenFGA model
10 → 11 → 12 → 13 → 14 → 15 → 16   TypeScript language and app structure
17 → 18 → 19                        OpenFGA adapter + theory-to-code walkthrough + factories
29                                  Authz call flow across the two HTTP services
06                                  Architecture synthesis: ports & adapters, dependency direction
30 → 31 → 32 → 33                  Docker + client/server
40                                  Production gaps
```

### Go path (no TypeScript required)

Read if Go is your primary goal. Docs 10-17 are TypeScript-specific and can be
skipped. Docs 20-25 are self-contained Go references that cover the language
concepts directly through the implementation code.

```
01 → 02 → 03 → 04 → 05             Authn/authz + ReBAC + OpenFGA model
20                                  Go language primer
21                                  Go ReBAC implementation
26                                  From-scratch ReBAC vs OpenFGA (optional but recommended)
27                                  Graph evaluator deep dive (read alongside evaluator.go)
28                                  Authz call flow across every layer (request → decision → status)
06                                  Architecture synthesis: ports & adapters, dependency direction
22                                  Concurrency: goroutines and channels
23                                  Generics: Result[T] and Map
24                                  Interfaces and embedding: decorator pattern
25                                  Testing: AAA, table-driven, benchmarks, fuzz
30 → 31 → 32 → 33                  Docker fundamentals + client/server
40                                  Production gaps
```

Start with `make go-test` to confirm the setup works, then open
`go/internal/authz/adapters/graph/evaluator.go` alongside `docs/21-go-rebac-implementation.md`.
Read `docs/27-graph-evaluator-walkthrough.md` for a step-by-step trace of the traversal
algorithm if the evaluator code feels hard to follow.

### Both languages

Read the shared docs first, then either implementation track. Docs 20-25 make TS-to-Go comparisons
at every step, so having the TypeScript mental model first makes the Go docs richer.

---

## Suggested pace

### Day 1: Authn and authz foundations

1. Read `01-oauth-authentication.md`.
2. Read `02-authorization-fundamentals.md`.
3. Explain the difference between authentication and authorization.
4. Explain why global roles are not enough for `document:roadmapDocument`.
5. Explain why an agent should authorize each tool call before execution.

Checkpoint: explain why OAuth/OIDC can identify a user but cannot by itself prove
that the user can edit one specific document.

### Day 2: Graphs and ReBAC

1. Read `03-graph-theory-for-rebac.md`.
2. Read `04-rebac-concepts.md`.
3. Draw the path from `user:alice` to `document:roadmapDocument#can_edit`.
4. Remove one tuple from either fixture file and predict which access check changes.

Checkpoint: explain why Bob can read but cannot edit.

### Day 3: OpenFGA model and the graph evaluator

1. Read `05-openfga-model.md`.
2. Open `typescript/src/authz-service/adapters/graph/permissionModel.ts` and `go/internal/authz/adapters/graph/permissionmodel.go`.
3. Compare the model with the in-memory graph evaluators in both languages.
4. (Go track) Read `docs/27-graph-evaluator-walkthrough.md` — it traces every step of the
   `alice / can_edit / roadmapDocument` check through the actual evaluator code.

Checkpoint: explain `workspace#editor from workspace` as a graph path.

### Day 4: Choose an implementation track

TypeScript:

1. Read `10-typescript-foundations.md` and `11-types-and-values.md`.
2. Run `make ts-build`.
3. Break one relation name in `typescript/src/shared/rebac.ts`.
4. Restore the code and run `make ts-test`.

Go:

1. Read `20-go-primer.md`.
2. Run `make go-test`.
3. Open `go/internal/authz/adapters/graph/evaluator_test.go`.
4. Break one tuple in `go/internal/fixtures/fixtures.go` and predict which tests fail.

Checkpoint: explain the same tuple in both languages.

### Day 5: Service boundaries

TypeScript:

1. Read `12-functions-modules-classes.md` and `13-async-errors-and-boundaries.md`.
2. Read `18-typescript-rebac-implementation.md` — traces a full request through the graph.
3. Inspect `typescript/src/documents-service/core/domain`.
4. Trace how an update request becomes an authorization check.

Go:

1. Read `21-go-rebac-implementation.md`.
2. Open `go/internal/documents/update.go` (and `domain.go` for the shared helpers).
3. Trace how `Update` becomes an authorization check.
4. If the graph traversal is hard to follow, read `docs/27-graph-evaluator-walkthrough.md`
   and step through the `alice / can_edit / roadmapDocument` trace.

Checkpoint: explain why the domain depends on `AuthzClient`, not the graph evaluator directly.

### Day 6: Tests as executable documentation

TypeScript:

1. Read `14-testing-with-vitest.md`.
2. Run `make ts-test`.
3. Change `seedPolicyTuples()` in `typescript/test/fixtures.ts` and predict which tests fail.

Go:

1. Open `go/internal/authz/adapters/graph/evaluator_test.go` — read each AAA section.
2. Run `make go-test`.
3. Change `SeedRelationshipTuples()` in `go/internal/fixtures/fixtures.go` and predict which tests fail.

Checkpoint: explain why Bob can read but cannot edit.

### Day 7: Language-specific depth

TypeScript:

1. Read `16-node-esm-and-module-patterns.md`.
2. Inspect the explicit `.ts` extensions in TypeScript imports.
3. Explain why `typescript/src/authz-service/index.ts` performs actions but `typescript/src/shared/rebac.ts` does not.

Go:

1. Read `22-go-concurrency.md`, `23-go-generics.md`, and `24-go-interfaces-embedding.md`.
2. Inspect `go/internal/authz/adapters/graph/parallel.go`, `go/internal/authz/adapters/graph/result.go`, and `go/internal/authz/adapters/graph/middleware.go`.
3. Run `make go-test` and explain which tests document each language feature.

Checkpoint: explain why relative ESM imports use explicit extensions, or
why Go passes `context.Context` through the authorization boundary.

### Day 8+: Local services and client/server

1. Read `30-docker-fundamentals.md`.
2. Read `31-docker-networking.md`.
3. Read `32-docker-compose-local-services.md`.
4. Start the TypeScript server with `make ts-server` or the Go server with `make go-server`.
5. Run the terminal client with `make ts-client`.

Checkpoint: explain what changes when the app runs on your host versus inside
Docker Compose.

### Day 9+: Production readiness

1. Read `40-production-readiness.md`.
2. For each gap listed, write one sentence describing where in this repo the
   production concern would be handled.
3. Use the checklist to separate learning-only shortcuts from production
   requirements.

Checkpoint: explain why the `AuthzClient` interface makes it straightforward to
swap the in-process evaluator for a real AuthZ service in a production deployment.

## Repo commands

TypeScript (from `typescript/` or via Docker):

```bash
make ts-deps
make ts-build
make ts-test
make ts-coverage
make ts-check
make ts-server
make ts-client
```

Go (via Docker):

```bash
make go-build
make go-test
make go-vet
make go-check
make go-server
```

Shared:

```bash
make openfga-up
make openfga-down
make clean
```

## How to study this repo

Do not read passively. ReBAC becomes useful when you can explain every decision
as a graph path.

Good study moves:

- rename a relation and follow the compiler errors
- remove a tuple and predict authorization behavior
- add one test before changing implementation
- compare how TypeScript unions and Go constants model the same relation names
- explain every permission as a graph path

Bad study moves:

- memorizing syntax without running code
- adding abstractions before the problem is visible
- treating `as` casts as a normal escape hatch
- testing mocks instead of behavior

The goal is not to write fancy TypeScript or fancy Go. The goal is to keep
important authorization rules obvious.

## Keep it fun without making it fluffy

The entertaining part of this repo is the feedback loop:

- break a relation and watch the compiler object
- remove a tuple and watch access disappear
- run the terminal client as Alice, Bob, then Casey
- start services locally and make the graph answer real HTTP requests

Every chapter should leave you with something you can run, break, or explain.
