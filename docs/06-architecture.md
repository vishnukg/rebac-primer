# Architecture: ports and adapters

This is the one place that states the whole architecture: the shape both
implementations share, the rules they follow, the evidence that they actually
follow them, and the few places they bend the rules **on purpose** for teaching.

If you only read one architecture chapter, read this one. The language-specific
detail lives in `docs/21` (Go) and `docs/18`/`docs/19` (TypeScript); the request
call-flow lives in `docs/28` (Go) and `docs/29` (TypeScript).

> It works as an early preview, but it lands best once you've read at least one
> implementation track (Go: 20 → 21; TypeScript: 17 → 18) and its call-flow
> chapter (28 or 29). The course map lists it as a synthesis step for that reason.

---

## The shape

Both the Go and TypeScript codebases use **ports and adapters** (a.k.a.
hexagonal architecture). At runtime a request flows left → right, from a driving
adapter, through the core, out to driven adapters:

```text
       DRIVING SIDE                    CORE                       DRIVEN SIDE
   (adapters call in)          (domain + the ports)        (core calls out via ports)

  ┌───────────────┐        ╔══════════════════════════╗        ┌───────────────────┐
  │ HTTP handler  │ ─────► ║                          ║ ─────► │ graph evaluator   │
  ├───────────────┤        ║   Service  (driving port)║        ├───────────────────┤
  │ CLI client    │ ─────► ║                          ║ ─────► │ in-memory store   │
  ├───────────────┤ calls  ║   + use cases / domain   ║ calls  ├───────────────────┤
  │ tests         │ ─────► ║   + DRIVEN PORTS         ║ ─────► │ token verifier    │
  └───────────────┘        ╚══════════════════════════╝        ├───────────────────┤
                                       ▲                        │ authz client      │
                                       │ wires concretes        └───────────────────┘
                            ┌──────────┴───────────┐
                            │  composition root    │
                            │  main.go / compose.ts│
                            └──────────────────────┘
```

- A **driving port** is what the outside world calls (e.g. `documents.Service` /
  `Documents`). A driving adapter (the HTTP handler) calls *into* it.
- A **driven port** is what the core needs from infrastructure (e.g.
  `DocumentRepository`, `AuthzClient`, `Evaluator`). A driven adapter (in-memory
  DB, graph evaluator, HTTP client) *implements* it.
- The **core** depends only on its own ports and the shared vocabulary. It never
  imports a concrete adapter.
- The **composition root** (`cmd/server/main.go` in Go; each `compose.ts` in TS)
  is the single place that knows the concrete types and wires them to the ports.

This repo runs the pattern twice — once per service — because the system is two
services (an authz service and a documents service), each with its own core,
ports, and adapters.

### The arrows that matter: dependencies point inward

The diagram above shows **calls**. The thing that actually keeps the design
clean is the opposite arrow — **dependencies** (who imports/knows whom). Calls go
outward, but every dependency points *in toward the core*:

```text
   RUNTIME CALLS  ───────────────────────────────────────────────────►  (outward)
   COMPILE-TIME DEPENDENCIES  ◄───────────────────────────────────────   (inward)

      driving adapter                 CORE                 driven adapter
     ┌───────────────┐      ┌────────────────────────┐      ┌───────────────┐
     │ HTTP handler  │─────►│  use case              │      │ graph eval /  │
     │               │ call │   ┌───────────┐        │      │ in-mem store /│
     │ imports the   │─────►│   │  driven   │        │      │ authz client  │
     │ core's port   │ dep  │   │  PORT     │◄───────┼──────│  implements   │
     └───────────────┘      │   │(interface)│        │ dep  │  the port     │
                            │   └───────────┘        │      └───────────────┘
                            │  core OWNS the port,    │
                            │  imports nothing out    │
                            └────────────────────────┘
            │                                                 │
            └────────► both sides DEPEND ON the core ◄────────┘
                  the core depends on neither (only `shared`)
```

Read it as two arrows:
- **`─────►` (calls):** driving adapter → core → driven adapter (the request's path).
- **`◄─────` (depends on):** the driven adapter depends on the core's **port**,
  not the other way round. The core defines the interface; the adapter implements
  it. So the dependency arrow points *into* the core even though the call points
  out of it.

The **driven port is the hinge**. At runtime the core *calls* the adapter
(core → adapter). But the core *defines* the interface and the adapter
*implements* it, so the compile-time dependency runs adapter → core — the arrow
flips. That inversion is the whole trick: it lets you swap the graph evaluator
for an OpenFGA client, or an in-process authz client for an HTTP one, without the
core knowing or changing. (This is "dependency inversion": both sides depend on
the abstraction the core owns, not on each other.)

---

## Where everything lives (Go ↔ TypeScript)

Concretely, the two services wire up like this (arrows = "depends on / calls
through a port"; the **port boxes are owned by the core on their left**):

```text
  ┌───────────────────── documents-service ─────────────────────┐
  │                                                              │
  │  HTTP handler ──► Service ──► use cases (create/read/update) │
  │  (driving adapter)  (driving    │        │          │        │
  │                      port)      ▼        ▼          ▼         │
  │                          Authenticator  Document   AuthzClient│
  │                          [PORT]         Repository [PORT]     │
  │                             ▲           [PORT]         ▲      │
  │                  implements │              ▲ impl      │ impl │
  │                    ┌────────┴───┐   ┌──────┴─────┐  ┌──┴────────────┐
  │                    │ demo token │   │ in-memory  │  │ authz client  │
  │                    │ verifier   │   │ repository │  │ adapter       │
  │                    └────────────┘   └────────────┘  └──────┬────────┘
  └─────────────────────────────────────────────────────────── │ ───────┘
                                                                │ in Go: in-process call
                                       Go: structural typing    │ in TS: HTTP POST /check
                                       TS: HTTP / in-proc stub   ▼
  ┌───────────────────── authz-service ─────────────────────────────────┐
  │                                                                      │
  │  HTTP handler ──► Service ──► Evaluator ◄──implements── graph        │
  │  (driving adapter)  (check/   [PORT]                    evaluator    │
  │                      write/         │                   (adapter)    │
  │                      list)          ▼                       │        │
  │                              TupleRepository ◄──impl── in-memory     │
  │                              [PORT]                    tuple store   │
  └──────────────────────────────────────────────────────────────────────┘
```

The single seam between the services is the documents core's `AuthzClient`
**port**. In Go the `authz.Service` satisfies it directly (same process); in
TypeScript an HTTP client adapter satisfies it by calling `POST /check` on the
authz service. Same port, two transports — see `docs/28` / `docs/29`.

| Concept | Go | TypeScript |
|---|---|---|
| Shared vocabulary (the "SDK") | `internal/rebac/rebac.go` | `src/shared/rebac.ts` |
| AuthZ driving port | `authz.Service` (`internal/authz/authz.go`) | `AuthzService` (`authz-service/core/domain/types.ts`) |
| AuthZ driven ports | `TupleRepository`, `Evaluator` (`internal/authz/authz.go`) | `TupleRepository`, `Evaluator` (`authz-service/core/ports/`) |
| AuthZ core impl | `internal/authz/service.go` | `authz-service/core/domain/makeAuthzService.ts` |
| AuthZ implementations | `internal/authz/{store,evaluator,model}.go` + `internal/openfga` | `authz-service/adapters/{db,graph,http}` |
| Documents driving port | `documents.Service` (`internal/documents/documents.go`) | `Documents` (`documents-service/core/domain/types.ts`) |
| Documents driven ports | `DocumentRepository`, `AuthzClient`, `Authenticator` (`internal/documents/documents.go`) | same names (`documents-service/core/ports/`) |
| Documents use cases | `internal/documents/service.go` (`Create`/`Read`/`Update` methods) | `documents-service/core/domain/makeDocuments.ts` (`create`/`read`/`update` methods, inline) |
| Documents implementations | `internal/documents/{store,token}.go` + `internal/api` | `documents-service/adapters/{db,authn,authz,http,client}` |
| Composition root | `cmd/server/main.go` | `*/compose.ts` (+ `index.ts` entrypoints) |

The names line up almost one-to-one on purpose — reading the two side by side is
part of the lesson.

---

## The rules — and the proof they hold

### 1. Dependencies point inward: callers depend on interfaces, not concretes

This is the defining property. The two codebases express it differently:

- **TypeScript** keeps a literal `adapters/` directory, so "the core never imports
  an adapter" is a grep you can run:

  ```bash
  # should print nothing:
  grep -rn 'adapters/' typescript/src/authz-service/core/ typescript/src/documents-service/core/
  ```

- **Go** has no `adapters/` directory. Each package keeps its concrete types beside
  the interfaces they satisfy (e.g. `authz.InMemoryStore` lives in the same package
  as the `TupleRepository` interface it implements), and only `cmd/server/main.go`
  wires them. The swappable backends that *do* get their own packages —
  `internal/openfga` and `internal/api` — are imported only by `cmd/`:

  ```bash
  # should print nothing — only cmd/ imports these:
  grep -rn 'internal/openfga\|internal/api' go/internal/
  ```

Either way the principle is the same: the service logic names interfaces it owns,
and the concrete graph is assembled at the composition root (`cmd/server/main.go`
in Go; each `compose.ts` in TypeScript).

### 2. Ports are owned by the core that needs them

`authz` declares `TupleRepository` and `Evaluator` because the authz core is what
calls them. `documents` declares `DocumentRepository`, `AuthzClient`, and
`Authenticator` because the documents core is what calls *them*. A port is a
statement of need ("I require something that can do X"), so it belongs to the
caller, not the implementer.

### 3. Interface segregation: depend on the narrowest port

The documents service needs only two things from authz — "check a permission"
and "write tuples on create". So it defines its **own** 2-method port:

```go
// internal/documents/documents.go
type AuthzClient interface {
    Check(ctx, req) (CheckResult, error)
    WriteTuples(ctx, tuples) error
}
```

…even though the full `authz.Service` has four methods. The authz service
satisfies `AuthzClient` for free (Go structural typing / TS structural shape),
but the documents core only ever sees the slice it actually uses. This is why
the same domain code runs against an in-process authz service **and** a remote
HTTP one without changing.

### 4. Constructors return the interface; the root wires concretes

`documents.New(...)` returns `Service` (the interface), not `*documentService`.
`makeAuthzService(...)` returns `AuthzService`. Callers hold a port, never a
concrete struct. Go adds compile-time assertions (`var _ Port = (*Impl)(nil)`)
so a missing method fails at build time, not at the first call.

---

## Why the pattern earns its keep here

Because the documents core depends on the `AuthzClient` **port**, the transport
behind it is a wiring choice, not a code change:

| Wiring | Where | What `authzClient.Check` does |
|---|---|---|
| In-process (Go default) | `cmd/server/main.go` | a direct method call to `authz.Service` |
| In-process stub (TS tests) | `test/fixtures.ts` | runs the real graph evaluator, no socket |
| HTTP (TS default) | `documents-service/compose.ts` | `fetch(POST /check)` to the authz service |
| Remote OpenFGA (future) | swap one line | calls the OpenFGA server (`docs/26`) |

The domain logic is identical across all four. `docs/28` and `docs/29` trace a
real request through this boundary in each language.

---

## Intentional deviations (honest caveats)

A clean primer should be honest about where it trades architectural purity for
teaching value. These are deliberate, not accidents:

1. **The repo ships Go-language lessons, quarantined under `go/examples/`.**
   `Result[T]`/`Map`/`Collect` (generics, `examples/generics/`),
   `AllPermissions`/`BulkCheck` (concurrency, `examples/concurrency/`),
   `AuditEvaluator`/`ReadOnlyStore` (embedding/decorator, `examples/middleware/`),
   and the authz HTTP adapter (`examples/authzhttp/`) are **not** part of the
   running authorization path. They sit under `examples/` so `docs/22`–`24` and
   `docs/33` can demonstrate language features and the client/server seam against
   real authz types, while `internal/` stays purely about authorization.
   `internal/` never imports `examples/` — delete the folder and the system still
   runs. (Earlier these lived inside the `graph/` package; they were moved out so
   the engine reads clean.)

2. **Adapter helpers are duplicated across services.** The `json` helpers and
   HTTP body parsing appear in both services' HTTP adapters (and in both of Go's
   HTTP adapters). That looks un-DRY, but these are **independent services**:
   sharing infrastructure code across a service boundary would couple them. The
   only thing they deliberately share is the `shared`/SDK vocabulary. So the
   duplication is the correct call for a two-service design, not a lapse.

3. **TypeScript uses barrel files** (`core/index.ts`) and a few convenience
   re-exports. Idiomatic for TS module ergonomics; mildly debated, harmless here.

If this were a single production service rather than a teaching primer, you would
delete the `examples/` tree entirely (caveat 1) and likely keep (2) and (3) as-is.

---

## Keeping it clean — a checklist for new code

Before adding code, ask:

- Does anything in `core/` (TS) or `internal/{authz,documents}/*.go` (Go) import
  an adapter? If so, you've inverted the dependency — move the concrete behind a
  port.
- Does a new port belong to the core that **calls** it, not the one that
  implements it?
- Is the new port as **narrow** as the caller's actual need?
- Does the new wiring live only in the composition root?
- Does the constructor return the **interface**, not the concrete struct?

---

## Checkpoint

1. Why does `documents` define its own `AuthzClient` instead of importing
   `authz.Service` directly?
2. Name the one grep that proves the core/adapter dependency direction.
3. `Result[T]` lives in `examples/generics/`, separate from `internal/`. Why is
   that the right home for it? Defend your answer.

Good answers:
1. Interface segregation: the documents core needs only `Check` and
   `WriteTuples`, so it depends on a 2-method port. `authz.Service` satisfies it
   structurally, but the narrow port is what lets the in-process and HTTP authz
   clients be interchangeable.
2. In Go, `grep -rn 'internal/openfga\|internal/api' go/internal/` returns nothing
   (only `cmd/` imports the swappable backends); in TS, `grep -rn 'adapters/' …/core/`
   returns nothing — both prove the core
   never imports an adapter.
3. `Result[T]` is a generic utility unrelated to authorization. Keeping it in
   `examples/generics/` (not `internal/`) means the engine has no dependency on a
   teaching artifact: `internal/` imports nothing from `examples/`, so the lessons
   can be deleted wholesale without touching the ReBAC code. In a real production
   service you would simply drop the `examples/` tree; here it earns its keep only
   as the subject of `docs/23`.
