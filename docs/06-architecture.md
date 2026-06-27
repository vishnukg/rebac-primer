# Architecture: Ports and Adapters

This repo uses ports and adapters. The point is simple: domain code names the
capabilities it needs, and concrete infrastructure is wired at the edge.

## Shape

```text
HTTP handler -> api.DocumentService port -> documents.Service
                api.Authenticator port   -> token verifier

documents.Service -> documents.AuthorizationService port -> authz.Service
                                                       or -> openfga.Service
                  -> documents.DocumentRepository port   -> document store

authz.Service -> authz.Evaluator port              -> graph evaluator
              -> authz.TupleWriter/TupleLister     -> tuple store

GraphEvaluator -> authz.TupleReader port           -> tuple store

cmd/server/main.go wires the concrete implementations.
```

The important dependency rule:

```text
domain code owns interfaces
adapters implement interfaces
cmd/server/main.go chooses adapters
```

## Packages

| Package | Role |
|---|---|
| `internal/rebac` | shared ReBAC vocabulary: objects, relations, tuples, checks |
| `internal/authz` | concrete authorization service plus the evaluator and tuple repository interfaces it consumes |
| `internal/documents` | document use cases and the ports they need |
| `internal/api` | HTTP adapter and the narrow interfaces it consumes |
| `internal/openfga` | concrete OpenFGA authorization adapter |
| `internal/fixtures` | demo users, tuples, and tokens |
| `cmd/server` | composition root |

## Narrow Ports

`documents` does not need every authz operation. It needs only:

```go
type AuthorizationService interface {
    Check(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error)
    WriteTuples(ctx context.Context, tuples []rebac.TupleKey) error
    DeleteTuples(ctx context.Context, tuples []rebac.TupleKey) error
}
```

Both `*authz.Service` and `*openfga.Service` satisfy that interface implicitly,
but the document domain depends only on the three methods it actually uses.
Delete is used only for
compensating cleanup if document creation cannot write its authorization tuples.

The HTTP adapter follows the same rule. `internal/api` declares
`DocumentService` and `Authenticator`; the documents package exports concrete
implementations and does not define interfaces on behalf of its consumers.

## Backend Swap

The default backend is in-process:

```text
documents -> authz.Service -> GraphEvaluator -> InMemoryStore
```

The OpenFGA backend is selected at startup:

```text
documents -> openfga.Service -> OpenFGA server
```

Both satisfy `documents.AuthorizationService`, so the documents domain and HTTP
handler do not change.

## Cleanliness Check

Production packages outside an adapter and the composition root should not
import concrete adapters. Excluding tests, this command should show OpenFGA
being selected only in `cmd/server`:

```bash
rg '"rebac-primer/internal/openfga"' --glob '*.go' --glob '!**/*_test.go'
```

If domain code imports `internal/openfga` directly, move that dependency behind
a port.

## Checkpoint

Why does `documents` own `AuthorizationService` instead of importing an OpenFGA client?
Because the document use case should describe the capability it needs, while
the composition root chooses how that capability is implemented.
