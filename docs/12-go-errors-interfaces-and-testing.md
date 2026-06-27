# Go Errors, Interfaces, Packages, and Testing

Go favors explicit control flow and small behavioral contracts. Errors are
returned values; interfaces are satisfied implicitly; tests use the same
language and toolchain as production code.

## Errors Are Values

The built-in `error` interface is:

```go
type error interface {
    Error() string
}
```

Return an error alongside the normal result:

```go
func ParseObject(s string) (ObjectType, string, error)
```

Handle it near the call:

```go
typ, id, err := rebac.ParseObject(raw)
if err != nil {
    return fmt.Errorf("parse object %q: %w", raw, err)
}
```

`%w` preserves the original error in the chain while adding useful context.
Use `%v` when wrapping is not intended.

Do not log and return the same error at every layer. Usually, lower layers add
context and return; the process or request boundary decides how to log or map it.

## Sentinel and Typed Errors

A sentinel represents a stable category:

```go
var ErrNotFound = errors.New("not found")
```

Check wrapped sentinels with:

```go
if errors.Is(err, ErrNotFound) {
    // ...
}
```

A typed error carries structured details:

```go
type DocumentNotFoundError struct {
    ID string
}

func (e *DocumentNotFoundError) Error() string {
    return fmt.Sprintf("document %q not found", e.ID)
}
```

Extract it with:

```go
var target *DocumentNotFoundError
if errors.As(err, &target) {
    fmt.Println(target.ID)
}
```

Use `errors.Join` when several independent cleanup operations can fail.

`panic` is for impossible internal states or unrecoverable programmer errors,
not expected request failures. Libraries should generally return errors.

## Interfaces

An interface declares methods:

```go
type DocumentRepository interface {
    FindByID(context.Context, string) (*CollaborativeDocument, error)
    Create(context.Context, CollaborativeDocument) error
    Save(context.Context, CollaborativeDocument) error
    Delete(context.Context, string) error
}
```

A type implements it automatically by having those methods. There is no
`implements` declaration.

Compile-time assertions document an intended implementation:

```go
var _ DocumentRepository = (*InMemoryRepository)(nil)
```

The right side is a typed nil pointer; no value is allocated.

## Accept Interfaces, Return Concrete Types

Constructors generally return concrete values:

```go
func New(repo DocumentRepository, authz AuthorizationService) *Service
```

Consumers declare the smallest interface they require. This keeps interfaces
close to the code whose needs define them:

```go
type Checker interface {
    Check(context.Context, rebac.CheckRequest) (rebac.CheckResult, error)
}
```

Avoid creating an interface solely to mock a type or placing every method a
provider supports into one large interface.

## Embedding and Composition

Struct embedding promotes fields and methods:

```go
type AuditedStore struct {
    TupleReader
    Logger *log.Logger
}
```

This is composition, not inheritance. The outer type can override a promoted
method by declaring its own method with the same name.

Interface embedding combines method sets:

```go
type ReadWriter interface {
    Reader
    Writer
}
```

Embedding a broad interface also exposes every method it contains. Use a narrow
interface when the compiler should enforce a capability boundary.

## Package Design

A package should expose a coherent capability, not merely group files by syntax.
Dependencies point from specific policy toward reusable mechanisms:

```text
api -> documents -> authz/rebac
cmd/server -> all concrete implementations
```

`cmd/server` is the composition root: it chooses implementations and wires them
together. Domain packages do not read process environment or choose databases.

Avoid import cycles by moving shared vocabulary downward or reconsidering the
boundary. Do not create a generic `util` package as a dumping ground.

## Testing Basics

Tests live in files ending `_test.go`:

```go
func TestParseObject(t *testing.T) {
    typ, id, err := ParseObject("document:roadmap")
    if err != nil {
        t.Fatalf("ParseObject returned error: %v", err)
    }
    if typ != ObjectTypeDocument {
        t.Errorf("type = %q, want %q", typ, ObjectTypeDocument)
    }
    if id != "roadmap" {
        t.Errorf("id = %q, want roadmap", id)
    }
}
```

Use `Fatal` only when continuing would make the rest of that test meaningless.
Use `Error` when more assertions can still provide useful information.

Run one package, test, or subtest:

```bash
go test ./internal/rebac
go test -run TestParseObject ./internal/rebac
go test -run 'TestName/subtest name' ./path/to/package
```

## Table-Driven Tests

Table tests make input/output rules visible:

```go
func TestParseObject(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {name: "document", input: "document:roadmap"},
        {name: "missing separator", input: "document", wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, _, err := ParseObject(tt.input)
            if (err != nil) != tt.wantErr {
                t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

Each table row should describe behavior, not reproduce implementation details.

## Test Doubles

Small interfaces make hand-written fakes straightforward:

```go
type fakeChecker struct {
    result rebac.CheckResult
    err    error
}

func (f fakeChecker) Check(
    context.Context,
    rebac.CheckRequest,
) (rebac.CheckResult, error) {
    return f.result, f.err
}
```

Prefer a small fake or stub over a mocking framework when it keeps the test
obvious. Assert externally visible behavior and important calls, not every
internal method invocation.

## Tooling Loop

Use this loop before considering a change complete:

```bash
gofmt -w .
go test ./...
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck ./...
go test -race ./...
```

Useful additional commands:

```bash
go test -cover ./...
go test -count=1 ./...         # bypass the test cache
go test -shuffle=on ./...      # expose ordering dependencies
go test -bench=. ./path        # run benchmarks
go test -fuzz=FuzzName ./path  # run a fuzz target
```

## Try It

Open `internal/documents/service_test.go` and identify:

1. the interface replaced by each fake
2. one allowed case and one denied case
3. where `errors.Is` or `errors.As` preserves error meaning
4. which assertions describe behavior rather than implementation

Then add a table row for an invalid object to an existing parser or validation
test and run only that test before running the entire suite.

## Checkpoint

You are ready to continue when you can explain:

- why `%w` matters
- when to use `errors.Is` versus `errors.As`
- why the consuming package often owns an interface
- why constructors normally return concrete types
- how a table-driven test differs from putting many assertions in one test

Next: [HTTP, JSON, context, and application lifecycle](13-go-http-json-and-context.md).
