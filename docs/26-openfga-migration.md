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

3. **The service port stays** — domain code (`documents/`) depends on the
   `authz.Service` behavior it needs: check permissions and write relationship
   tuples. The in-process backend builds that service from a graph evaluator and
   an in-memory store; the OpenFGA adapter implements the whole service directly.
   Swapping backends is a composition-root choice in `cmd/server/main.go`.

---

## Concept mapping

Every piece of our scratch implementation has a direct OpenFGA counterpart.

| Our code | OpenFGA concept | Notes |
|---|---|---|
| `rebac.TupleKey` | Relationship tuple | Same structure: `(object, relation, user)` |
| `rebac.Object` (`"user:alice"`) | Object ID | Same `type:id` format |
| `rebac.Subject` with `#` (`"team:platformTeam#member"`) | Subject set (userset) | Same `object#relation` format |
| `rebac.CheckRequest` | Check call parameters | Same three fields |
| `authz.TupleRepository` | Tuple store | We own the interface; OpenFGA owns the store |
| `authz.InMemoryStore` | (our own, not in OpenFGA) | In-memory only; OpenFGA uses PostgreSQL or MySQL |
| `authz.Evaluator` | Check engine | In-process only; OpenFGA does this inside the server |
| `authz.GraphEvaluator` | OpenFGA's check engine | Our traversal mirrors what OpenFGA does internally |
| `authz/model.go` (`teamRules`, `workspaceRules`, `documentRules`) | Authorization model DSL | We express rules as Go maps; OpenFGA uses a DSL stored in a database |
| `authz.Service` | OpenFGA-backed authz service | The app-facing port: Check, WriteTuples, DeleteTuples, ListTuples |

### The permission model side by side

Our Go table for documents (`model.go`):

```go
var documentRules = impliedBy{
    rebac.RelationDocumentCanRead:    {rebac.RelationDocumentViewer},
    rebac.RelationDocumentCanComment: {rebac.RelationDocumentViewer},
    rebac.RelationDocumentCanEdit:    {rebac.RelationDocumentEditor},
    rebac.RelationDocumentCanDelete:  {rebac.RelationDocumentOwner},
    rebac.RelationDocumentViewer:     {rebac.RelationDocumentEditor},
    rebac.RelationDocumentEditor:     {rebac.RelationDocumentOwner},
}
```

The same rules in OpenFGA DSL:

```text
type document
  relations
    define workspace:  [workspace]
    define owner:      [user] or owner from workspace
    define editor:     [user] or editor from workspace or owner
    define viewer:     [user] or viewer from workspace or editor
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
define editor: [user] or editor from workspace or owner
                         ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                         OpenFGA traverses this; we wrote it in Go
```

---

## Architectural fit

The key is that the documents service talks to an authz **service**, not to a
specific evaluator or tuple store:

```go
type Service interface {
    Check(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error)
    WriteTuples(ctx context.Context, tuples []rebac.TupleKey) error
    DeleteTuples(ctx context.Context, tuples []rebac.TupleKey) error
    ListTuples(ctx context.Context, filter ...TupleFilter) ([]rebac.TupleKey, error)
}
```

Both backends satisfy that same app-facing shape, but they get there differently:

```text
                         authz.Service
                              │
              ┌───────────────┴──────────────────┐
              │                                  │
  authz.New(store, evaluator)            openfga.Service
  (in-process service)                   (SDK adapter)
              │                                  │
  GraphEvaluator + InMemoryStore         OpenFGA Check/Write/Read APIs
```

The documents service sees only `documents.AuthzClient`, which `authz.Service`
satisfies. The dependency chain is:

```text
cmd/server/main.go   ← only place that knows about concrete types
    └── choose graph authz.Service OR OpenFGA authz.Service
            └── documents.New(repo, authzService)
```

---

## How to use the OpenFGA backend

**This is already implemented.** The OpenFGA adapter and the flag-driven wiring
ship in the repo, so the swap is a runtime flag, not a code change:

- Go adapter: `go/internal/openfga/openfga.go` (implements `authz.Service`)
- TS adapter: `typescript/src/authz-service/adapters/openfga/makeOpenFgaAuthzService.ts`
- Model: `deployments/openfga/model.fga` · Seed: `deployments/openfga/seed.sh`

The composition root reads `AUTHZ_BACKEND`: `openfga` selects the adapter (using
`OPENFGA_API_URL` / `OPENFGA_STORE_ID` / `OPENFGA_MODEL_ID`); anything else uses
the in-process graph evaluator. Both return an `authz.Service` / `AuthzService`,
so the documents domain, HTTP handlers, and tests are unchanged.

Run it (the make target sets the flag for you):

```bash
make openfga/up         # OpenFGA on :8080 (memory datastore)
make openfga/seed       # create store, write model.fga, seed policy tuples (needs fga CLI + jq)
make go/server-openfga  # or: make ts/server-openfga
```

The sections below explain what each step does under the hood.

## Bootstrapping: what exists when?

OpenFGA has three separate things, and bootstrapping is just creating them in the
right order:

| Step | What exists after this step | Where it lives |
|---|---|---|
| `make openfga/up` | An empty OpenFGA server | Docker container, memory datastore |
| `make openfga/seed` creates a store | A namespace called `rebac-primer` | OpenFGA store |
| `make openfga/seed` writes `model.fga` | The policy schema: object types, relations, inheritance, computed permissions | OpenFGA authorization model |
| `make openfga/seed` writes policy tuples | Demo workspace/team facts: Alice is in a team, that team edits the workspace, Bob views the workspace | OpenFGA tuple store |
| `make openfga/seed` writes `.ids.env` | The API URL, store ID, and model ID the app must use | `deployments/openfga/.ids.env` |
| `make go/server-openfga` starts the app | The app connects to that store/model and creates the demo document | Go process + OpenFGA tuple store |

The last row matters: `seed.sh` does **not** write the document-level tuples.
When the server starts, `cmd/server/main.go` calls `documentsService.Create(...)`
to create `roadmapDocument`. That domain operation calls `authzService.WriteTuples`,
so in OpenFGA mode the adapter writes:

```text
(document:roadmapDocument, workspace, workspace:productWorkspace)
(document:roadmapDocument, owner,     user:alice)
```

So there are two kinds of tuples:

```text
bootstrap tuples  -> long-lived/demo policy facts, written by seed.sh
runtime tuples    -> facts created by app behavior, written through WriteTuples
```

The local Docker setup uses OpenFGA's memory datastore, so all of this disappears
when the OpenFGA container restarts. That is why `make openfga/seed` writes fresh
IDs each time and why the server targets source `.ids.env`.

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

### Step 3 — the OpenFGA adapter

The adapter ships at `go/internal/openfga/openfga.go` (and the TS
equivalent), and `github.com/openfga/go-sdk` is already in `go.mod`. It implements
the full `authz.Service` driving port — `Check`, `WriteTuples`, `DeleteTuples`,
`ListTuples` — **not** the inner `Evaluator` port. That choice is deliberate:
`Evaluator` only covers checks, and the in-memory `TupleRepository.Write` is
synchronous with no `ctx`/error, a poor fit for a network backend. `authz.Service`
has `ctx` + error on every method, so it is the right seam to back the whole authz
service with OpenFGA — checks and tuple writes both go to the store, staying
consistent.

Here is the shape of the check path (the real file also implements the write/read
methods):

```go
package openfga

import (
    "context"
    "fmt"

    openfga "github.com/openfga/go-sdk/client"

    "rebac-primer/internal/authz"
    "rebac-primer/internal/rebac"
)

type Config struct {
    APIURL  string
    StoreID string
    ModelID string
}

// Service satisfies authz.Service by delegating to an OpenFGA server.
type Service struct {
    client *openfga.OpenFgaClient
}

var _ authz.Service = (*Service)(nil)

func New(cfg Config) (*Service, error) {
    client, err := openfga.NewSdkClient(&openfga.ClientConfiguration{
        ApiUrl:               cfg.APIURL,
        StoreId:              cfg.StoreID,
        AuthorizationModelId: cfg.ModelID,
    })
    if err != nil {
        return nil, fmt.Errorf("openfga: new client: %w", err)
    }
    return &Service{client: client}, nil
}

func (s *Service) Check(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error) {
    resp, err := s.client.Check(ctx).Body(openfga.ClientCheckRequest{
        User:     string(req.User),
        Relation: string(req.Relation),
        Object:   string(req.Object),
    }).Execute()
    if err != nil {
        return rebac.CheckResult{}, fmt.Errorf("openfga: check: %w", err)
    }
    return rebac.CheckResult{Allowed: resp.GetAllowed(), Trace: []string{"OpenFGA evaluated remotely"}}, nil
}
```

`WriteTuples` (and `DeleteTuples`, `ListTuples`) round-trip to the store so that
tuples written at runtime are visible to subsequent checks:

```go
func (s *Service) WriteTuples(ctx context.Context, tuples []rebac.TupleKey) error {
    writes := make([]openfga.ClientTupleKey, 0, len(tuples))
    for _, t := range tuples {
        writes = append(writes, openfga.ClientTupleKey{
            User: string(t.User), Relation: string(t.Relation), Object: string(t.Object),
        })
    }
    _, err := s.client.Write(ctx).Body(openfga.ClientWriteRequest{Writes: writes}).Execute()
    if err != nil {
        return fmt.Errorf("openfga: write tuples: %w", err)
    }
    return nil
}
```

### Step 4 — select the backend (already wired)

`go/cmd/server/main.go` chooses the backend from `AUTHZ_BACKEND` inline — no
hand-editing required:

```go
var authzService authz.Service
if os.Getenv("AUTHZ_BACKEND") == "openfga" {
    authzService, err = openfga.New(openfga.Config{
        APIURL:  envOr("OPENFGA_API_URL", "http://127.0.0.1:8080"),
        StoreID: os.Getenv("OPENFGA_STORE_ID"),
        ModelID: os.Getenv("OPENFGA_MODEL_ID"),
    })
} else {
    store := authz.NewInMemoryStore(fixtures.SeedRelationshipTuples()...)
    authzService = authz.New(store, authz.NewGraphEvaluator(store))
}
```

The TypeScript side does the same in `authz-service/compose.ts`. Everything else
— the documents service, the HTTP handler, the tests — is unchanged.

### Step 5 — seed the store

`deployments/openfga/seed.sh` (run via `make openfga/seed`) creates the store,
writes `model.fga`, and seeds the workspace/team **policy** tuples — the same ones
`fixtures.SeedRelationshipTuples` / `seedPolicyTuples` hold for the in-memory
backend. The **document-level** tuples (`workspace`, `owner`) are written at
runtime by the documents service on create; with OpenFGA they just land in the
store. Because the demo OpenFGA uses the ephemeral `memory` datastore, re-run the
seed whenever you restart OpenFGA.

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

Nothing is thrown away. The `authz.Service` interface, the in-process
`authz.Evaluator` implementation, the documents domain, and the HTTP layer all
still exist. The only thing that changes is which concrete `authz.Service` is
plugged into `documents.New(...)` in `main.go`.

That is the payoff of the ports-and-adapters design: you can replace the
persistence and evaluation strategy without touching the business logic.
