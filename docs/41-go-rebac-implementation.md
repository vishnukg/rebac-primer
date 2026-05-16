# Go ReBAC implementation

This chapter walks through the Go implementation of the collaborative documents
authorization system. Read the Go primer (doc 40) before this one — it covers the
language concepts that appear here.

The goal is to read the same design in a second language and notice what changes,
what stays the same, and why.

---

## Package map

```
go/
├── go.mod
├── Dockerfile
├── cmd/server/main.go          Entry point — starts the HTTP server
└── internal/
    ├── authz/                  Types, tuple store, graph evaluator, OpenFGA stub
    │   ├── types.go
    │   ├── store.go
    │   ├── graph.go
    │   ├── model.go
    │   └── openfga.go
    ├── domain/                 Domain model and service layer
    │   ├── document.go
    │   ├── repository.go
    │   └── service.go
    ├── httpserver/             HTTP handler and routing
    │   ├── server.go
    │   └── handler.go
    ├── fixtures/               Shared test data
    │   └── fixtures.go
    └── app/                    Composition root
        └── app.go
```

The layering mirrors the TypeScript implementation exactly:

```
cmd/server → app → httpserver → domain → authz
```

Each layer depends only on the layer below it, through interfaces. The `app`
package is the only place that imports both `authz` and `domain` concrete types.

---

## The `authz` package

### Named types — Go's answer to branded strings

Open `go/internal/authz/types.go`.

The TypeScript implementation uses template literal types to make strings carry
type information:

```ts
// typescript/src/authz/types.ts
type RebacObject<TType extends ObjectType = ObjectType> = `${TType}:${string}`;
```

Go uses named types. A named type has the same memory layout as its base type
(`string`) but is a distinct type at compile time:

```go
// go/internal/authz/types.go
type Object   string  // "type:id" — e.g. "document:roadmapDocument"
type Relation string  // "can_edit", "viewer", etc.
type Subject  string  // an Object string, or a subject-set like "team:x#member"
```

You cannot accidentally pass a `Relation` where an `Object` is required:

```go
func Tuple(obj Object, rel Relation, subject Subject) TupleKey { ... }

// This would not compile — cannot use Relation as Object:
// authz.Tuple(authz.RelationDocumentCanEdit, authz.Document("x"), ...)
```

This is less precise than TypeScript's approach (you can still `Object("oops")`
cast a raw string) but catches the most common mistakes at compile time.

### Relation constants

TypeScript uses a union type for relations:

```ts
export type DocumentRelation = "workspace" | "owner" | "editor" | "viewer"
  | "can_read" | "can_comment" | "can_edit" | "can_delete";
```

Go uses a `const` block with the named `Relation` type:

```go
// go/internal/authz/types.go
const (
    RelationTeamAdmin  Relation = "admin"
    RelationTeamMember Relation = "member"

    RelationWorkspaceOwner  Relation = "owner"
    RelationWorkspaceEditor Relation = "editor"
    RelationWorkspaceViewer Relation = "viewer"

    RelationDocumentWorkspace  Relation = "workspace"
    RelationDocumentOwner      Relation = "owner"
    RelationDocumentEditor     Relation = "editor"
    RelationDocumentViewer     Relation = "viewer"
    RelationDocumentCanRead    Relation = "can_read"
    RelationDocumentCanComment Relation = "can_comment"
    RelationDocumentCanEdit    Relation = "can_edit"
    RelationDocumentCanDelete  Relation = "can_delete"
)
```

### Helper constructors

TypeScript:

```ts
export function user(id: string): RebacObject<"user"> {
  return `user:${id}`;
}
```

Go:

```go
// go/internal/authz/types.go
func User(id string) Object {
    return newObject(ObjectTypeUser, id)
}

func newObject(typ ObjectType, id string) Object {
    if strings.TrimSpace(id) == "" {
        panic(fmt.Sprintf("authz: %s id cannot be empty", typ))
    }
    return Object(fmt.Sprintf("%s:%s", typ, id))
}
```

Both panic on an empty id because an empty id is always a programming error, not
a runtime condition to handle gracefully.

### The `Authorizer` interface

```go
// go/internal/authz/types.go
type Authorizer interface {
    Check(ctx context.Context, req CheckRequest) (CheckResult, error)
}
```

Two differences from the TypeScript interface:

1. **`context.Context` as first argument** — Go's universal cancellation
   mechanism. The in-memory implementations ignore it (`_ context.Context`), but
   the signature is ready for production use without a refactor.

2. **`error` as a second return value** — Go's error model. TypeScript uses
   `Promise` rejection; Go uses explicit return values.

### Compile-time interface check

`openfga.go` has this line at the bottom:

```go
// go/internal/authz/openfga.go
var _ Authorizer = (*OpenFGAAuthorizer)(nil)
```

This declares a blank variable of interface type and assigns a nil pointer to it.
If `OpenFGAAuthorizer` is missing the `Check` method, this line will not compile.
It is a zero-cost guard that makes the compiler your test.

---

## The tuple store

Open `go/internal/authz/store.go`.

### Interface segregation

`GraphAuthorizer` only reads tuples — it never writes. It holds a `TupleReader`:

```go
// go/internal/authz/store.go
type TupleReader interface {
    Has(object Object, relation Relation, user Subject) bool
    FindByObjectRelation(object Object, relation Relation) []TupleKey
}
```

`InMemoryTupleStore` implements `TupleReader`, `TupleWriter`, and `TupleStore`
(all three). But `GraphAuthorizer` only sees `TupleReader` — it cannot call
`Write` or `Delete` even if they exist on the concrete type.

TypeScript equivalent: `GraphAuthorizer` accepts `TupleReader` in its
constructor, not the full `InMemoryTupleStore`.

### Thread-safe map with `sync.RWMutex`

TypeScript runs in a single thread so `Map` is safe without locks. Go is
multi-threaded:

```go
// go/internal/authz/store.go
type InMemoryTupleStore struct {
    mu     sync.RWMutex
    tuples map[string]TupleKey
}

func (s *InMemoryTupleStore) Has(object Object, relation Relation, user Subject) bool {
    s.mu.RLock()           // multiple readers allowed simultaneously
    defer s.mu.RUnlock()   // released when this function returns, even on panic
    _, ok := s.tuples[keyFor(TupleKey{Object: object, Relation: relation, User: user})]
    return ok
}

func (s *InMemoryTupleStore) Write(key TupleKey) {
    s.mu.Lock()            // exclusive — no reads or writes while held
    defer s.mu.Unlock()
    s.tuples[keyFor(key)] = key
}
```

`defer` guarantees the unlock runs even if a panic occurs inside the function.

### Map key format

Both implementations use the same string format for map keys:

```go
// go/internal/authz/store.go
func keyFor(k TupleKey) string {
    return fmt.Sprintf("%s|%s|%s", k.Object, k.Relation, k.User)
}
```

```ts
// typescript/src/authz/memory-store.ts
function keyFor(tupleKey: TupleKey): string {
  return `${tupleKey.object}|${tupleKey.relation}|${tupleKey.user}`;
}
```

Identical logic, different syntax.

---

## The graph authorizer

Open `go/internal/authz/graph.go`. This is the most important file to read side
by side with the TypeScript `GraphAuthorizer`.

The algorithm is identical. The structure looks different because of Go idioms.

### Pointer to slice for the trace

TypeScript passes `trace: string[]` and appends to it. JavaScript arrays are
reference types — the callee and caller share the same backing array.

Go slices are value types. Passing a slice copies its header (pointer, length,
capacity). To let recursive calls append to the *same* underlying array and have
the caller see those appends, the implementation passes a pointer to the slice:

```go
// go/internal/authz/graph.go
func (g *GraphAuthorizer) hasRelation(
    user Object,
    object Object,
    relation Relation,
    trace *[]string,           // pointer — callee can append and caller sees it
    visited map[string]bool,   // map — already a reference type, no pointer needed
) bool {
    *trace = append(*trace, fmt.Sprintf("Already evaluated %s; stop this branch", visitKey))
    // ...
}
```

Maps are reference types in Go (like in JavaScript), so `visited` does not need
the same treatment.

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
if visited[visitKey] { ... }   // missing key returns false (zero value for bool)
visited[visitKey] = true
```

Reading a missing key from a Go map returns the zero value for the value type —
`false` for `bool`. So `visited[key]` is safe without an existence check.

### Expanding document relations

The expansion table uses a `map` literal:

```go
// go/internal/authz/graph.go — inside expandDocument
expansions := map[Relation]Relation{
    RelationDocumentCanRead:    RelationDocumentViewer,
    RelationDocumentCanComment: RelationDocumentViewer,
    RelationDocumentCanEdit:    RelationDocumentEditor,
    RelationDocumentCanDelete:  RelationDocumentOwner,
    RelationDocumentViewer:     RelationDocumentEditor,
    RelationDocumentEditor:     RelationDocumentOwner,
}

if implied, ok := expansions[relation]; ok {
    *trace = append(*trace, fmt.Sprintf("document.%s includes document.%s", relation, implied))
    if g.hasRelation(user, object, implied, trace, visited) {
        return true
    }
}
```

The two-value map lookup (`value, ok := m[key]`) is Go's safe way to distinguish
"key is present with value X" from "key is absent". This is equivalent to
TypeScript's `map.get(key) ?? []`.

---

## The domain layer

Open `go/internal/domain/service.go`.

### Unexported struct, exported interface

TypeScript uses a class with private fields:

```ts
// typescript/src/domain/service.ts
export class DocumentService implements DocumentOperations {
  constructor(
    private readonly repository: DocumentRepository,
    private readonly authorizer: Authorizer
  ) {}
}
```

Go uses an unexported struct and an exported constructor that returns an interface:

```go
// go/internal/domain/service.go
type documentService struct {   // lowercase — unexported outside this package
    repo DocumentRepository
    auth authz.Authorizer
}

func NewDocumentService(repo DocumentRepository, auth authz.Authorizer) DocumentOperations {
    return &documentService{repo: repo, auth: auth}
}
```

`NewDocumentService` returns the `DocumentOperations` interface, not
`*documentService`. Callers can only call methods declared on the interface.
They cannot inspect or mutate the struct's fields, and they cannot construct one
without going through the constructor. This is Go's equivalent of TypeScript's
`private readonly` constructor parameters.

### JSON tags on the domain type

```go
// go/internal/domain/document.go
type CollaborativeDocument struct {
    ID        string       `json:"id"`
    Title     string       `json:"title"`
    Body      string       `json:"body"`
    Workspace authz.Object `json:"workspace"`
    UpdatedBy authz.Object `json:"updatedBy"`
}
```

Without these tags, `encoding/json` would use the Go field names verbatim and
produce `{"ID":"...","UpdatedBy":"..."}`. The tags tell the encoder to use
lowercase camelCase names, matching the TypeScript API response format.

### Copying a struct for immutable update

TypeScript spreads to create an updated copy:

```ts
const updated = { ...existing, body: input.body, updatedBy: input.actor };
```

Go dereferences the pointer to copy the struct, then modifies fields on the copy:

```go
// go/internal/domain/service.go — inside Update
updated := *existing       // dereference: makes a copy of the struct value
updated.Body = input.Body
updated.UpdatedBy = input.Actor
```

The original (`*existing`) is unchanged. `updated` is a separate value. This is
idiomatic Go for "create a modified copy without mutating the original".

---

## The HTTP layer

Open `go/internal/httpserver/server.go` and `handler.go`.

### Go 1.22+ routing

TypeScript uses Node's `http` module with manual path matching:

```ts
// typescript/src/http/handler.ts
const documentId = matchDocumentPath(request.path);
if (documentId && request.method === "GET") { ... }
```

Go 1.22+ `ServeMux` handles method + path patterns directly:

```go
// go/internal/httpserver/server.go
mux := http.NewServeMux()
mux.HandleFunc("GET /health", h.handleHealth)
mux.HandleFunc("POST /documents", h.handleCreateDocument)
mux.HandleFunc("GET /documents/{id}", h.handleGetDocument)
mux.HandleFunc("PATCH /documents/{id}", h.handleUpdateDocument)
```

Path variables are extracted with:

```go
// go/internal/httpserver/handler.go
id := r.PathValue("id")
```

No external router required.

### JSON helpers

```go
// go/internal/httpserver/server.go
func writeJSON(w http.ResponseWriter, status int, body any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(body)
}

func readJSON(r *http.Request, dst any) error {
    return json.NewDecoder(r.Body).Decode(dst)
}
```

`encoding/json` is in the standard library. `json.NewEncoder` streams directly
to the `ResponseWriter` without building an intermediate string. `json.NewDecoder`
reads from the request body the same way.

### Error dispatch with `errors.As`

```go
// go/internal/httpserver/handler.go
func (h *handler) writeError(w http.ResponseWriter, err error) {
    var forbidden *domain.ForbiddenError
    if errors.As(err, &forbidden) {
        writeJSON(w, http.StatusForbidden, errorBody(err.Error()))
        return
    }

    var notFound *domain.DocumentNotFoundError
    if errors.As(err, &notFound) {
        writeJSON(w, http.StatusNotFound, errorBody(err.Error()))
        return
    }

    writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
}
```

This is the Go equivalent of the TypeScript `errorResponse` function. The domain
layer returns typed errors; the HTTP layer maps them to status codes. Neither
layer knows about the other's details.

---

## The composition root

Open `go/internal/app/app.go`. This is the only file that imports both `authz`
and `domain` concrete types. Everything else depends only on interfaces.

```go
// go/internal/app/app.go
func New(ctx context.Context) (*App, error) {
    // authz layer — concrete types
    tupleStore := authz.NewInMemoryTupleStore(fixtures.SeedRelationshipTuples()...)
    authorizer := authz.NewGraphAuthorizer(tupleStore)

    // domain layer — depends only on authz.Authorizer interface
    repo := domain.NewInMemoryDocumentRepository()
    docs := domain.NewDocumentService(repo, authorizer)

    // seed demo document
    _, err := docs.Create(ctx, domain.CreateDocumentInput{
        ID:        "roadmapDocument",
        Title:     "Roadmap",
        Body:      "Initial roadmap document",
        Workspace: fixtures.ProductWorkspace,
        Actor:     fixtures.WorkspaceEditor,
    })
    if err != nil {
        return nil, fmt.Errorf("app: seed demo document: %w", err)
    }

    // HTTP layer — depends only on domain.DocumentOperations interface
    handler := httpserver.NewServer(docs)

    // ...
    return &App{Handler: handler, Port: port}, nil
}
```

To swap `GraphAuthorizer` for `OpenFGAAuthorizer`, change one line:

```go
// authorizer := authz.NewGraphAuthorizer(tupleStore)
authorizer := authz.NewOpenFGAAuthorizer(authz.OpenFGAConfig{
    APIURL:  "http://localhost:8080",
    StoreID: "your-store-id",
})
```

Nothing else changes. The domain and HTTP layers never find out.

---

## Tests — AAA style

Open all three test files. Each test follows the Arrange → Act → Assert pattern
with explicit section comments.

### `graph_test.go` — unit tests for the graph traversal

```go
// go/internal/authz/graph_test.go
func TestGraphAuthorizer_TeamMemberCanEditDocument(t *testing.T) {
    // Arrange: workspaceEditor is a member of platformTeam, which is an editor of
    // productWorkspace. roadmapDocument lives in productWorkspace.
    store := seedStore()
    auth := authz.NewGraphAuthorizer(store)
    req := authz.CheckRequest{
        User:     fixtures.WorkspaceEditor,
        Relation: authz.RelationDocumentCanEdit,
        Object:   fixtures.RoadmapDocument,
    }

    // Act
    result, err := auth.Check(context.Background(), req)

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

The trace is printed only when the test fails (`t.Logf`), so it doesn't clutter
passing runs. When the test fails you can read every step of the graph walk.

### `service_test.go` — unit tests for the domain service

```go
// go/internal/domain/service_test.go
func TestDocumentService_Update_ForbiddenForViewer(t *testing.T) {
    // Arrange: workspaceViewer has viewer, not editor — update must be denied.
    svc := newSeededService(t)
    input := domain.UpdateDocumentInput{
        ID:    "roadmapDocument",
        Body:  "should not save",
        Actor: fixtures.WorkspaceViewer,
    }

    // Act
    _, err := svc.Update(context.Background(), input)

    // Assert
    if err == nil {
        t.Fatal("expected ForbiddenError but got nil")
    }
    var forbidden *domain.ForbiddenError
    if !errors.As(err, &forbidden) {
        t.Errorf("expected *ForbiddenError, got %T: %v", err, err)
    }
}
```

`newSeededService` is a test helper that wires the full stack and seeds the
roadmap document. Marking it with `t.Helper()` means failure lines point at the
test function, not inside the helper.

### `handler_test.go` — integration tests via `httptest`

The handler tests use `net/http/httptest` from the standard library — no test
server started, no network involved:

```go
// go/internal/httpserver/handler_test.go
func TestHandler_GetDocument_Returns200ForViewer(t *testing.T) {
    // Arrange
    handler := newTestHandler(t)
    req := httptest.NewRequest(http.MethodGet, "/documents/roadmapDocument?actorId=workspaceViewer", nil)
    rec := httptest.NewRecorder()

    // Act
    handler.ServeHTTP(rec, req)

    // Assert
    if rec.Code != http.StatusOK {
        t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
    }
    var resp map[string]any
    if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
        t.Fatalf("decode response: %v", err)
    }
    doc, ok := resp["document"].(map[string]any)
    if !ok {
        t.Fatalf("expected 'document' to be an object, got %T", resp["document"])
    }
    if doc["id"] != "roadmapDocument" {
        t.Errorf("expected id=%q, got %v", "roadmapDocument", doc["id"])
    }
}
```

`httptest.NewRecorder()` captures the response. `httptest.NewRequest()` builds
a request. `handler.ServeHTTP(rec, req)` calls the handler in-process. The
response is available in `rec.Code` and `rec.Body` immediately after.

Note that `doc["id"]` uses `"id"` (lowercase) because the `CollaborativeDocument`
struct has `json:"id"` tags. Without those tags the field would serialize as `"ID"`.

---

## Running the Go server

```bash
make go-server   # starts the server on port 4001
```

Then test the API:

```bash
# Health check
curl http://localhost:4001/health

# Read the pre-seeded document as the workspace viewer
curl "http://localhost:4001/documents/roadmapDocument?actorId=workspaceViewer"

# Try to read as an outsider (403)
curl "http://localhost:4001/documents/roadmapDocument?actorId=outsideCollaborator"

# Create a document as the workspace editor
curl -X POST http://localhost:4001/documents \
  -H "Content-Type: application/json" \
  -d '{"id":"notes","title":"Notes","body":"hello","workspaceId":"productWorkspace","actorId":"workspaceEditor"}'

# Try to update as the workspace viewer (403)
curl -X PATCH http://localhost:4001/documents/roadmapDocument \
  -H "Content-Type: application/json" \
  -d '{"body":"should not save","actorId":"workspaceViewer"}'
```

The same HTTP API as the TypeScript server (`make ts-server`), just on port 4001.

---

## Side-by-side comparison

| Concern               | TypeScript (`typescript/src/`)         | Go (`go/internal/`)                      |
|-----------------------|----------------------------------------|------------------------------------------|
| Branded types         | Template literal types                 | Named types (`type Object string`)       |
| Relation constants    | Union type `"can_edit" \| "viewer"…`   | `const` block with named `Relation` type |
| Interface satisfaction| `implements Authorizer`                | Implicit — method set must match         |
| Error signalling      | `throw new ForbiddenError(...)`        | `return &ForbiddenError{...}`            |
| Error dispatch        | `instanceof`                           | `errors.As`                              |
| Constructor injection | `constructor(private repo: ...)`       | `NewDocumentService(repo, auth)`         |
| Immutable copy        | `{ ...existing, body: newBody }`       | `updated := *existing; updated.Body = …` |
| Async / cancellation  | `async`/`await`, `Promise`             | Synchronous + `context.Context`          |
| JSON serialization    | Automatic                              | Struct tags: `json:"fieldName"`          |
| HTTP routing          | Manual `if method && path`             | Go 1.22+ `ServeMux` patterns             |
| Test assertions       | Vitest `expect(x).toBe(y)`             | `if x != y { t.Errorf(...) }`           |
| Test recorder         | Vitest mocking                         | `httptest.NewRecorder()`                 |

---

## Exercise

The `OpenFGAAuthorizer` in `go/internal/authz/openfga.go` is a stub. Complete it:

1. Inside the `go/` directory run `go get github.com/openfga/go-sdk@latest` (or
   use `make go-shell` to enter the container first).
2. Implement `Check` using the SDK's `client.Check()` method — the pattern is the
   same as `typescript/src/authz/openfga-client.ts`.
3. Wire it into `app/app.go` by replacing `NewGraphAuthorizer` with
   `NewOpenFGAAuthorizer`, then run `make openfga-up` and confirm the server
   still responds to the same `curl` requests.

---

## Checkpoint

`NewDocumentService` returns `DocumentOperations` (an interface) instead of
`*documentService` (the concrete type). Why does this matter?

Good answer: callers can only use methods declared on `DocumentOperations`. They
cannot inspect the struct's fields, call any unexported or extra methods, or
construct one without going through the constructor. Combined with the `internal/`
directory restriction (only code inside `rebac-primer` can import
`rebac-primer/internal/domain`), this gives Go the same layered encapsulation
that TypeScript achieves with `private` modifiers and barrel-file access control.
