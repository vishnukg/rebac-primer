# From scratch to OpenFGA

This project implements ReBAC in two layers:

1. **From scratch** — a hand-rolled graph evaluator and in-memory tuple store, no
   external service needed. This is what `go run ./cmd/server` runs today.
2. **OpenFGA** — the same model expressed in OpenFGA's DSL, evaluated by the
   OpenFGA server. The architecture is designed so you can swap to this with a
   one-line wiring change.

Read this document to understand what each piece of our scratch implementation
does, what the equivalent OpenFGA concept is, and how to perform the migration.

---

## Why we built from scratch first

The from-scratch implementation exists for three reasons:

1. **Learning** — reading a graph traversal you wrote yourself is the fastest way
   to understand how `user:alice` flows through four tuples to get `can_edit` on
   a document. OpenFGA does this internally; we just made it visible.

2. **No server dependency** — the in-process graph runs inside the Go binary. You
   can run all tests with `go test ./...` without Docker, without a network, and
   without an OpenFGA server.

3. **The port stays** — domain code (`documents/`, `authz/`) depends only on the
   `authz.Evaluator` interface. The graph evaluator and the OpenFGA SDK adapter
   both satisfy that interface. Swapping one for the other changes exactly one
   line in `cmd/server/main.go`.

---

## Concept mapping

Every piece of our scratch implementation has a direct OpenFGA counterpart.

| Our code | OpenFGA concept | Notes |
|---|---|---|
| `shared.TupleKey` | Relationship tuple | Same structure: `(object, relation, user)` |
| `shared.Object` (`"user:alice"`) | Object ID | Same `type:id` format |
| `shared.Subject` with `#` (`"team:platformTeam#member"`) | Subject set (userset) | Same `object#relation` format |
| `shared.CheckRequest` | Check call parameters | Same three fields |
| `authz.TupleRepository` | Tuple store | We own the interface; OpenFGA owns the store |
| `authz/adapters/db.InMemoryTupleStore` | (our own, not in OpenFGA) | In-memory only; OpenFGA uses PostgreSQL or MySQL |
| `authz.Evaluator` | Check API | Our port; OpenFGA's HTTP/gRPC endpoint is the adapter |
| `authz/adapters/graph.GraphEvaluator` | OpenFGA's check engine | Our traversal mirrors what OpenFGA does internally |
| `graph/permissionmodel.go` (`teamRules`, `workspaceRules`, `documentRules`) | Authorization model DSL | We express rules as Go maps; OpenFGA uses a DSL stored in a database |
| `authz.New(store, evaluator)` | (wiring only) | No OpenFGA equivalent; OpenFGA merges store and evaluator in one server |

### The permission model side by side

Our Go table for documents (`permissionmodel.go`):

```go
var documentRules = impliedBy{
    shared.RelationDocumentCanRead:    {shared.RelationDocumentViewer},
    shared.RelationDocumentCanComment: {shared.RelationDocumentViewer},
    shared.RelationDocumentCanEdit:    {shared.RelationDocumentEditor},
    shared.RelationDocumentCanDelete:  {shared.RelationDocumentOwner},
    shared.RelationDocumentViewer:     {shared.RelationDocumentEditor},
    shared.RelationDocumentEditor:     {shared.RelationDocumentOwner},
}
```

The same rules in OpenFGA DSL:

```text
type document
  relations
    define workspace:  [workspace]
    define owner:      [user] or workspace#owner from workspace
    define editor:     [user] or workspace#editor from workspace or owner
    define viewer:     [user] or workspace#viewer from workspace or editor
    define can_read:    viewer
    define can_comment: viewer
    define can_edit:    editor
    define can_delete:  owner
```

The logic is identical. The difference is execution location:

- **Our evaluator** — runs in the same Go process, traverses the `TupleRepository`
  in memory, returns a `CheckResult` with a human-readable `Trace`.
- **OpenFGA** — runs as a separate server, stores tuples in a database, evaluates
  the DSL model, returns `allowed: true/false`.

One thing to notice: our `expandDocument` in `evaluator.go` hard-codes the
workspace inheritance logic in Go. In OpenFGA, the `from` keyword in the DSL
handles that:

```text
define editor: [user] or workspace#editor from workspace or owner
                         ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                         OpenFGA traverses this; we wrote it in Go
```

---

## Architectural fit

The key is that `authz.Evaluator` is an interface with one method:

```go
type Evaluator interface {
    Evaluate(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error)
}
```

Both our `GraphEvaluator` and an OpenFGA SDK adapter satisfy this interface.
The authz domain, the documents domain, and the HTTP layer never know which one
is plugged in.

```text
                       authz.Evaluator (interface)
                              │
              ┌───────────────┴─────────────────┐
              │                                 │
  graph.GraphEvaluator                 openfga.Authorizer
  (in-process, today)                 (SDK adapter, migration target)
              │                                 │
  reads InMemoryTupleStore          calls OpenFGA server over HTTP/gRPC
```

The documents service sees only `documents.AuthzClient`, which `authz.Service`
satisfies. The authz service sees only `authz.Evaluator`. The dependency chain is:

```text
cmd/server/main.go   ← only place that knows about concrete types
    └── authz.New(tupleStore, evaluator)   ← evaluator is swappable here
            └── documents.New(repo, authzSvc)
```

---

## How to migrate to OpenFGA

### Step 1 — run the OpenFGA server

The `docker-compose.yml` already has it:

```bash
# from the repo root
docker compose up openfga
```

OpenFGA is now at:
- HTTP API: `http://localhost:8080`
- Playground UI: `http://localhost:3000`

### Step 2 — create a store and upload the model

Use the playground at `http://localhost:3000` or the CLI:

```bash
# install the CLI
brew install openfga/tap/fga

# create a store
fga store create --name "rebac-primer"
# → prints a store ID, e.g. 01JXYZ...

# write the model (create a file model.fga with the DSL from doc 05-openfga-model.md)
fga model write --store-id 01JXYZ... --file model.fga
# → prints an authorization model ID
```

Note both IDs — you will need them in Step 4.

### Step 3 — restore the OpenFGA adapter

The adapter was removed from the repo to keep things simple, but it is easy to
restore. Add `github.com/openfga/go-sdk` back to `go.mod`:

```bash
cd go
go get github.com/openfga/go-sdk
```

Then create `go/internal/authz/adapters/openfga/authorizer.go`:

```go
package openfga

import (
    "context"
    "fmt"

    openfga "github.com/openfga/go-sdk"
    fgaclient "github.com/openfga/go-sdk/client"

    "rebac-primer/internal/authz"
    "rebac-primer/internal/shared"
)

type Config struct {
    APIURL               string
    StoreID              string
    AuthorizationModelID string
}

// Authorizer satisfies authz.Evaluator by delegating to an OpenFGA server.
type Authorizer struct {
    client *fgaclient.OpenFgaClient
}

var _ authz.Evaluator = (*Authorizer)(nil)

func New(cfg Config) (*Authorizer, error) {
    sdk, err := fgaclient.NewSdkClient(&fgaclient.ClientConfiguration{
        ApiUrl:               cfg.APIURL,
        StoreId:              cfg.StoreID,
        AuthorizationModelId: cfg.AuthorizationModelID,
    })
    if err != nil {
        return nil, fmt.Errorf("openfga: create sdk client: %w", err)
    }
    return &Authorizer{client: sdk}, nil
}

func (a *Authorizer) Evaluate(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error) {
    resp, err := a.client.Check(ctx).Body(fgaclient.ClientCheckRequest{
        User:     string(req.User),
        Relation: string(req.Relation),
        Object:   string(req.Object),
    }).Execute()
    if err != nil {
        return shared.CheckResult{}, fmt.Errorf("openfga: check: %w", err)
    }
    return shared.CheckResult{
        Allowed: resp.GetAllowed(),
        Trace:   []string{"OpenFGA evaluated the relationship graph remotely"},
    }, nil
}
```

The adapter also needs a `WriteTuples` method if you want the OpenFGA server to
hold the tuples (rather than keeping the in-memory store):

```go
func (a *Authorizer) WriteTuples(ctx context.Context, tuples []shared.TupleKey) error {
    sdkTuples := make([]fgaclient.ClientTupleKey, 0, len(tuples))
    for _, t := range tuples {
        sdkTuples = append(sdkTuples, *openfga.NewTupleKey(
            string(t.User), string(t.Relation), string(t.Object),
        ))
    }
    _, err := a.client.WriteTuples(ctx).Body(sdkTuples).Execute()
    if err != nil {
        return fmt.Errorf("openfga: write tuples: %w", err)
    }
    return nil
}
```

### Step 4 — swap the evaluator in main.go

Open `go/cmd/server/main.go`. In `buildHandler`, find these two lines:

```go
tupleStore := authzdb.New(fixtures.SeedRelationshipTuples()...)
evaluator  := graph.NewGraphEvaluator(tupleStore)
```

Replace them with:

```go
evaluator, err := openfga.New(openfga.Config{
    APIURL:               "http://localhost:8080",
    StoreID:              "<your-store-id>",
    AuthorizationModelID: "<your-model-id>",
})
if err != nil {
    return nil, fmt.Errorf("openfga evaluator: %w", err)
}
```

Also update the `authz.New` call — the tuple store is no longer needed because
OpenFGA stores the tuples:

```go
// Before (in-memory store)
authzSvc := authz.New(tupleStore, evaluator)

// After (OpenFGA holds the tuples; pass a no-op store or nil if you refactor the port)
authzSvc := authz.New(openfgaStore, evaluator)
```

> **Note** — `authz.New` takes a `TupleRepository` because our from-scratch service
> uses it for writes too. With OpenFGA you can either:
> - wire the `Authorizer` as both `TupleRepository` (via `WriteTuples`) and
>   `Evaluator` (a single struct satisfying both interfaces), or
> - keep a thin in-memory store for writes and let the graph evaluator read back
>   from OpenFGA (a hybrid, useful during migration).

Everything else — the documents service, the HTTP handler, the tests — stays
exactly as it is.

### Step 5 — seed the OpenFGA store

The fixture tuples need to be written to OpenFGA once (or on each deploy). You
can do this via the CLI:

```bash
fga tuple write --store-id 01JXYZ... \
  '{"object":"team:platformTeam","relation":"member","user":"user:alice"}' \
  '{"object":"workspace:productWorkspace","relation":"editor","user":"team:platformTeam#member"}' \
  '{"object":"workspace:productWorkspace","relation":"viewer","user":"user:bob"}' \
  '{"object":"document:roadmapDocument","relation":"workspace","user":"workspace:productWorkspace"}'
```

Or write a small seed script that calls `WriteTuples` via the adapter.

---

## What you gain from OpenFGA

| | From scratch | OpenFGA |
|---|---|---|
| Tuple persistence | In-memory, lost on restart | PostgreSQL / MySQL |
| Permission model changes | Edit Go code and redeploy | Upload new DSL version, no code change |
| Multi-tenant | Must shard manually | Multiple stores out of the box |
| Audit log | Our `Trace` field | Built-in, queryable |
| Scale | Single process | Horizontally scalable server |
| Debuggability | Our trace is readable | Playground UI, explain endpoint |
| Type safety | Compile-time (Go types) | Runtime (model validation) |

## What you keep from the from-scratch version

Nothing is thrown away. The `authz.Service` interface, the `authz.Evaluator` port,
the documents domain, and the HTTP layer are all unchanged. The only thing that
changes is the concrete type plugged into the `evaluator` slot in `main.go`.

That is the payoff of the ports-and-adapters design: you can replace the
persistence and evaluation strategy without touching the business logic.
