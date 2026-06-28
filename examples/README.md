# `examples/` — Go-language lessons, NOT the ReBAC engine

Everything in this folder exists to teach a **Go language feature**. None of it is
imported by the running server (`cmd/server`). You can delete this whole directory
and the ReBAC system still works.

Keep this split in your head:

| You want to learn… | Read… |
|--------------------|-------|
| How ReBAC actually works | `internal/` (start at `internal/authz/evaluator.go`) |
| A Go language feature | `examples/` (this folder) |

## What's here

| Package | Teaches | Paired doc |
|---------|---------|------------|
| `generics/` | Generic type parameters (`Result[T]`, `Map`, `Collect`) | `docs/23-go-generics.md` |
| `concurrency/` | Goroutines, channels, `WaitGroup.Go` (`AllPermissions`, `BulkCheck`) | `docs/22-go-concurrency.md` |
| `middleware/` | Decorator pattern + interface embedding (`AuditEvaluator`, `ReadOnlyStore`) | `docs/24-go-interfaces-embedding.md` |
| `authzhttp/` | Exposing the authz service over HTTP (the client/server seam) | `docs/33-client-server-rebac.md` |

These packages import the *real* code in `internal/` (e.g. the graph evaluator)
so the examples operate on the same domain you're learning — but the arrow only
points one way. `internal/` never imports `examples/`. That is the whole point of
the separation: the production engine has no idea these lessons exist.

## Why separate them at all?

When you open `internal/` you should see *only* authorization logic — nothing
competing for your attention. When you want a Go lesson, you come here and it's
clearly labelled as a lesson, not as something you must understand to ship ReBAC.

Run every example package with:

```bash
go test ./examples/...
```

The examples are executable lessons: read the paired doc, predict a test result,
then change one small thing and rerun it. Pair them with
`docs/25-go-testing.md` when you want to practice benchmarks, fuzzing, race
tests, and test helpers against small packages before touching the main service.
