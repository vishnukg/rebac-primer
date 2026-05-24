# Go ReBAC implementation

This chapter walks through the Go implementation of the collaborative documents
authorization system. Read `20-go-primer.md` before this one — it covers the Go
language concepts that appear here.

The goal is to read the same design in a second language and notice what changes,
what stays the same, and why.

---

## Package map

The Go code mirrors the TypeScript ports/adapters structure exactly.
Each service has a `core/` layer (pure logic + interface definitions) and an
`adapters/` layer (concrete implementations).

```
go/
├── cmd/server/main.go               Entry point — starts the HTTP server
└── internal/
    ├── shared/
    │   └── rebac.go                 Object / Relation / TupleKey / CheckRequest types
    │                                — mirrors typescript/src/shared/rebac.ts
    ├── authzservice/
    │   ├── core/
    │   │   ├── ports/ports.go       TupleRepository + Evaluator interfaces (driven ports)
    │   │   └── domain/domain.go     AuthzService interface + authzDomain impl
    │   └── adapters/
    │       ├── db/store.go          InMemoryTupleStore
    │       ├── graph/
    │       │   ├── evaluator.go     GraphEvaluator (graph traversal)
    │       │   ├── permissionmodel.go  Implied-by rules
    │       │   ├── parallel.go      AllPermissions / BulkCheck (concurrency demo)
    │       │   ├── middleware.go    AuditEvaluator, ReadOnlyStore (decorator demo)
    │       │   └── result.go        Result[T] generic (generics demo)
    │       └── openfga/authorizer.go   OpenFGA SDK adapter
    ├── documentsservice/
    │   ├── core/
    │   │   ├── ports/ports.go       CollaborativeDocument + DocumentRepository +
    │   │   │                        AuthzClient + Authenticator interfaces
    │   │   └── domain/
    │   │       ├── document.go      Type alias + input/error types
    │   │       ├── service.go       Documents interface + documentService struct
    │   │       ├── create.go        Create use case
    │   │       ├── read.go          Read use case
    │   │       └── update.go        Update use case
    │   └── adapters/
    │       ├── db/repository.go     InMemoryDocumentRepository
    │       ├── authn/verifier.go    DemoTokenVerifier
    │       └── http/
    │           ├── server.go        NewServer() — registers routes
    │           ├── handler.go       Route handlers + error dispatch
    │           └── json.go          writeJSON / readJSON helpers
    ├── fixtures/fixtures.go         Shared test data (users, tuples)
    └── app/app.go                   Composition root
```

The dependency arrows flow inward:

```
cmd/server → app → adapters → core/domain → core/ports
                                           ↑
                                        shared/
```

No arrows point outward. `shared/` has no project imports at all.

---

## `shared/rebac.go` — the type foundation

Open `go/internal/shared/rebac.go`.

### Named types — Go's answer to branded strings

The TypeScript implementation uses template literal types to make strings carry
type information:

```ts
// typescript/src/shared/rebac.ts
type RebacObject<TType extends ObjectType = ObjectType> = `${TType}:${string}`;
```

Go uses named types. A named type has the same memory layout as its base type
(`string`) but is a distinct type at compile time:

```go
// go/internal/shared/rebac.go
type Object   string  // "type:id" — e.g. "document:roadmapDocument"
type Relation string  // "can_edit", "viewer", etc.
type Subject  string  // an Object string, or a subject-set like "team:x#member"
```

You cannot accidentally pass a `Relation` where an `Object` is required:

```go
shared.Tuple(shared.RelationDocumentCanEdit, shared.Document("x"), ...)
//           ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
// compile error: cannot use Relation as Object
```

This is less precise than TypeScript's generic approach (you can still cast a
raw string with `Object("oops")`) but catches the most common mistakes at
compile time.

### Relation constants

TypeScript uses a union type:

```ts
export type DocumentRelation = "workspace" | "owner" | "editor" | "viewer"
  | "can_read" | "can_comment" | "can_edit" | "can_delete";
```

Go uses a `const` block with the named `Relation` type:

```go
// go/internal/shared/rebac.go
const (
    RelationTeamAdmin   Relation = "admin"
    RelationTeamMember  Relation = "member"

    RelationWorkspaceOwner  Relation = "owner"
    RelationWorkspaceEditor Relation = "editor"
    RelationWorkspaceViewer Relation = "viewer"

    RelationDocumentWorkspace  Relation = "workspace"
    RelationDocumentCanRead    Relation = "can_read"
    RelationDocumentCanEdit    Relation = "can_edit"
    // ...
)
```

---

## `authzservice/core/ports/` — driven ports

Open `go/internal/authzservice/core/ports/ports.go`.

A **driven port** is an interface the domain calls out to. The domain says "I
need something that can do X". Adapters supply the concrete X. The domain never
imports a concrete type.

```go
// go/internal/authzservice/core/ports/ports.go

// TupleRepository is the persistence port.
type TupleRepository interface {
    Has(object shared.Object, relation shared.Relation, user shared.Subject) bool
    FindByObjectRelation(object shared.Object, relation shared.Relation) []shared.TupleKey
    FindAll(filter ...TupleFilter) []shared.TupleKey
    Write(tuple shared.TupleKey)
    Delete(tuple shared.TupleKey)
}

// Evaluator is the graph-traversal port.
type Evaluator interface {
    Evaluate(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error)
}
```

Two differences from the TypeScript port:

1. **`context.Context` as first argument** — Go's universal cancellation
   mechanism. In-memory implementations ignore it (`_ context.Context`), but
   the signature is ready for production use without a refactor.

2. **`error` as a second return value** — Go's error model. TypeScript uses
   `Promise` rejection; Go uses explicit return values.

---

## `authzservice/core/domain/` — the authz use cases

Open `go/internal/authzservice/core/domain/domain.go`.

```go
// AuthzService is the driving port — what callers depend on.
type AuthzService interface {
    Check(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error)
    WriteTuples(ctx context.Context, tuples []shared.TupleKey) error
    DeleteTuples(ctx context.Context, tuples []shared.TupleKey) error
    ListTuples(ctx context.Context, filter ...ports.TupleFilter) ([]shared.TupleKey, error)
}

// authzDomain is the private implementation.
type authzDomain struct {
    repository ports.TupleRepository
    evaluator  ports.Evaluator
}

// New wires the two driven ports together.
func New(repository ports.TupleRepository, evaluator ports.Evaluator) AuthzService {
    return &authzDomain{repository: repository, evaluator: evaluator}
}
```

`New` returns `AuthzService` (the interface), not `*authzDomain` (the struct).
Callers can only use methods on the interface — they cannot reach into the struct
or construct one without going through `New`. This is Go's equivalent of
TypeScript's factory boundary.

---

## `authzservice/adapters/db/` — in-memory tuple store

Open `go/internal/authzservice/adapters/db/store.go`.

### Thread-safe map with `sync.RWMutex`

TypeScript runs in a single thread so `Map` is safe without locks. Go is
multi-threaded:

```go
type InMemoryTupleStore struct {
    mu     sync.RWMutex
    tuples map[string]shared.TupleKey
}

func (s *InMemoryTupleStore) Has(object shared.Object, relation shared.Relation, user shared.Subject) bool {
    s.mu.RLock()           // multiple concurrent readers allowed
    defer s.mu.RUnlock()   // released when this function returns, even on panic
    _, ok := s.tuples[keyFor(shared.TupleKey{...})]
    return ok
}

func (s *InMemoryTupleStore) Write(key shared.TupleKey) {
    s.mu.Lock()            // exclusive — blocks all readers and writers
    defer s.mu.Unlock()
    s.tuples[keyFor(key)] = key
}
```

`defer` guarantees the unlock runs even if a panic occurs.

### Map key format — identical to TypeScript

```go
// go/internal/authzservice/adapters/db/store.go
func keyFor(k shared.TupleKey) string {
    return fmt.Sprintf("%s|%s|%s", k.Object, k.Relation, k.User)
}
```

```ts
// typescript/src/authz-service/adapters/db/makeInMemoryTupleRepository.ts
const keyFor = (t: TupleKey) => `${t.object}|${t.relation}|${t.user}`;
```

Identical logic, different syntax.

---

## `authzservice/adapters/graph/` — the graph evaluator

Open `go/internal/authzservice/adapters/graph/evaluator.go`. This is the most
important file to read side by side with `makeGraphEvaluator.ts`.

The algorithm is identical. Go idioms make it look different.

### Pointer to slice for the trace

TypeScript passes `trace: string[]` and appends to it. JavaScript arrays are
reference types — callee and caller share the same backing array.

Go slices are value types. Passing a slice copies its header (pointer, length,
capacity). To let recursive calls append to the *same* underlying array, the
implementation passes a **pointer to the slice**:

```go
func (g *GraphEvaluator) hasRelation(
    user     shared.Object,
    object   shared.Object,
    relation shared.Relation,
    trace    *[]string,          // pointer — appends visible to all callers
    visited  map[string]bool,    // map — already a reference type, no pointer needed
) bool {
    *trace = append(*trace, fmt.Sprintf("Already evaluated %s; stop", visitKey))
    // ...
}
```

### Visited set as `map[string]bool`

TypeScript:

```ts
const visited = new Set<VisitKey>();
if (visited.has(visitKey)) { ... }
visited.add(visitKey);
```

Go:

```go
visited := make(map[string]bool)
if visited[visitKey] { ... }   // missing key returns false (zero value)
visited[visitKey] = true
```

Reading a missing key returns the zero value for the value type — `false` for
`bool`. So `visited[key]` is safe without an explicit existence check.

### Compile-time interface assertion

```go
// go/internal/authzservice/adapters/graph/evaluator.go
var _ ports.Evaluator = (*GraphEvaluator)(nil)
```

This declares a blank variable of interface type and assigns a nil pointer to it.
If `GraphEvaluator` is ever missing the `Evaluate` method, this line will not
compile. It is a zero-cost guard that makes the compiler your test.

---

## `authzservice/adapters/graph/` — Go-specific extras

These files have no TypeScript equivalent. They demonstrate Go-specific patterns
using the authz types.

### `result.go` — generic value-or-error container

```go
// go/internal/authzservice/adapters/graph/result.go
type Result[T any] struct {
    value T
    err   error
    ok    bool
}

func OK[T any](v T) Result[T]          { return Result[T]{value: v, ok: true} }
func Fail[T any](err error) Result[T]  { return Result[T]{err: err, ok: false} }
```

Go generics use type parameters in square brackets: `[T any]` means "T can be
any type." The constraint `any` is an alias for `interface{}`.

Compare with TypeScript:
```ts
type Result<T> = { ok: true; value: T } | { ok: false; error: string }
```

Go achieves the same idea with a struct and a bool field.

`Map` is a free function rather than a method because Go does not yet support
new type parameters in methods — only in free functions:

```go
func Map[T, U any](r Result[T], f func(T) U) Result[U] {
    if !r.ok {
        return Fail[U](r.err)
    }
    return OK(f(r.value))
}
```

### `parallel.go` — concurrent permission checks

`AllPermissions` checks all four computed permissions concurrently using
goroutines and a buffered channel:

```go
func AllPermissions(ctx context.Context, auth Checker, user, object shared.Object) (PermissionSummary, error) {
    relations := computedRelationsFor(object)
    ch := make(chan outcome, len(relations))     // buffered — goroutines never block

    for _, rel := range relations {
        go func(rel shared.Relation) {
            result, err := auth.Evaluate(ctx, shared.CheckRequest{
                User: user, Relation: rel, Object: object,
            })
            ch <- outcome{relation: rel, allowed: result.Allowed, err: err}
        }(rel)                                  // pass rel as argument — captures a copy
    }

    summary := make(PermissionSummary, len(relations))
    for range len(relations) {
        out := <-ch
        if out.err != nil {
            return nil, fmt.Errorf("check %s: %w", out.relation, out.err)
        }
        summary[out.relation] = out.allowed
    }
    return summary, nil
}
```

`BulkCheck` uses a `sync.WaitGroup` to preserve the input ordering despite
goroutines finishing in an arbitrary sequence.

### `middleware.go` — decorator pattern

`AuditEvaluator` wraps any `Checker` (= `ports.Evaluator`) and logs every call:

```go
type AuditEvaluator struct {
    inner  Checker
    logger *log.Logger
}

func (a *AuditEvaluator) Evaluate(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error) {
    start := time.Now()
    result, err := a.inner.Evaluate(ctx, req)
    a.logger.Printf("check user=%s ... -> %s (%s)", req.User, status, time.Since(start))
    return result, err
}
```

This is the classic Go middleware pattern: take an interface, return the same
interface, do something before/after. `AuditEvaluator` itself satisfies
`Checker`, so it can replace any `Checker` at any call site transparently.

`ReadOnlyStore` demonstrates Go embedding:

```go
type ReadOnlyStore struct {
    ports.TupleRepository               // all read methods promoted automatically
}
```

Embedding `TupleRepository` promotes `Has` and `FindByObjectRelation` onto
`ReadOnlyStore`. The write methods (`Write`, `Delete`) are not promoted because
`TupleRepository` is the whole interface — you have to expose all of it. But
`ReadOnlyStore` can be passed anywhere a `TupleRepository` is expected, and the
compiler prevents calling write methods directly on it.

---

## `documentsservice/core/ports/` — driven ports

Open `go/internal/documentsservice/core/ports/ports.go`.

```go
// CollaborativeDocument is defined here, not in domain/, to avoid an import cycle:
// domain imports ports; repository adapters import ports; neither imports the other.
type CollaborativeDocument struct {
    ID        string        `json:"id"`
    Title     string        `json:"title"`
    Body      string        `json:"body"`
    Workspace shared.Object `json:"workspace"`
    UpdatedBy shared.Object `json:"updatedBy"`
}

type DocumentRepository interface {
    Save(ctx context.Context, doc CollaborativeDocument) error
    FindByID(ctx context.Context, id string) (*CollaborativeDocument, error)
    List(ctx context.Context) ([]CollaborativeDocument, error)
}

// AuthzClient is what the documents domain needs from the authz service.
// AuthzService from authzservice/core/domain satisfies this automatically via
// Go's structural typing — it has Check and WriteTuples with matching signatures.
type AuthzClient interface {
    Check(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error)
    WriteTuples(ctx context.Context, tuples []shared.TupleKey) error
}

type Authenticator interface {
    VerifyAccessToken(authorizationHeader string) (AuthenticatedUser, error)
}
```

### Structural typing satisfies `AuthzClient`

`authzdomain.AuthzService` has `Check` and `WriteTuples` with the same
signatures as `AuthzClient`. Go's structural typing means `*authzDomain`
automatically satisfies `AuthzClient` — no `implements` keyword, no explicit
declaration. `app.go` passes the authz domain directly as an `AuthzClient`:

```go
// go/internal/app/app.go
authzSvc := authzdomain.New(tupleStore, evaluator)   // type: authzdomain.AuthzService
docs := docsdomain.New(docRepo, authzSvc)            // authzSvc satisfies AuthzClient
```

To replace the in-process evaluator with an HTTP call to a separate authz
service, you would write an HTTP client that implements `AuthzClient` and pass
it here. Nothing else changes.

---

## `documentsservice/core/domain/` — use cases split by file

Each use case lives in its own file, mirroring the TypeScript structure.

| Go file     | TypeScript file                    | What it does             |
|-------------|------------------------------------|--------------------------| 
| `service.go`| `makeDocuments.ts`                 | Interface + struct + `New` |
| `create.go` | `makeCreateDocument.ts`            | Create use case          |
| `read.go`   | `makeReadDocument.ts`              | Read use case            |
| `update.go` | `makeUpdateDocument.ts`            | Update use case          |
| `document.go`| `types.ts`                        | Type alias + inputs + errors |

### Type alias avoids conversion boilerplate

`CollaborativeDocument` is defined in `ports/` (so both the domain and the
repository adapter can reference it without a cycle). The domain re-exports it
as a type alias:

```go
// go/internal/documentsservice/core/domain/document.go
type CollaborativeDocument = ports.CollaborativeDocument
```

The `=` means this is a **true alias**, not a new type. Domain code writes
`CollaborativeDocument` everywhere; the compiler sees it as identical to
`ports.CollaborativeDocument`. No `toPort()`/`fromPort()` conversion needed.

### Copying a struct for immutable update

TypeScript spreads to create an updated copy:

```ts
const updated = { ...existing, body: input.body, updatedBy: input.actor };
```

Go dereferences the pointer to copy the struct, then modifies fields:

```go
// go/internal/documentsservice/core/domain/update.go
updated := *existing       // dereference: copies the full struct value
updated.Body = input.Body
updated.UpdatedBy = input.Actor
```

`existing` is unchanged. `updated` is a separate value on the stack.

---

## `documentsservice/adapters/http/` — HTTP adapter

Open `go/internal/documentsservice/adapters/http/server.go`.

### Go 1.22+ routing

TypeScript uses manual path matching:

```ts
const documentId = matchDocumentPath(request.path);
if (documentId && request.method === "GET") { ... }
```

Go 1.22+ `ServeMux` handles method + path patterns natively:

```go
mux.HandleFunc("GET /health", h.handleHealth)
mux.HandleFunc("POST /documents", h.handleCreateDocument)
mux.HandleFunc("GET /documents/{id}", h.handleGetDocument)
mux.HandleFunc("PATCH /documents/{id}", h.handleUpdateDocument)
```

Path variables are extracted with `r.PathValue("id")`. No external router needed.

### Error dispatch with `errors.As`

```go
func (h *handler) writeError(w http.ResponseWriter, err error) {
    if ports.IsAuthenticationError(err) {
        writeJSON(w, http.StatusUnauthorized, errorBody(err.Error()))
        return
    }
    var notFound *domain.DocumentNotFoundError
    if errors.As(err, &notFound) {
        writeJSON(w, http.StatusNotFound, errorBody(err.Error()))
        return
    }
    var forbidden *domain.ForbiddenError
    if errors.As(err, &forbidden) {
        writeJSON(w, http.StatusForbidden, errorBody(err.Error()))
        return
    }
    writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
}
```

This is the Go equivalent of the TypeScript `toErrorResponse` function. The
domain returns typed errors; the HTTP adapter maps them to status codes. Neither
layer knows about the other's details.

`errors.As` unwraps error chains — it works correctly even when the domain error
is wrapped inside a `fmt.Errorf("...: %w", err)` call.

---

## `app/app.go` — composition root

Open `go/internal/app/app.go`. This is the **only** file that imports every
concrete type. Everything else depends only on interfaces.

```go
func NewWithConfig(ctx context.Context, cfg Config) (*App, error) {
    // ── Authz service ─────────────────────────────────────────────────────────
    tupleStore := authzdb.New(fixtures.SeedRelationshipTuples()...)
    evaluator  := graph.NewGraphEvaluator(tupleStore)
    authzSvc   := authzdomain.New(tupleStore, evaluator)

    // ── Documents service ─────────────────────────────────────────────────────
    docRepo       := docsdb.New()
    tokenVerifier := docsauthn.New(fixtures.DemoTokens())
    docs          := docsdomain.New(docRepo, authzSvc)   // authzSvc satisfies AuthzClient

    // seed
    _, err := docs.Create(ctx, docsdomain.CreateDocumentInput{ ... })

    // ── HTTP layer ────────────────────────────────────────────────────────────
    httpHandler := docshttp.NewServer(tokenVerifier, docs)

    return &App{Handler: httpHandler, Port: cfg.Port}, nil
}
```

To swap `GraphEvaluator` for `OpenFGAAuthorizer`:

```go
// Replace these two lines:
evaluator := graph.NewGraphEvaluator(tupleStore)
authzSvc  := authzdomain.New(tupleStore, evaluator)

// With:
authzSvc, err := openfga.New(openfga.Config{
    APIURL:  "http://localhost:8080",
    StoreID: "your-store-id",
})
```

Nothing else changes. The documents domain and HTTP layer never find out.

---

## Tests

### Evaluator tests — `authzservice/adapters/graph/evaluator_test.go`

```go
func TestGraphEvaluator_TeamMemberCanEditDocument(t *testing.T) {
    // Arrange
    ev := graph.NewGraphEvaluator(seedStore())
    req := shared.CheckRequest{
        User:     fixtures.Alice,
        Relation: shared.RelationDocumentCanEdit,
        Object:   fixtures.RoadmapDocument,
    }

    // Act
    result, err := ev.Evaluate(context.Background(), req)

    // Assert
    if !result.Allowed {
        t.Error("expected allowed=true but got false")
        for _, line := range result.Trace {
            t.Logf("  trace: %s", line)  // only printed on failure
        }
    }
}
```

### Domain service tests — `documentsservice/core/domain/service_test.go`

```go
func TestDocumentService_Update_ForbiddenForViewer(t *testing.T) {
    // Arrange: bob has viewer, not editor — update must be denied.
    svc := newSeededService(t)

    // Act
    _, err := svc.Update(context.Background(), domain.UpdateDocumentInput{
        ID: "roadmapDocument", Body: "should not save", Actor: fixtures.Bob,
    })

    // Assert
    var forbidden *domain.ForbiddenError
    if !errors.As(err, &forbidden) {
        t.Errorf("expected *ForbiddenError, got %T: %v", err, err)
    }
}
```

`newSeededService` wires the full in-process stack (no HTTP). Marking it with
`t.Helper()` means failure lines point at the test function, not inside the helper.

### HTTP integration tests — `documentsservice/adapters/http/handler_test.go`

Uses `net/http/httptest` — no network, no port:

```go
func TestHandler_GetDocument_Returns200ForViewer(t *testing.T) {
    handler := newTestHandler(t)
    req := httptest.NewRequest(http.MethodGet, "/documents/roadmapDocument", nil)
    req.Header.Set("Authorization", "Bearer demo-token-bob")
    rec := httptest.NewRecorder()

    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK { ... }
}
```

`httptest.NewRecorder()` captures the response. `handler.ServeHTTP(rec, req)`
calls the full stack in-process. No server, no port, no teardown.

---

## Side-by-side comparison

| Concern                | TypeScript                              | Go                                        |
|------------------------|-----------------------------------------|-------------------------------------------|
| Branded types          | Template literal `\`${T}:${string}\``   | Named type `type Object string`           |
| Relation constants     | Union type `"can_edit" \| ...`          | `const` block with named `Relation` type  |
| Port definitions       | `interface` in `core/ports/*.ts`        | `interface` in `core/ports/ports.go`      |
| Interface satisfaction | Object shape must match                 | Implicit — method set must match          |
| Factory functions      | `makeDocuments({ repo, authzClient })`  | `domain.New(repo, authzClient)`           |
| Error signalling       | `throw new ForbiddenError(...)`         | `return &ForbiddenError{...}`             |
| Error dispatch         | `instanceof`                            | `errors.As`                               |
| Immutable copy         | `{ ...existing, body: newBody }`        | `updated := *existing; updated.Body = …`  |
| Async / cancellation   | `async`/`await`, `Promise`              | Synchronous + `context.Context`           |
| JSON serialization     | Automatic                               | Struct tags: `json:"fieldName"`           |
| HTTP routing           | Manual `if method && path`              | Go 1.22+ `ServeMux` patterns              |
| Test assertions        | Vitest `expect(x).toBe(y)`              | `if x != y { t.Errorf(...) }`            |
| Test HTTP recorder     | Custom `MockRequest`                    | `httptest.NewRecorder()`                  |

---

## Running the Go server

```bash
make go-server   # starts the server on port 4001
```

Then test the API:

```bash
# Health check
curl http://localhost:4001/health

# Read as Bob (viewer — allowed)
curl http://localhost:4001/documents/roadmapDocument \
  -H "Authorization: Bearer demo-token-bob"

# Read as Casey (outsider — 403)
curl http://localhost:4001/documents/roadmapDocument \
  -H "Authorization: Bearer demo-token-casey"

# Create as Alice (editor — 201)
curl -X POST http://localhost:4001/documents \
  -H "Authorization: Bearer demo-token-alice" \
  -H "Content-Type: application/json" \
  -d '{"id":"notes","title":"Notes","body":"hello","workspaceId":"productWorkspace"}'

# Update as Bob (viewer — 403)
curl -X PATCH http://localhost:4001/documents/roadmapDocument \
  -H "Authorization: Bearer demo-token-bob" \
  -H "Content-Type: application/json" \
  -d '{"body":"should not save"}'
```

The same HTTP API as the TypeScript server (`make ts-server`), just on a different
port.

---

## Checkpoint

`domain.New` returns `Documents` (an interface) instead of `*documentService`
(the concrete struct). Why does this matter?

Good answer: callers can only use the three methods declared on `Documents`. They
cannot inspect the struct's fields, call unexported methods, or construct a
`documentService` without going through `New`. Combined with the `internal/`
directory restriction (nothing outside `rebac-primer` can import the internal
packages), this gives Go the same layered encapsulation that TypeScript achieves
with `private` modifiers and barrel-file access control.
