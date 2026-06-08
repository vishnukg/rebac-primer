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
internal/openfga/        OpenFGA-backed authz.Service
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

Minimal path:

1. `docs/01-oauth-authentication.md`
2. `docs/02-authorization-fundamentals.md`
3. `docs/03-graph-theory-for-rebac.md`
4. `docs/04-rebac-concepts.md`
5. `docs/05-openfga-model.md`
6. `docs/21-go-rebac-implementation.md`
7. `docs/27-graph-evaluator-walkthrough.md`
8. `docs/28-go-authz-call-flow.md`

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
adapter is the production direction: it implements the same `authz.Service` port
and stores/evaluates relationships in OpenFGA.
