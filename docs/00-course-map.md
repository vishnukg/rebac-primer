# Course Map

This is a Go-first ReBAC course. The goal is to understand relationship-based
authorization, read the implementation, run it, and know how the OpenFGA backend
fits.

## Core Path

Read these in order:

| Doc | Topic | Code to inspect |
|---|---|---|
| 01 | OAuth/OIDC authentication fundamentals | conceptual |
| 02 | Authorization fundamentals: RBAC, ABAC, ReBAC | conceptual |
| 03 | Graph theory needed for ReBAC | conceptual |
| 04 | ReBAC concepts: tuples, subject sets, checks | `internal/rebac/rebac.go` |
| 05 | OpenFGA model DSL | `deployments/openfga/model.fga`, `internal/authz/model.go` |
| 06 | Architecture: ports and adapters | `internal/authz/authz.go`, `internal/documents/documents.go` |
| 21 | Go ReBAC implementation walkthrough | `internal/authz/evaluator.go`, `internal/documents/service.go` |
| 26 | From-scratch ReBAC vs OpenFGA | `internal/openfga/openfga.go` |
| 27 | Graph evaluator walkthrough | `internal/authz/evaluator.go` |
| 28 | Request call flow | `cmd/server/main.go`, `internal/api/handler.go` |
| 34 | OpenFGA adapter walkthrough | `internal/openfga/openfga.go` |
| 40 | Production readiness | conceptual |

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
| 33 | Client/server ReBAC demo | `cmd/server/main.go` |

## Suggested Pace

1. Read docs 01-05.
2. Run `make test`.
3. Read `internal/authz/evaluator.go` with doc 27 open beside it.
4. Run `go test -v -run TestTrace ./internal/authz`.
5. Read docs 26 and 34 to understand the OpenFGA backend.
6. Run `make openfga/up && make openfga/seed && make server-openfga`.
