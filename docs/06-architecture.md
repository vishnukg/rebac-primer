# Architecture: Ports and Adapters

This repo uses ports and adapters. The point is simple: domain code names the
capabilities it needs, and concrete infrastructure is wired at the edge.

## Shape

```text
HTTP handler -> documents.Service -> AuthzClient port -> authz.Service
                                      Repository port -> document store
                                      Authenticator port -> token verifier

authz.Service -> Evaluator port -> graph evaluator
              -> TupleRepository port -> tuple store

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
| `internal/authz` | authorization service, evaluator interface, tuple repository interface |
| `internal/documents` | document use cases and the ports they need |
| `internal/api` | HTTP adapter for the documents service |
| `internal/openfga` | OpenFGA adapter implementing `authz.Service` |
| `internal/fixtures` | demo users, tuples, and tokens |
| `cmd/server` | composition root |

## Narrow Ports

`documents` does not need every authz operation. It needs only:

```go
type AuthzClient interface {
    Check(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error)
    WriteTuples(ctx context.Context, tuples []rebac.TupleKey) error
}
```

The full `authz.Service` satisfies that interface, but the document domain only
depends on the two methods it actually uses.

## Backend Swap

The default backend is in-process:

```text
documents -> authz.Service -> GraphEvaluator -> InMemoryStore
```

The OpenFGA backend is selected at startup:

```text
documents -> openfga.Service -> OpenFGA server
```

Both implement the same app-facing authz service shape, so the documents domain
and HTTP handler do not change.

## Cleanliness Check

Only `cmd/server` should import swappable backend packages such as
`internal/openfga` and `internal/api`:

```bash
grep -rn 'internal/openfga\|internal/api' internal/
```

That should print nothing. If domain code imports a concrete adapter directly,
move that dependency behind a port.
