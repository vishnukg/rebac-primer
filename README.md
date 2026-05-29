# ReBAC Primer: TypeScript and Go

This repository teaches relationship-based access control (ReBAC) with OpenFGA
through two parallel implementations of the same application — one in TypeScript,
one in Go.

The project domain is a collaborative document workspace. Workspaces contain
documents, teams get workspace access, and users inherit permissions through a
relationship graph. That domain is small enough to stay concrete but rich enough
to demonstrate the core ideas in both languages.

## Repository map

```
typescript/                TypeScript implementation
  src/shared/              ReBAC value helpers (objects, relations, tuples)
  src/authz-service/       AuthZ service: core/ (domain + ports), adapters/ (db, graph, http)
  src/documents-service/   Documents service: core/, adapters/ (authn, authz, db, http, client)
  src/cli/                 Terminal client composition root and entry point
  test/                    Vitest tests

go/                        Go implementation
  internal/shared/         ReBAC primitives (Object, Relation, TupleKey, CheckRequest)
  internal/authz/          AuthZ core (Service, ports) + adapters/ (db, graph, http)
  internal/documents/      Documents core (Service, ports, use cases) + adapters/ (db, authn, http)
  internal/fixtures/       Shared test data
  cmd/server/              Composition root + entry point

docs/             Tutorial chapters (read these in order)
deployments/      Docker Compose for both implementations + OpenFGA
```

## Where to start

Read [docs/00-course-map.md](docs/00-course-map.md) for the full learning path.

Short version:

1. Shared concepts: authn/authz, ReBAC, and the architecture -> docs 01-06
2. TypeScript implementation track -> docs 10-19 (+ doc 29, the authz call flow)
3. Go implementation track -> docs 20-28
4. Shared Docker/local services -> docs 30-33
5. Shared production concerns -> doc 40

You can learn either language without reading the other language track. The
authorization, OpenFGA model, Docker, and production-readiness chapters are the
common spine for both implementations.

## Commands

This repo uses the [3 Musketeers](https://3musketeers.io/) pattern:

```
make → docker compose → containerized tools
```

**TypeScript** (server on port 4000):

```bash
make ts-deps
make ts-build
make ts-test
make ts-coverage
make ts-check
make ts-server
make ts-client
```

**Go** (server on port 4001):

```bash
make go-build
make go-test
make go-vet
make go-check
make go-server
```

**Shared**:

```bash
make openfga-up    # start local OpenFGA
make openfga-down  # stop everything
make clean         # remove containers, volumes, and build output
```

Run `make` with no arguments to see all targets.

## The authorization story

```text
Alice can edit the roadmap document
  because she is in the platform team
  which is an editor of the product workspace
  which the roadmap document lives in.

Bob can read but not edit.

Casey has no path through the graph — access is denied.
```

| Person or object | ReBAC ID | Role in the example |
|------------------|----------|---------------------|
| Alice | `user:alice` | platform team member; can edit |
| Bob | `user:bob` | workspace viewer; can read only |
| Casey | `user:casey` | outside collaborator; denied |
| Platform Team | `team:platformTeam` | grants workspace editor access |
| Product Workspace | `workspace:productWorkspace` | contains the roadmap document |
| Roadmap Document | `document:roadmapDocument` | protected document |

Both implementations answer the same question with the same graph traversal
algorithm. Reading them side by side is the lesson.

## TypeScript ports and adapters

The TypeScript project is organized around a small ports-and-adapters shape:

```text
adapters -> core <- composition roots
```

Each service has a `core/` (the domain language and ports: ReBAC objects,
relations, tuples, `Evaluator`, `Authenticator`, `DocumentRepository`, and the
document operations) and an `adapters/` (concrete details: demo token
verification, the graph evaluator, in-memory persistence, HTTP, and
terminal/HTTP clients).

Domain code does not import concrete infrastructure. For example, document
operations receive a repository and an authz client. Each service's `compose.ts`
chooses the in-memory repository, demo OAuth2 token verifier, and graph
evaluator, then wires them together.

Good files to read first:

1. `typescript/src/documents-service/compose.ts`
2. `typescript/src/authz-service/core/ports/evaluator.ts`
3. `typescript/src/documents-service/core/domain/makeDocuments.ts`
4. `typescript/src/authz-service/adapters/graph/makeGraphEvaluator.ts`
5. `typescript/src/documents-service/adapters/http/makeDocumentsHttpHandler.ts`
