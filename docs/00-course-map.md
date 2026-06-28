# Course Map

This is a self-contained Go-first ReBAC course. The goal is to learn enough Go
to change a real service, understand relationship-based authorization, read and
run the implementation, and know how the OpenFGA backend fits.

## Go Foundations

If you are new to Go, read these first. Experienced Go programmers can start at
the core ReBAC path.

| Doc | Topic | Practice |
|---|---|---|
| 09 | Go learning path, coverage map, practice loop | choose the route and drills |
| 10 | Toolchain, modules, packages, syntax, control flow, structs, `defer` | inspect and run `internal/rebac` |
| 11 | Values, pointers, methods, slices, maps, strings, nil | inspect document copying and token slices |
| 12 | Errors, interfaces, package design, table tests, test doubles | extend a parser or validation test |
| 13 | `net/http`, JSON, context, mutexes, lifecycle, `httptest` | trace and run the HTTP service |
| 14 | Go idioms and patterns: package shape, constructors, interfaces, errors, concurrency taste | review one package boundary and explain why it is shaped that way |

## Core ReBAC Path

Read these in order:

| Doc | Topic | Code to inspect |
|---|---|---|
| 02 | Authorization fundamentals: RBAC, ABAC, ReBAC | conceptual |
| 03 | Graph theory needed for ReBAC | conceptual |
| 04 | ReBAC concepts: why tuples, subject sets, checks | `internal/rebac/rebac.go` |
| 05 | OpenFGA model DSL: why this policy is structured this way | `deployments/openfga/model.fga`, `internal/authz/model.go` |
| 07 | Designing a ReBAC authz service: product sentences to policy | `deployments/openfga/model.fga.yaml` |
| 27 | Graph evaluator walkthrough | `internal/authz/evaluator.go` |

This path answers one question: how does a relationship become an allow/deny
decision?

## Go Implementation Path

| Doc | Topic | Code to inspect |
|---|---|---|
| 20 | Go language guide for this repository | representative files across `internal/` |
| 06 | Architecture: ports and adapters | `internal/authz/authz.go`, `internal/documents/documents.go` |
| 21 | Go ReBAC implementation walkthrough | `internal/authz/evaluator.go`, `internal/documents/service.go` |
| 25 | Testing patterns and authorization test strategy | `internal/authz/evaluator_test.go` |
| 28 | Request call flow | `cmd/server/main.go`, `internal/api/handler.go` |
| 29 | Guided end-to-end feature lab | add document deletion through service, HTTP, authz, and tests |

## Authentication and Production Path

| Doc | Topic | Code to inspect |
|---|---|---|
| 01 | OAuth/OIDC and the identity handoff to ReBAC | `internal/documents/token.go`, `internal/api/handler.go` |
| 26 | Build-vs-OpenFGA decision and migration path | `internal/openfga/openfga.go` |
| 34 | OpenFGA adapter walkthrough | `internal/openfga/openfga.go` |
| 40 | Production readiness | boundaries and replacement checklist |

## Advanced Go Language Practice

These examples are not part of the running authorization path. They are small
teaching modules that use the same domain types. If your main goal is learning
Go, do them; they are optional only for understanding ReBAC.

| Doc | Topic | Code |
|---|---|---|
| 22 | Concurrency | `examples/concurrency/parallel.go` |
| 23 | Generics | `examples/generics/result.go` |
| 24 | Interfaces and embedding | `examples/middleware/middleware.go` |
| 25 | Testing, fuzzing, benchmarks, race detector, contract tests | `internal/*_test.go`, `examples/*_test.go` |

## Operations

| Doc | Topic | Code |
|---|---|---|
| 30 | Docker fundamentals | `Dockerfile` |
| 31 | Docker networking | `deployments/docker-compose.yml` |
| 32 | Docker Compose local services | `deployments/docker-compose.yml` |
| 33 | Product HTTP API and authz-service HTTP seam | `cmd/server/main.go`, `examples/authzhttp` |

## Suggested Pace

For programmers new to Go:

1. Read doc 09, then complete docs 10–14 and their experiments.
2. Read docs 02–05 and 07.
3. Run `make test`.
4. Read `internal/authz/evaluator.go` with doc 27 open beside it.
5. Run `make trace`.
6. Complete docs 20, 21, 22, 23, 24, 25, and 28.
7. Complete the guided feature lab in doc 29.
8. Read docs 01, 26, 34, and 40 for the OpenFGA production path.

For programmers who already use Go, begin at step 2.
