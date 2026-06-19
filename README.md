# ReBAC Primer: Go

This repository teaches relationship-based access control (ReBAC) with a Go
implementation and an optional OpenFGA backend.

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

Minimal ReBAC path:

1. [Authorization fundamentals](docs/02-authorization-fundamentals.md)
2. [Graph theory for ReBAC](docs/03-graph-theory-for-rebac.md)
3. [ReBAC concepts](docs/04-rebac-concepts.md)
4. [OpenFGA model](docs/05-openfga-model.md)
5. [Graph evaluator walkthrough](docs/27-graph-evaluator-walkthrough.md)

Then choose the Go implementation or production path from the course map.

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
