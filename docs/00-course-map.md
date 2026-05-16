# Course map

Welcome to **TS ReBAC Primer**.

This repo is meant to be more than a code sample. It is a practical TypeScript
course wrapped around a real authorization problem: implementing
relationship-based access control with OpenFGA.

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
- **Concept**: the TypeScript, ReBAC, Node, or Docker idea
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
- TypeScript types keep the authorization vocabulary explicit

The important idea is that TypeScript and ReBAC support each other. ReBAC gives
you a precise domain language. TypeScript lets you encode that language so
mistakes are caught while the code is still cheap to fix.

Here is the story you will keep coming back to:

```text
The workspace editor edits the roadmap document because she is in the platform team.
The workspace viewer can read the roadmap document but cannot edit it.
The outside collaborator has no path through the graph, so access is denied.
```

That tiny cast of users keeps the examples grounded.

## Track 1: TypeScript primer

Read these first if you want this repo to be your TypeScript source of truth.

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 01 | TypeScript mental model, `strict`, project setup | `typescript/tsconfig.json`, `typescript/package.json` |
| 02 | Types, unions, narrowing, template literal types | `typescript/src/authz/types.ts` |
| 03 | Functions, modules, classes, interfaces | `typescript/src/domain/service.ts`, `typescript/src/domain/repository.ts` |
| 04 | Async TypeScript, errors, and service boundaries | `typescript/src/domain/service.ts`, `typescript/src/authz/openfga-client.ts` |
| 05 | Testing TypeScript with Vitest | `typescript/test/*.test.ts` |
| 06 | Coding style for maintainable TypeScript | `docs/06-typescript-code-style.md` |
| 07 | Node ESM, module loading, module patterns, singletons | `typescript/package.json`, `typescript/tsconfig.json`, `typescript/src/main.ts` |
| 08 | OAuth/OIDC authentication fundamentals | conceptual |
| 09 | Authorization fundamentals: RBAC, ABAC, ReBAC | conceptual |

## Track 2: ReBAC with OpenFGA

Read these after docs 08, 09, and 10, or in parallel if authorization is your
main goal.

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 10 | Graph theory needed for ReBAC | conceptual |
| 11 | ReBAC concepts and relationship graphs | `typescript/src/authz/graph-authorizer.ts` |
| 12 | OpenFGA model DSL | `typescript/src/authz/model.ts` |
| 13 | TypeScript OpenFGA implementation | `typescript/src/authz/openfga-client.ts` |

## Track 3: Docker and local services

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 20 | Docker fundamentals: images, containers, Dockerfile | `typescript/Dockerfile` |
| 21 | Docker networking: host ports, service names, Compose DNS | `deployments/docker-compose.yml` |
| 22 | Docker Compose local services | `deployments/docker-compose.yml` |
| 23 | Client/server ReBAC demo with terminal client | `typescript/src/server.ts`, `typescript/src/client/tui.ts` |

## Track 4: Going to production

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 30 | Production readiness: what this repo does not cover | conceptual |

## Track 5: Go implementation

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 40 | Go language primer | `go/internal/authz/types.go`, `go/internal/domain/service.go` |
| 41 | Go ReBAC implementation walkthrough | `go/internal/authz/graph.go`, `go/internal/httpserver/handler.go` |
| 42 | Go concurrency: goroutines, channels, WaitGroups | `go/internal/authz/parallel.go` |
| 43 | Go generics: type parameters, constraints, Result[T] | `go/internal/authz/result.go` |
| 44 | Go interfaces and embedding: decorator pattern | `go/internal/authz/middleware.go` |
| 45 | Go testing: AAA, table-driven, benchmarks, fuzz | `go/internal/authz/graph_test.go` |

---

## Reading paths

### TypeScript path (docs 01 → 13 → 20 → 23)

Read if TypeScript or Node.js is your primary goal.

```
01 → 02 → 03 → 04 → 05 → 06 → 07   TypeScript language
08 → 09 → 10 → 11 → 12 → 13        Authorization + OpenFGA
20 → 21 → 22 → 23                  Docker + client/server
30                                  Production gaps
```

### Go path (no TypeScript required)

Read if Go is your primary goal. Docs 01–07 are TypeScript-specific and can be
skipped. Docs 40–45 are self-contained Go references that cover the language
concepts directly through the implementation code.

```
08 → 09 → 10 → 11 → 12             Authorization theory (language-agnostic)
40                                  Go language primer
41                                  Go ReBAC implementation
42                                  Concurrency: goroutines and channels
43                                  Generics: Result[T] and Map
44                                  Interfaces and embedding: decorator pattern
45                                  Testing: AAA, table-driven, benchmarks, fuzz
20 → 21 → 22                        Docker fundamentals
30                                  Production gaps
```

Start with `make go-test` to confirm the setup works, then open
`go/internal/authz/graph.go` alongside `docs/41-go-rebac-implementation.md`.

### Both languages

Read the TypeScript path first, then Track 5. Docs 40–45 make TS→Go comparisons
at every step, so having the TypeScript mental model first makes the Go docs richer.

---

## Suggested pace

### Day 1 (TypeScript path): Make TypeScript feel less mysterious

1. Read `01-typescript-foundations.md`.
2. Run `make ts-build`.
3. Break one type in `typescript/src/authz/types.ts`.
4. Read the compiler error carefully.
5. Restore the code and run `make ts-test`.

Checkpoint: explain why TypeScript catches `can_edti` before Node runs anything.

### Day 1 (Go path): Get the graph talking

1. Run `make go-test` — confirm all tests pass.
2. Read `40-go-primer.md` sections: named types, interfaces, errors as values, `defer`.
3. Open `go/internal/authz/graph_test.go` and read the trace from `TestGraphAuthorizer_TeamMemberCanEditDocument`.
4. Break one tuple in `go/internal/fixtures/fixtures.go` and predict which tests fail.

Checkpoint: explain what `defer s.mu.RUnlock()` guarantees and why it is better than unlocking manually.

### Day 2 (TypeScript path): Learn the type system through the ReBAC vocabulary

1. Read the type aliases in `typescript/src/authz/types.ts`.
2. Read `02-types-and-values.md`.
3. Add a new permission name to `DocumentRelation`.
4. Watch which files need to change.

Checkpoint: explain why `Relation` is better than `string`.

### Day 2 (Go path): Read the service layer

1. Read `41-go-rebac-implementation.md`.
2. Open `go/internal/domain/service.go` and trace how `Update` becomes an auth check.
3. Run `make go-test -v` and read the service test output.

Checkpoint: explain why `NewDocumentService` returns `DocumentOperations` instead of `*documentService`.

### Day 3 (TypeScript path): Read the service layer like production code

1. Read `03-functions-modules-classes.md`.
2. Inspect `DocumentService` in `typescript/src/domain/service.ts`.
3. Trace how an update request becomes an authorization check.

Checkpoint: explain why `DocumentService` depends on `Authorizer`, not `OpenFgaClient`.

### Day 4: Tests as executable documentation

TypeScript:

1. Read `05-testing-with-vitest.md`.
2. Run `make ts-test`.
3. Change `seedRelationshipTuples()` in `typescript/src/testing/fixtures.ts` and predict which tests fail.

Go:

1. Open `go/internal/authz/graph_test.go` — read each AAA section.
2. Run `make go-test`.
3. Change `SeedRelationshipTuples()` in `go/internal/fixtures/fixtures.go` and predict which tests fail.

Checkpoint: explain why the workspace viewer can read but cannot edit.

### Day 5 (TypeScript path): Understand Node modules

1. Read `07-node-esm-and-module-patterns.md`.
2. Inspect the `.js` extensions in TypeScript imports.
3. Explain why `typescript/src/main.ts` performs actions but `typescript/src/authz/types.ts` does not.

Checkpoint: explain why relative ESM imports use `.js` in TypeScript source.

### Day 5–6: ReBAC and OpenFGA

1. Read `08-oauth-authentication.md`.
2. Read `09-authorization-fundamentals.md`.
3. Read `10-graph-theory-for-rebac.md`.
4. Read `11-rebac-concepts.md`.
5. Read `12-openfga-model.md`.
6. Run `make go-test -v` or `make ts-test` and trace each step against the model.

Checkpoint: draw the path from `user:workspaceEditor` to `document:roadmapDocument#can_edit`.

### Day 7+: Local services and client/server

1. Read `20-docker-fundamentals.md`.
2. Read `21-docker-networking.md`.
3. Read `22-docker-compose-local-services.md`.
4. Start the server with `npm run server`.
5. Run the terminal client with `npm run client`.

Checkpoint: explain what changes when the app runs on your host versus inside
Docker Compose.

### Day 8+: Production readiness

1. Read `30-production-readiness.md`.
2. For each gap listed, write one sentence describing where in this repo the
   production concern would be handled.

Checkpoint: explain why the `Authorizer` interface makes it straightforward to
swap `GraphAuthorizer` for a real OpenFGA client in a production deployment.

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

Do not read passively. TypeScript becomes useful when you make the compiler do
work for you.

Good study moves:

- rename a relation and follow the compiler errors
- remove a tuple and predict authorization behavior
- add one test before changing implementation
- replace a broad type with a narrower union
- explain every permission as a graph path

Bad study moves:

- memorizing syntax without running code
- adding abstractions before the problem is visible
- treating `as` casts as a normal escape hatch
- testing mocks instead of behavior

The goal is not to write fancy TypeScript. The goal is to write TypeScript that
keeps important business rules obvious.

## Keep it fun without making it fluffy

The entertaining part of this repo is the feedback loop:

- break a relation and watch the compiler object
- remove a tuple and watch access disappear
- run the terminal client as the workspace editor, the workspace viewer, then the outside collaborator
- start services locally and make the graph answer real HTTP requests

Every chapter should leave you with something you can run, break, or explain.
