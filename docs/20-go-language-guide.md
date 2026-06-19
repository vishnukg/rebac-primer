# Go Language Guide for This Repository

This chapter is the shortest path from basic Go syntax to reading the ReBAC
implementation. It deliberately teaches only concepts used by this repository.

## Start With the Package Shape

Every directory is a package. Files in one package share declarations:

```text
internal/rebac      domain vocabulary and parsing
internal/authz      authorization interfaces, service, store, evaluator
internal/documents  document use cases and ports
internal/api        HTTP adapter
cmd/server          program entry point and dependency wiring
```

`internal` is a Go rule: packages outside this module cannot import these
packages. `cmd/server` is `package main`; its `main` function starts the program.

## Types, Structs, and Constructors

Go uses small named types to prevent accidental mixing:

```go
type Object string
type Relation string

type TupleKey struct {
    Object   Object
    Relation Relation
    User     Subject
}
```

Constructors such as `rebac.User("alice")` and
`authz.NewInMemoryStore(...)` make valid values and initialize internal state.
The zero value is useful for many Go types, but not every domain type should be
constructed as a zero value.

## Functions, Methods, and Errors

A function has no receiver:

```go
func ParseObject(s string) (ObjectType, string, error)
```

A method has a receiver before its name:

```go
func (s *documentService) Read(ctx context.Context, id string, actor rebac.Object) (...)
```

Go returns errors as values. Check them immediately and add context with `%w`
when crossing a useful boundary:

```go
if err != nil {
    return fmt.Errorf("write tuple: %w", err)
}
```

Use `errors.Is` for sentinel errors and `errors.As` for typed errors. Do not use
panic for normal request failures.

## Interfaces and Dependency Direction

Interfaces describe behavior:

```go
type AuthzClient interface {
    Check(context.Context, rebac.CheckRequest) (rebac.CheckResult, error)
    WriteTuples(context.Context, []rebac.TupleKey) error
    DeleteTuples(context.Context, []rebac.TupleKey) error
}
```

Go interface satisfaction is implicit. The package that consumes behavior
usually owns the small interface it needs. `cmd/server` supplies concrete
implementations.

Read `internal/documents/documents.go` before
`internal/documents/service.go`: the interfaces make the service's dependencies
explicit.

## Context

`context.Context` is the first parameter for request-scoped work. It carries
cancellation and deadlines across HTTP, domain, datastore, and OpenFGA calls.

Do not store a context in a long-lived struct. Pass it through the call chain.
Tests commonly use `context.Background()`.

## Slices, Maps, and Copies

A slice is a view over an array; a map is a reference-like hash table. Both can
be mutated by code that receives them. The demo token verifier copies input and
output scope slices so callers cannot mutate its internal state.

The tuple store uses a map keyed by `TupleKey`. Struct values containing
comparable fields are valid map keys, which keeps exact tuple lookup simple.
`FindByObjectRelation` scans the small teaching store linearly. A production
datastore would use indexes; adding an in-memory secondary index here would make
the learning implementation harder to follow without changing the lesson.

## Goroutines and Synchronization

The in-memory stores use `sync.RWMutex` because HTTP handlers can run
concurrently. `RLock` permits concurrent readers; `Lock` gives exclusive access
to writers.

Read `docs/22-go-concurrency.md` for goroutines, channels, cancellation, and the
race detector. Concurrency is not automatically faster: use it when independent
work is slow enough to justify scheduling and coordination.

## Testing and Tooling

The normal quality loop is:

```bash
gofmt -w .
go test ./...
go vet ./...
go tool staticcheck ./...
go test -race ./...
go fix -diff ./...
```

`go fix -diff` reports modern Go rewrites without changing files. Go 1.26
revamped `go fix` around analyzers, so it is now useful as a modernization
check. Optimize only after measuring with a benchmark or profile.

## Suggested Reading Order

1. `internal/rebac/rebac.go` — types, constants, parsing, constructors
2. `internal/authz/authz.go` — interfaces
3. `internal/authz/store.go` — maps and mutexes
4. `internal/authz/service.go` — validation and delegation
5. `internal/authz/evaluator.go` — recursion and graph traversal
6. `internal/documents/service.go` — business use cases
7. `internal/api/handler.go` — HTTP boundary
8. `cmd/server/main.go` — dependency wiring and graceful shutdown

## Official Go References

- [A Tour of Go](https://go.dev/tour/)
- [Go User Manual](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Go 1.26 release notes](https://go.dev/doc/go1.26)

Effective Go remains useful, but it is not a complete modern style guide by
itself. Prefer current standard-library documentation, release notes, analyzers,
tests, and clear code over mechanically applying old idioms.
