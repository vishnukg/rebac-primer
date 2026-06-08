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

Open `internal/authz/authz.go` and `service.go`.

`authz.Service` is the app-facing port:

```go
type Service interface {
    Check(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error)
    WriteTuples(ctx context.Context, tuples []rebac.TupleKey) error
    DeleteTuples(ctx context.Context, tuples []rebac.TupleKey) error
    ListTuples(ctx context.Context, filter ...TupleFilter) ([]rebac.TupleKey, error)
}
```

The service delegates checks to an `Evaluator` and writes to a
`TupleRepository`. Tuple writes are validated before they reach a backend.

## Graph Evaluator

Open `internal/authz/evaluator.go`.

For each `(user, relation, object)` check, the evaluator tries:

1. direct tuple
2. subject-set tuple
3. relation expansion from `model.go`
4. document workspace inheritance

`docs/27-graph-evaluator-walkthrough.md` traces those steps line by line.

## Documents Service

Open `internal/documents/service.go`.

The document operations are:

```text
Create -> check workspace editor -> save doc -> write document tuples
Read   -> load doc -> check can_read
Update -> load doc -> check can_edit -> save update
```

The service depends on narrow ports from `documents.go`:

```text
DocumentRepository
AuthzClient
Authenticator
```

## HTTP Adapter

Open `internal/api/handler.go`.

The handler authenticates the bearer token, decodes JSON, calls the documents
service, and maps domain errors to HTTP statuses.

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
