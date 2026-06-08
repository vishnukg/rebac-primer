# Go primer for TypeScript developers

This chapter is a fast introduction to Go for someone who already knows TypeScript.
It covers the parts of the language that come up in the Go implementation of this
repo. It is not a complete reference — it is the minimum surface area needed to
read and modify the code.

## Why Go alongside TypeScript?

The same domain — collaborative documents with ReBAC authorization — is
implemented in both languages. Having them side by side lets you focus on the
*language* differences instead of the *problem* differences. When you see a Go
file and a TypeScript file solving the same problem, the contrast is the lesson.

---

## Types and values

TypeScript and Go are both statically typed, but they handle absence, mutation,
and type construction differently.

### No `undefined` — every variable has a zero value

Every Go type has a zero value — the empty state it starts in if you do not
assign one:

```go
var s string   // ""
var n int      // 0
var b bool     // false
var p *string  // nil  (pointer to nothing)
```

There is no `undefined`. When you declare a variable, it always has a value.

The Go pattern for "this might not exist" is a pointer (`*T`) or a named sentinel:

```go
// Returns nil when not found — the caller must check before using the result.
func (r *InMemoryDocumentRepository) FindByID(_ context.Context, id string) (*CollaborativeDocument, error) {
    doc, ok := r.docs[id]
    if !ok {
        return nil, nil  // nil pointer means "not found"
    }
    return &doc, nil
}
```

TypeScript equivalent: `Promise<CollaborativeDocument | undefined>`. Go uses a
pointer that can be `nil` instead of a union with `undefined`.

### Named types as branded strings

TypeScript brands strings with template literal types:

```ts
export type RebacObject<TType extends ObjectType = ObjectType> =
  `${TType}:${string}`;
```

Go uses named types. A named type has the same underlying representation as its
base type but is a distinct type at compile time:

```go
// go/internal/shared/rebac.go
type Object   string  // "type:id" — e.g. "document:roadmapDocument"
type Relation string  // "can_edit", "viewer", etc.
type Subject  string  // Object or a subject-set like "team:platformTeam#member"
```

`Object` and `Relation` are different types even though both are `string` under
the hood. You cannot pass a `Relation` where an `Object` is expected:

```go
func Tuple(obj Object, rel Relation, subject Subject) TupleKey { ... }

// Compile error — cannot use Relation as Object:
// shared.Tuple(shared.RelationDocumentCanEdit, shared.Document("x"), ...)
```

### Structs instead of object types

TypeScript:

```ts
type TupleKey = Readonly<{
  user: Subject;
  relation: Relation;
  object: RebacObject;
}>;
```

Go:

```go
// go/internal/shared/rebac.go
type TupleKey struct {
    Object   Object
    Relation Relation
    User     Subject
}
```

Go does not have `Readonly`. The convention is to not expose mutation methods
and to copy the struct before returning when you need an immutable result.

### JSON tags control serialization

When Go encodes a struct to JSON (via `encoding/json`), it uses the field names
by default. Field names in Go are capitalized (exported), which would produce
`{"ID":"...", "UpdatedBy":"..."}` — not idiomatic JSON. Add struct tags to
control the output:

```go
// go/internal/documents/ports.go
type CollaborativeDocument struct {
    ID        string        `json:"id"`
    Title     string        `json:"title"`
    Body      string        `json:"body"`
    Workspace shared.Object `json:"workspace"`
    UpdatedBy shared.Object `json:"updatedBy"`
}
```

With these tags the JSON response is `{"id":"...", "updatedBy":"..."}`, which
matches the TypeScript API. Without them you would get `{"ID":"...", "UpdatedBy":"..."}`.

---

## Interfaces: implicit satisfaction

This is the biggest conceptual shift from TypeScript.

In Go, a type satisfies an interface if it has all the required methods. There is
no `implements` keyword. The connection is established at the point of assignment:

```go
// go/internal/authz/ports.go — the interface, owned by the domain
type Evaluator interface {
    Evaluate(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error)
}

// go/internal/authz/adapters/graph/evaluator.go — the implementation, no "implements" keyword
type GraphEvaluator struct {
    store authz.TupleRepository
}

func (g *GraphEvaluator) Evaluate(_ context.Context, req shared.CheckRequest) (shared.CheckResult, error) {
    // ...
}
```

The compiler proves the connection here:

```go
// go/cmd/server/main.go — the only place that names both the interface and the concrete type
evaluator := graph.NewGraphEvaluator(tupleStore)   // concrete type
authzSvc  := authz.New(tupleStore, evaluator)      // evaluator satisfies authz.Evaluator
// If GraphEvaluator were missing the Evaluate method, authz.New would not compile.
```

TypeScript requires `implements Authorizer` on the class. Go requires nothing on
the implementation — the interface is "satisfied" automatically when the method
signatures match.

### Interfaces tend to be small

The Go community convention: keep interfaces to one or two methods. The smaller
the interface, the easier it is to satisfy with a mock in tests.

```go
// go/internal/authz/ports.go
// TupleRepository is what GraphEvaluator depends on.
// The interface lists only the methods the evaluator needs.
type TupleRepository interface {
    Has(ctx context.Context, object shared.Object, relation shared.Relation, user shared.Subject) (bool, error)
    FindByObjectRelation(ctx context.Context, object shared.Object, relation shared.Relation) ([]shared.TupleKey, error)
    FindAll(ctx context.Context, filter ...TupleFilter) ([]shared.TupleKey, error)
    Write(ctx context.Context, tuple shared.TupleKey) error
    Delete(ctx context.Context, tuple shared.TupleKey) error
}
```

`InMemoryTupleStore` (in `authz/adapters/db/store.go`) satisfies the full
`TupleRepository` interface. `GraphEvaluator` only calls the read methods, but
accepting the full interface keeps the dependency graph simple.

### Compile-time interface check

A common Go pattern is to add an assertion at package level so the compiler
catches missing methods immediately, before any code runs:

```go
// go/internal/authz/adapters/graph/evaluator.go
var _ authz.Evaluator = (*GraphEvaluator)(nil)
```

Breaking this down:
- `var _` — declare a variable and throw it away (blank identifier)
- `authz.Evaluator` — the type of that variable is the interface
- `= (*GraphEvaluator)(nil)` — assign a nil pointer of the concrete type

If `GraphEvaluator` is ever missing the `Evaluate` method, the assignment fails
to compile because a nil `*GraphEvaluator` cannot satisfy `authz.Evaluator`. This
catches the mistake at compile time rather than at the first runtime call.

You will also see `_` used to explicitly discard return values:

```go
_, err := docs.Create(ctx, input)  // first return value discarded
_ = json.NewEncoder(w).Encode(body)  // error intentionally ignored
```

Go requires every declared variable to be used. `_` is the escape hatch — it
signals "I know this value exists and I am deliberately not using it."

---

## Errors as values

TypeScript throws and catches exceptions. Go returns errors as values.

```ts
// TypeScript — throws
async function requireDocument(id: string): Promise<CollaborativeDocument> {
  const existing = await this.repository.findById(id);
  if (!existing) {
    throw DocumentNotFoundError(id);
  }
  return existing;
}
```

```go
// go/internal/documents/domain.go — returns error
func (s *documentService) requireDocument(ctx context.Context, id string) (*CollaborativeDocument, error) {
    doc, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }
    if doc == nil {
        return nil, &DocumentNotFoundError{ID: id}
    }
    return doc, nil
}
```

The caller must handle the error. If you ignore `err`, the compiler does not warn
you — but `go vet` and standard linters will.

### Custom error types

TypeScript:

```ts
export class DocumentNotFoundError extends Error {
  constructor(id: DocumentId) {
    super(`Document not found: ${id}`);
  }
}
```

Go:

```go
// go/internal/documents/documents.go
type DocumentNotFoundError struct {
    ID string
}

func (e *DocumentNotFoundError) Error() string {
    return fmt.Sprintf("document not found: %s", e.ID)
}
```

Any type with an `Error() string` method satisfies the built-in `error` interface.
There is no `extends Error`. The interface does the work.

### Wrapping errors with `%w`

When a function receives an error and wants to add context before returning it
upstream, it wraps the error:

```go
// go/cmd/server/main.go
_, err := docsSvc.Create(ctx, input)
if err != nil {
    return nil, fmt.Errorf("seed demo document: %w", err)
}
```

The `%w` verb (as opposed to `%v`) preserves the original error type inside the
wrapper. This matters because `errors.As` can then unwrap the chain and find the
original type:

```go
// This works even when the error was wrapped with fmt.Errorf:
var forbidden *documents.ForbiddenError
if errors.As(err, &forbidden) { ... }
```

If you use `%v` instead of `%w`, the original type is lost and `errors.As` will
not find it. Use `%w` whenever you want callers to be able to inspect the original
error type. Use `%v` when you are producing a log message and the caller should
not act on the error type.

### `errors.As` instead of `instanceof`

TypeScript checks the error type with `instanceof`:

```ts
if (error instanceof ForbiddenError) {
  return json(403, { error: error.message });
}
```

Go uses `errors.As`:

```go
// go/internal/documents/adapters/http/handler.go
func (h *handler) writeError(w http.ResponseWriter, err error) {
    var notFound *documents.DocumentNotFoundError
    if errors.As(err, &notFound) {
        writeJSON(w, http.StatusNotFound, errorBody(err.Error()))
        return
    }

    var forbidden *documents.ForbiddenError
    if errors.As(err, &forbidden) {
        writeJSON(w, http.StatusForbidden, errorBody(err.Error()))
        return
    }

    writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
}
```

`errors.As` unwraps error chains. When a service wraps an error with
`fmt.Errorf("context: %w", err)`, the underlying type is preserved but hidden.
`errors.As` digs through the chain to find the target type, so the HTTP layer
never needs to know how many layers of wrapping occurred.

---

## The two-value return — Go's core idiom

Almost every function that can fail returns two values: the result and an error.
This pattern appears in every layer of the Go code:

```go
// Map lookup — value and existence flag
doc, ok := r.docs[id]
if !ok {
    return nil, nil
}

// Function call — result and error
doc, err := s.repo.FindByID(ctx, id)
if err != nil {
    return nil, err
}

// Decode — fills a struct and returns an error
err := json.NewDecoder(r.Body).Decode(&body)
```

When you do not need one of the values, assign it to the blank identifier `_`:

```go
_, err := docs.Create(ctx, input)  // discard the document, keep the error
_ = json.NewEncoder(w).Encode(body)  // discard the encode error (already writing response)
```

The blank `_` tells the compiler "I am intentionally ignoring this value." Without it,
Go would refuse to compile — unused variables are a compile error.

---

## Functions, methods, and constructors

### Methods on structs

TypeScript methods live on classes. Go methods live on any named type — structs
are the most common. A method's receiver can be a value or a pointer:

```go
// Value receiver — sees a copy of p. Caller-side mutation cannot escape.
func (p Point) DistanceFromOrigin() float64 { ... }

// Pointer receiver — sees the address; can mutate fields and shares state.
func (s *InMemoryTupleStore) Write(ctx context.Context, key shared.TupleKey) error { ... }
func (s *InMemoryTupleStore) FindAll(ctx context.Context, filter ...authz.TupleFilter) ([]shared.TupleKey, error) { ... } // also a pointer — needs the mutex
```

The rule in this repo: use pointer receivers when the method mutates state,
holds a mutex/RWMutex, or is large enough that copying is wasteful. The
`InMemoryTupleStore` methods all use pointer receivers because `FindAll()`,
`Has()`, and friends acquire `s.mu.RLock()` — and a `sync.RWMutex` must not be
copied. Value receivers are reserved for small, immutable value types.

Mixing value and pointer receivers on the same type is legal but breaks the
addressability rules for interface satisfaction. Pick one form per type and
stick to it.

### Constructor functions

Go has no `new` keyword for custom initialization. The convention is a `New*`
function that returns a pointer to an initialized struct:

```go
// go/internal/authz/adapters/db/store.go — package db, so callers write db.New(...)
func New(seed ...shared.TupleKey) *InMemoryTupleStore {
    s := &InMemoryTupleStore{
        tuples: make(map[string]shared.TupleKey, len(seed)),
    }
    for _, k := range seed {
        s.Write(k)
    }
    return s
}
```

This is constructor injection — the same pattern as the TypeScript implementation,
just without the `constructor` keyword. Notice the variadic `seed ...TupleKey`:
you can call it with no arguments, one tuple, or a slice spread with `...`:

```go
authzdb.New()                                      // empty store
authzdb.New(t1, t2, t3)                            // three tuples
authzdb.New(fixtures.SeedRelationshipTuples()...)  // spread a slice
```

### `defer` — guaranteed cleanup

`defer` schedules a function call to run when the surrounding function returns,
regardless of how it returns — normal return, early return, or panic:

```go
// go/internal/authz/adapters/db/store.go
func (s *InMemoryTupleStore) Has(_ context.Context, object Object, relation Relation, user Subject) (bool, error) {
    s.mu.RLock()          // acquire a read lock
    defer s.mu.RUnlock()  // release it when this function exits — guaranteed
    _, ok := s.tuples[...]
    return ok, nil
}
```

Without `defer` you would have to call `s.mu.RUnlock()` before every `return`
statement and hope you never miss one. `defer` makes it impossible to forget.

Deferred calls run in LIFO order (last in, first out), so multiple defers in one
function unwind in the reverse order they were written. In this repo, each
function only ever has one lock, so the order does not matter.

### `sync.RWMutex` — concurrent reads, exclusive writes

Go is multi-threaded. Multiple goroutines can run code simultaneously, and
reading and writing a map concurrently causes a data race (undefined behaviour).

`sync.RWMutex` (read-write mutex) lets many readers proceed concurrently while
ensuring that writes are exclusive:

```go
// go/internal/authz/adapters/db/store.go
type InMemoryTupleStore struct {
    mu     sync.RWMutex
    tuples map[string]TupleKey
}

func (s *InMemoryTupleStore) Has(...) (bool, error) {
    s.mu.RLock()            // any number of readers can hold RLock simultaneously
    defer s.mu.RUnlock()
    ...
}

func (s *InMemoryTupleStore) Write(_ context.Context, key TupleKey) error {
    s.mu.Lock()             // exclusive — blocks all readers and other writers
    defer s.mu.Unlock()
    ...
}
```

TypeScript does not need this because the JavaScript event loop is single-threaded.

### Return the interface, not the concrete type

```go
// go/internal/documents/domain.go
func New(repo DocumentRepository, authzClient AuthzClient) Service {
    return &documentService{repo: repo, authzClient: authzClient}
}
```

`documents.New` accepts interfaces and returns an interface. Callers never
see `*documentService` — they only see `Service`. This is the same
intent as TypeScript's `private readonly` constructor parameters, enforced by the
package boundary instead of the class modifier.

---

## Packages and the `internal/` directory

TypeScript organizes code into files with `import`/`export`. Go organizes code
into packages — every `.go` file in the same directory belongs to the same
package.

```
go/
├── go.mod                  — module root, declares "module rebac-primer"
├── cmd/server/             — package main (the composition root + entry point)
└── internal/
    ├── shared/             — package shared (rebac.go: Object, Relation, TupleKey, …)
    ├── authz/              — package authz (authz.go, ports.go, domain.go)
    │   └── adapters/       — db (tuple store), graph (evaluator), http
    ├── documents/          — package documents (documents.go, ports.go, create/read/update.go)
    │   └── adapters/       — db (repo), authn (verifier), http
    └── fixtures/           — package fixtures (fixtures.go)
```

The `internal/` directory is special: packages inside it can only be imported by
code rooted at the same parent. `rebac-primer/internal/authz` can be imported
by `rebac-primer/internal/documents`, but not by any module outside `rebac-primer`.
This enforces the same encapsulation you get from TypeScript barrel files and
access modifiers.

`cmd/server/main.go` declares `package main` — the entry point. Any directory
under `cmd/` that has `package main` produces a separate binary.

---

## `context.Context` — cancellation without async/await

Go does not have `async`/`await`. All I/O runs synchronously in Go goroutines
(lightweight threads). Long-running operations accept `context.Context` as their
first argument so callers can cancel them or attach a deadline:

```go
// go/internal/documents/read.go
func (s *documentService) Read(ctx context.Context, id string, actor shared.Object) (*CollaborativeDocument, error) {
    doc, err := s.requireDocument(ctx, id)  // passes ctx to the repo
    // ...
}
```

The in-memory implementations ignore `ctx` (they mark it `_`), but the signatures
are ready for a real database without a refactor. If you later swap
`InMemoryDocumentRepository` for a Postgres implementation, `ctx` will carry the
request deadline automatically.

TypeScript equivalent: `AbortController` / `AbortSignal`, but most code ignores
it. In Go the convention is universal and consistent.

In tests, the root context is `context.Background()` — it is never cancelled and
has no deadline:

```go
result, err := ev.Evaluate(context.Background(), req)
```

Production handlers receive a context from the incoming HTTP request via
`r.Context()`, which the framework automatically cancels when the client
disconnects. That context flows through every function call down to the repository.

---

## HTTP — no framework needed

Since Go 1.22, the standard `net/http` `ServeMux` supports method-prefixed path
patterns and path variables directly:

```go
// go/internal/documents/adapters/http/server.go
mux := http.NewServeMux()
mux.HandleFunc("GET /health", h.handleHealth)
mux.HandleFunc("POST /documents", h.handleCreateDocument)
mux.HandleFunc("GET /documents/{id}", h.handleGetDocument)
mux.HandleFunc("PATCH /documents/{id}", h.handleUpdateDocument)
```

Extract path variables with:

```go
// go/internal/documents/adapters/http/handler.go
id := r.PathValue("id")
```

There is no Express, Gin, or Echo needed for this kind of REST API. The standard
library is enough.

---

## Testing in Go — AAA style

Go's testing package is built in. Tests live in `*_test.go` files alongside the
production code. The `go test ./...` command discovers and runs them.

Test functions follow the Arrange → Act → Assert pattern with explicit comments
marking each section:

```go
// go/internal/authz/adapters/graph/evaluator_test.go
func TestGraphEvaluator_TeamMemberCanEditDocument(t *testing.T) {
    // Arrange: alice is a member of platformTeam, which is an editor of
    // productWorkspace. roadmapDocument lives in productWorkspace.
    ev := newEvaluator()
    req := shared.CheckRequest{
        User:     fixtures.Alice,
        Relation: shared.RelationDocumentCanEdit,
        Object:   fixtures.RoadmapDocument,
    }

    // Act
    result, err := ev.Evaluate(context.Background(), req)

    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !result.Allowed {
        t.Error("expected allowed=true but got false")
        for _, line := range result.Trace {
            t.Logf("  trace: %s", line)
        }
    }
}
```

**`t.Fatalf`** stops the test immediately — use it when the rest of the test
cannot run (e.g. an unexpected error making later assertions nonsensical).

**`t.Errorf`** marks the test as failed but continues — use it when you can
still check other things.

**`t.Logf`** prints only when the test fails — no noise on passing runs.

### Test helpers with `t.Helper()`

When a helper function calls `t.Fatal` or `t.Error`, Go normally reports the
line in the helper, not the caller. Mark the helper with `t.Helper()` so the
failure points at the test:

```go
// go/internal/documents/service_test.go
func newSeededService(t *testing.T) documents.Service {
    t.Helper()
    // wires up a full service and pre-creates the roadmap document
    ...
}
```

Run all tests:

```bash
make go/test
# or inside the container:
go test ./...
# with verbose output showing each test name:
go test -v ./...
```

---

## Go vs TypeScript comparison table

| Concept               | TypeScript                          | Go                                      |
|-----------------------|-------------------------------------|-----------------------------------------|
| Absence               | `undefined` / `null`                | Zero values; `nil` for pointers         |
| Branded types         | Template literal types              | Named types (`type Object string`)      |
| Interface             | Explicit `implements`, structural   | Implicit — method set must match        |
| Error handling        | `throw` / `try-catch`               | Multiple return values; `errors.As`     |
| Classes               | `class` with `constructor`          | Struct + `New*` function                |
| Async                 | `async`/`await`, `Promise`          | Goroutines; `context.Context` for cancel|
| Modules               | Files + `import`/`export`           | Packages (directory = package)          |
| Immutability          | `readonly`, `as const`              | Convention; copy structs                |
| Enum-like constants   | `enum` or union types               | `const` block + named type              |
| JSON serialization    | Automatic camelCase with decorators | Struct tags: `json:"fieldName"`         |
| Optional chaining     | `a?.b?.c`                           | Explicit `nil` checks                   |

---

## Try this

Open `go/internal/authz/adapters/graph/evaluator_test.go`.

1. Read `TestGraphEvaluator_TeamMemberCanEditDocument`. Notice the trace is
   printed when the test fails — the same `Trace []string` that the TypeScript
   `makeGraphEvaluator` builds.

2. Add a new test: `TestGraphEvaluator_WorkspaceOwnerCanDelete`. Make
   Casey (`user:casey`) an `owner` of `productWorkspace` by adding an extra
   tuple, then verify they can `can_delete` `roadmapDocument`. Use the AAA
   structure with `// Arrange`, `// Act`, `// Assert` comments.

3. Run it: `make go/test`. Read the trace output and match each line to the
   expansion rules in `go/internal/authz/adapters/graph/evaluator.go`.

---

## Checkpoint

Why does Go use `errors.As` instead of a type switch or a type assertion?

Good answer: `errors.As` unwraps error chains. When a service wraps an
underlying error with `fmt.Errorf("context: %w", err)`, the original error type
is preserved but hidden behind the wrapper. A type switch or `.(type)` assertion
would fail on the wrapper. `errors.As` digs through the chain and finds the
concrete type no matter how many wrappers are in the way.
