# Go: how an authorization check flows through the code

This chapter traces a single HTTP request end to end and shows **every function
call** it makes on the way to an allow/deny decision. Read it with the files
open. By the end you should be able to point at any layer and say what it does
and what it calls next.

It complements two other chapters:

- `docs/21-go-rebac-implementation.md` — the package map and what each file is.
- `docs/27-graph-evaluator-walkthrough.md` — the recursion *inside* the evaluator.

This chapter is the glue between them: how a request reaches the evaluator and
how the answer gets back to the client.

---

## The request we are tracing

```text
GET /documents/roadmapDocument
Authorization: Bearer demo-token-bob
```

Bob is a viewer of the workspace the roadmap lives in, so this should return
**200** with the document. The same request as `PATCH` (an edit) returns
**403**, because Bob is not an editor. We trace the read in full, then show where
the 403 branches off.

---

## The shape of the system

Everything runs in **one process**. The Go binary contains both the documents
domain and the authz engine. They talk through an interface, not a network:

```text
                          one Go process (cmd/server)
 ┌───────────────────────────────────────────────────────────────────────┐
 │                                                                         │
 │  HTTP (documents)        documents domain            authz core         │
 │  ┌───────────────┐       ┌────────────────┐         ┌───────────────┐   │
 │  │ handler.go    │──────►│ read.go         │────────►│ authz.Service │   │
 │  │ (ServeMux)    │       │ domain.go       │ Check() │ (domain.go)   │   │
 │  └──────┬────────┘       └────────────────┘   ▲     └──────┬────────┘   │
 │         │ authn                                │            │ Evaluate() │
 │         ▼                                 AuthzClient        ▼           │
 │  ┌───────────────┐                       (interface)  ┌───────────────┐ │
 │  │ authn/        │                                     │ graph/        │ │
 │  │ verifier.go   │                                     │ evaluator.go  │ │
 │  └───────────────┘                                     └──────┬────────┘ │
 │                                                               │ reads     │
 │                                                        ┌──────▼────────┐ │
 │                                                        │ db/store.go   │ │
 │                                                        │ (tuples)      │ │
 │                                                        └───────────────┘ │
 └───────────────────────────────────────────────────────────────────────┘
```

The dashed boundary `AuthzClient (interface)` is the important part. The
documents domain depends on an **interface**, and the concrete thing on the
other side happens to be the authz service in the same process. (In TypeScript
that same boundary is an HTTP call — see `docs/29-typescript-authz-call-flow.md`.)

---

## The call flow, step by step

Each step names the file and function, the call it makes, and what comes back.

### 0. Wiring — `cmd/server/main.go`

`buildHandler` is the composition root. It is the only place that knows the
concrete types:

```go
tupleStore := authzdb.New(fixtures.SeedRelationshipTuples()...)
evaluator  := graph.NewGraphEvaluator(tupleStore)
authzSvc   := authz.New(tupleStore, evaluator)   // type: authz.Service

docRepo       := docsdb.New()
tokenVerifier := docsauthn.New(fixtures.DemoTokens())
docsSvc       := documents.New(docRepo, authzSvc) // authzSvc passed as AuthzClient
```

The last line is the whole trick: `authzSvc` is an `authz.Service`, which has
`Check` and `WriteTuples` (plus more). `documents.AuthzClient` requires exactly
`Check` and `WriteTuples`. By Go's structural typing, `authz.Service` satisfies
`documents.AuthzClient` automatically — no adapter, no declaration. So the
documents domain calls the authz engine **directly**.

### 1. HTTP routing — `documents/adapters/http/server.go`

`NewServer` registers the route on a Go 1.22 `ServeMux`:

```go
mux.HandleFunc("GET /documents/{id}", h.handleGetDocument)
```

The mux matches `GET /documents/roadmapDocument` and calls `handleGetDocument`.

### 2. Authn + dispatch — `documents/adapters/http/handler.go`

```go
func (h *handler) handleGetDocument(w http.ResponseWriter, r *http.Request) {
    user, err := h.authenticator.VerifyAccessToken(r.Header.Get("Authorization"))  // ── step 3
    if err != nil { h.writeError(w, err); return }

    id := r.PathValue("id")                       // "roadmapDocument"
    doc, err := h.docs.Read(r.Context(), id, user.Subject)  // ── step 4
    if err != nil { h.writeError(w, err); return }

    writeJSON(w, http.StatusOK, map[string]any{"document": doc})  // ── step 9
}
```

### 3. Authn — `documents/adapters/authn/verifier.go`

`VerifyAccessToken` strips `Bearer `, looks the token up in the demo map, and
returns `AuthenticatedUser{Subject: "user:bob"}`. (In production this verifies a
JWT instead — the port is unchanged.) **This establishes *who* is asking. The
*what-can-they-do* question is the authz check in step 5.**

### 4. Use case — `documents/read.go`

```go
func (s *documentService) Read(ctx, id, actor) (*CollaborativeDocument, error) {
    doc, err := s.requireDocument(ctx, id)        // exists? else DocumentNotFoundError (404)
    if err != nil { return nil, err }

    if err := s.requireAllowed(ctx, actor, shared.RelationDocumentCanRead,
        shared.Document(id), "read"); err != nil {   // ── step 5
        return nil, err
    }
    return doc, nil
}
```

Note the order: **existence is checked before authorization**, so a missing
document returns 404 (not found), not 403 (forbidden).

### 5. The authorization boundary — `documents/domain.go`

```go
func (s *documentService) requireAllowed(ctx, actor, relation, object, action) error {
    result, err := s.authzClient.Check(ctx, shared.CheckRequest{
        User: actor, Relation: relation, Object: object,
    })
    if err != nil { return err }
    if !result.Allowed {
        return &ForbiddenError{Message: fmt.Sprintf("%s cannot %s %s", actor, action, object)}
    }
    return nil
}
```

`s.authzClient` is the interface. The call `s.authzClient.Check(...)` lands on
`authz.Service.Check` **in the same process** — an ordinary method call.

### 6. Authz core — `authz/domain.go`

```go
func (d *authzDomain) Check(ctx, req) (shared.CheckResult, error) {
    return d.evaluator.Evaluate(ctx, req)   // delegates to the Evaluator port
}
```

The authz core does no graph work itself; it delegates to whatever `Evaluator`
was wired in (the graph evaluator today; an OpenFGA client tomorrow — see
`docs/26-openfga-migration.md`).

### 7. Graph traversal — `authz/adapters/graph/evaluator.go`

`Evaluate` runs the recursive `hasRelation` search:
`can_read → viewer → (workspace inheritance) → workspace:productWorkspace viewer
→ Bob is a direct viewer → allowed`. It returns:

```go
shared.CheckResult{Allowed: true, Trace: [...]}
```

For the line-by-line recursion (and the trace lines it produces), read
`docs/27-graph-evaluator-walkthrough.md`. The evaluator reads tuples from
`db/store.go` (step 7→8) but never writes — it only answers questions.

### 8. The answer propagates back

```text
evaluator.Evaluate → {Allowed:true}      (graph/evaluator.go)
  authzDomain.Check returns it           (authz/domain.go)
    requireAllowed sees Allowed → nil err (documents/domain.go)
      Read returns the document           (documents/read.go)
        handler writes 200 + JSON         (documents/adapters/http/handler.go)
```

### 9. … and becomes an HTTP status

`writeJSON(w, http.StatusOK, {"document": doc})` → **200**.

---

## Where the 403 comes from (the same flow, denied)

For `PATCH /documents/roadmapDocument` as Bob, step 4 is `Update` (in
`update.go`) and step 5 checks `can_edit` instead of `can_read`. Bob is a
*viewer*, so the graph returns `Allowed: false`. Then:

```go
// documents/domain.go
if !result.Allowed {
    return &ForbiddenError{Message: "user:bob cannot edit document:roadmapDocument"}
}
```

That error travels back up to the HTTP handler, where `writeError` maps the
domain error to a status code:

```go
// documents/adapters/http/handler.go
var forbidden *documents.ForbiddenError
if errors.As(err, &forbidden) {
    writeJSON(w, http.StatusForbidden, errorBody(err.Error()))   // 403
    return
}
```

So the **domain decides allow/deny** (typed errors) and the **HTTP adapter
decides the status code**. Neither layer knows the other's vocabulary. The full
mapping in `writeError`:

| Domain outcome | Error type | HTTP status |
|---|---|---|
| Bad/absent token | `AuthenticationError` | 401 |
| Forbidden by authz | `ForbiddenError` | 403 |
| Document missing | `DocumentNotFoundError` | 404 |
| Anything else | (any `error`) | 500 (logged server-side, generic body) |

---

## Layer summary

| Layer | File | Responsibility | Calls next |
|---|---|---|---|
| HTTP in | `documents/adapters/http/handler.go` | parse request, map errors→status | authn, then domain |
| Authn | `documents/adapters/authn/verifier.go` | token → identity | (returns) |
| Use case | `documents/read.go` / `update.go` | orchestrate: exists? allowed? | `requireAllowed` |
| Domain boundary | `documents/domain.go` | call authz, raise `ForbiddenError` | `authzClient.Check` |
| Authz core | `authz/domain.go` | delegate to evaluator | `evaluator.Evaluate` |
| Evaluator | `authz/adapters/graph/evaluator.go` | traverse the tuple graph | `store` reads |
| Store | `authz/adapters/db/store.go` | hold tuples | (returns) |

---

## The one idea to take away

The documents domain says `s.authzClient.Check(...)` and does not care what is
on the other side. Today it is a method call to the in-process authz engine.
Swap the wiring in `main.go` and it becomes a call to a remote OpenFGA server —
**the domain code does not change a line**. That is the payoff of depending on
the `AuthzClient` interface instead of a concrete evaluator.

The same request in the TypeScript implementation crosses an HTTP boundary
between two services. Reading `docs/29-typescript-authz-call-flow.md` side by
side shows how the identical domain logic sits behind both an in-process call
and a network call.

---

## Try it

1. Start the server: `make go/server` (listens on 4001).
2. Read as Bob (allowed): `curl :4001/documents/roadmapDocument -H "Authorization: Bearer demo-token-bob"` → 200.
3. Edit as Bob (denied): `curl -X PATCH :4001/documents/roadmapDocument -H "Authorization: Bearer demo-token-bob" -H "content-type: application/json" -d '{"body":"x"}'` → 403.
4. Open `documents/domain.go` and add a `log.Printf` inside `requireAllowed` printing the `CheckRequest`. Re-run both curls and watch the relation change from `can_read` to `can_edit`.

## Checkpoint

1. Which file turns "Bob is denied" into the number `403`, and which file *decided* he was denied?
2. `documents.New(docRepo, authzSvc)` passes an `authz.Service` where an
   `AuthzClient` is expected. Why does that compile with no conversion?
3. In step 5, `requireDocument` runs before `requireAllowed`. What would break if
   you swapped the order?

Good answers:
1. `documents/adapters/http/handler.go` (`writeError`) maps it to 403; the
   decision was made in `documents/domain.go` (`requireAllowed` returning
   `ForbiddenError` after the authz check came back `Allowed: false`).
2. `authz.Service` has `Check` and `WriteTuples` (among others), which is a
   superset of `AuthzClient`'s methods. Go interface satisfaction is structural,
   so it satisfies `AuthzClient` automatically.
3. You would leak existence information: a request for a document the user
   cannot see would return 403 (telling them it exists) instead of 404. Checking
   existence first keeps not-found and forbidden honest.
