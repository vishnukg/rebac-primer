# ReBAC Primer: Go

This repository is a self-contained practical Go course built around a
relationship-based access control (ReBAC) service, with an optional OpenFGA
backend. Programmers who are new to Go can begin with the language foundation;
experienced Go programmers can skip directly to the ReBAC implementation.

The project domain is a collaborative document workspace. Workspaces contain
documents, teams get workspace access, and users inherit permissions through a
relationship graph. The domain is small enough to trace by hand but rich enough
to show the important ReBAC ideas.

## Repository Map

```text
internal/rebac/          ReBAC primitives: Object, Relation, TupleKey, CheckRequest
internal/authz/          AuthZ service, tuple store, graph evaluator, model rules
internal/openfga/        OpenFGA-backed authorization service
internal/documents/      Documents service, repository, demo token verifier
internal/api/            HTTP server for the documents service
internal/fixtures/       Shared demo/test data
cmd/server/              Composition root and entry point

deployments/               Docker Compose + OpenFGA model/seed script
docs/                      Tutorial chapters
```

## Start Here

Read [START-HERE.md](START-HERE.md), then follow
[docs/00-course-map.md](docs/00-course-map.md).

New to Go:

1. [Toolchain and core syntax](docs/10-go-toolchain-and-syntax.md)
2. [Values, pointers, collections, and methods](docs/11-go-values-pointers-and-methods.md)
3. [Errors, interfaces, packages, and testing](docs/12-go-errors-interfaces-and-testing.md)
4. [HTTP, JSON, context, and application lifecycle](docs/13-go-http-json-and-context.md)

Minimal ReBAC path:

1. [Authorization fundamentals](docs/02-authorization-fundamentals.md)
2. [Graph theory for ReBAC](docs/03-graph-theory-for-rebac.md)
3. [ReBAC concepts](docs/04-rebac-concepts.md)
4. [OpenFGA model](docs/05-openfga-model.md)
5. [Designing a ReBAC authorization service](docs/07-rebac-authorization-service-design.md)
6. [Graph evaluator walkthrough](docs/27-graph-evaluator-walkthrough.md)

Then choose the Go implementation or production path from the course map.
Finish the Go path with the
[guided feature lab](docs/29-go-guided-feature-lab.md), which adds an operation
through the domain, HTTP, authorization, and testing layers.

## Commands

This repo uses the [3 Musketeers](https://3musketeers.io/) pattern:

```text
make -> docker compose -> containerized tools
```

Go:

```bash
make build
make test
make vet
make check
make server
```

Local OpenFGA:

```bash
make openfga/up
make openfga/seed
make server-openfga
```

Run `make` with no arguments to see all targets.

## The Authorization Story

```text
Alice can edit the roadmap document
  because she is in the platform team
  which is an editor of the product workspace
  which the roadmap document lives in.

Bob can read but not edit.

Casey has no path through the graph, so access is denied.
```

| Person or object | ReBAC ID | Role |
|---|---|---|
| Alice | `user:alice` | platform team member; can edit |
| Bob | `user:bob` | workspace viewer; can read only |
| Casey | `user:casey` | outside collaborator; denied |
| Platform Team | `team:platformTeam` | grants workspace editor access |
| Product Workspace | `workspace:productWorkspace` | contains the roadmap document |
| Roadmap Document | `document:roadmapDocument` | protected document |

The in-process graph evaluator is the learning implementation. The OpenFGA
adapter demonstrates the external authorization-service direction. Both concrete
backends satisfy the narrow interface declared by each consumer, while OpenFGA
stores and evaluates the relationships remotely. The rest of the demo still
requires the production work listed in doc 40.
