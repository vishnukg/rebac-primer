# Course Map

This is a Go-first ReBAC course. The goal is to understand relationship-based
authorization, read the implementation, run it, and know how the OpenFGA backend
fits.

## Core ReBAC Path

Read these in order:

| Doc | Topic | Code to inspect |
|---|---|---|
| 02 | Authorization fundamentals: RBAC, ABAC, ReBAC | conceptual |
| 03 | Graph theory needed for ReBAC | conceptual |
| 04 | ReBAC concepts: tuples, subject sets, checks | `internal/rebac/rebac.go` |
| 05 | OpenFGA model DSL | `deployments/openfga/model.fga`, `internal/authz/model.go` |
| 27 | Graph evaluator walkthrough | `internal/authz/evaluator.go` |

This path answers one question: how does a relationship become an allow/deny
decision?

## Go Implementation Path

| Doc | Topic | Code to inspect |
|---|---|---|
| 20 | Go language guide for this repository | representative files across `internal/` |
| 06 | Architecture: ports and adapters | `internal/authz/authz.go`, `internal/documents/documents.go` |
| 21 | Go ReBAC implementation walkthrough | `internal/authz/evaluator.go`, `internal/documents/service.go` |
| 28 | Request call flow | `cmd/server/main.go`, `internal/api/handler.go` |

## Authentication and Production Path

| Doc | Topic | Code to inspect |
|---|---|---|
| 01 | OAuth/OIDC and the identity handoff to ReBAC | `internal/documents/token.go`, `internal/api/handler.go` |
| 26 | From-scratch ReBAC vs OpenFGA | `internal/openfga/openfga.go` |
| 34 | OpenFGA adapter walkthrough | `internal/openfga/openfga.go` |
| 40 | Production readiness | boundaries and replacement checklist |

## Optional Go Language Examples

These examples are not part of the running authorization path. They are small
teaching modules that use the same domain types:

| Doc | Topic | Code |
|---|---|---|
| 22 | Concurrency | `examples/concurrency/parallel.go` |
| 23 | Generics | `examples/generics/result.go` |
| 24 | Interfaces and embedding | `examples/middleware/middleware.go` |
| 25 | Testing patterns | `internal/authz/evaluator_test.go` |

## Operations

| Doc | Topic | Code |
|---|---|---|
| 30 | Docker fundamentals | `Dockerfile` |
| 31 | Docker networking | `deployments/docker-compose.yml` |
| 32 | Docker Compose local services | `deployments/docker-compose.yml` |
| 33 | Product HTTP API and authz-service HTTP seam | `cmd/server/main.go`, `examples/authzhttp` |

## Suggested Pace

1. Read docs 02-05.
2. Run `make test`.
3. Read `internal/authz/evaluator.go` with doc 27 open beside it.
4. Run `go test -v -run TestTrace ./internal/authz`.
5. Follow the Go path if you want to understand the implementation.
6. Read docs 01, 26, 34, and 40 when you want the production boundaries.
