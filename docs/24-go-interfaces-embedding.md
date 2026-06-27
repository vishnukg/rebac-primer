# Go Interfaces and Embedding

This optional chapter uses `examples/middleware/middleware.go`.

## Interface Satisfaction

Go has implicit interface satisfaction. A type implements an interface when it
has the required methods.

```go
var _ Checker = (*AuditEvaluator)(nil)
```

That compile-time assertion fails if `AuditEvaluator` stops satisfying the
interface.

Interfaces are satisfied implicitly: no `implements` declaration is required.
Prefer small interfaces at the point of use. `AuditEvaluator` needs only the
ability to evaluate a check, so the middleware package declares its own
one-method `Checker` interface rather than depending on a concrete graph
evaluator or aliasing a provider interface.

## Decorator Pattern

`AuditEvaluator` wraps another evaluator:

```text
caller -> AuditEvaluator -> inner evaluator
```

It adds logging but returns the same result shape.

This shape is common Go middleware:

```text
accept interface -> wrap behavior -> return same interface
```

The wrapper can be inserted without changing the caller or the wrapped
implementation.

## Interface Embedding

`ReadOnlyStore` demonstrates interface embedding. Embedding promotes methods
from the embedded interface onto the wrapper type.

It embeds `authz.TupleReader`, not the full tuple repository. That means the
compiler exposes only read methods through `ReadOnlyStore`; `Write` and
`Delete` are not available on that value. This is the useful abstraction lesson:
choose the smallest interface that represents the capability the caller should
have.

## Try It

```bash
go test -v ./examples/middleware
```

Then try to call `Write` on `ReadOnlyStore`. It does not compile, because the
wrapper embeds only the reader capability. Change the embedded interface to
`authz.TupleRepository` and observe that write methods become available.

This is a language lesson only. The production ReBAC path is under
`internal/`.

## Checkpoint

What is the difference between wrapping a concrete type and wrapping a small
interface? The interface keeps the decorator reusable and prevents it from
depending on behavior it does not need.
