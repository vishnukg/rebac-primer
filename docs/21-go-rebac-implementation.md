# Go ReBAC Implementation

This chapter maps the ReBAC concepts to the Go code.

## Layout

```text
cmd/server/          composition root
internal/rebac/      shared vocabulary
internal/authz/      authorization service and graph evaluator
internal/documents/  document service and authn/repository ports
internal/api/        HTTP adapter
internal/openfga/    OpenFGA-backed authz service
internal/fixtures/   demo users, tuples, tokens
```

## Vocabulary

Open `internal/rebac/rebac.go`.

```go
type Object string
type Relation string
type Subject string
```

`TupleKey` is one relationship fact:

```go
type TupleKey struct {
    Object   Object
    Relation Relation
    User     Subject
}
```

Read a tuple as: object has relation to user.

## Authz Service

Open `internal/authz/authz.go` and `internal/authz/service.go`.

`authz.Service` is the concrete in-process implementation:

```go
type Service struct {
    repository TupleRepository
    evaluator  Evaluator
}
```

Consumers declare narrow interfaces that `*authz.Service` satisfies implicitly.
The service delegates checks to an `Evaluator` and writes to a
`TupleRepository`. Check requests and tuple writes are validated against the
known model before they reach a backend. This avoids turning caller mistakes
into silent denials or storing facts that can never match.

## Graph Evaluator

Open `internal/authz/evaluator.go`.

For each `(user, relation, object)` check, the evaluator tries:

1. direct tuple
2. subject-set tuple
3. relation expansion from `internal/authz/model.go`
4. document workspace inheritance

`docs/27-graph-evaluator-walkthrough.md` traces those steps line by line.

## Documents Service

Open `internal/documents/service.go`.

The document operations are:

```text
Create -> check workspace editor -> atomically create doc -> write document tuples
Read   -> load doc -> check can_read
Update -> load doc -> check can_edit -> save update
```

If tuple creation fails, `Create` performs compensating cleanup. This keeps the
example coherent without pretending two independent stores share a transaction.

The document service depends on two narrow ports from
`internal/documents/documents.go`:

```text
DocumentRepository
AuthorizationService
```

The HTTP adapter declares its own `DocumentService` and `Authenticator`
interfaces because it is the package that consumes those capabilities.

## HTTP Adapter

Open `internal/api/handler.go`.

The handler authenticates the bearer token, enforces the endpoint's coarse OAuth
scope, decodes bounded JSON, calls the documents service, and maps domain errors
to HTTP statuses. ReBAC remains the separate object-level decision.

## Composition Root

Open `cmd/server/main.go`.

This is the only place that chooses concrete implementations. Default mode wires
the in-process graph evaluator. `AUTHZ_BACKEND=openfga` wires the OpenFGA
adapter instead.

## Run

```bash
make test
make server
```

Trace the evaluator:

```bash
go test -v -run TestTrace ./internal/authz
```

## Read It Actively

After each package, answer one question:

- `rebac`: what values can represent a graph edge?
- `authz`: where are policy facts stored, and where are rules stored?
- `documents`: which business operations require which permissions?
- `api`: which failures are authentication, scope, ReBAC, or malformed input?
- `cmd/server`: which concrete implementations are selected?

If you cannot answer one, return to that package before continuing.
