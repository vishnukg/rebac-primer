# Go Interfaces and Embedding

This optional chapter uses `examples/middleware/middleware.go`.

## Interface Satisfaction

Go has implicit interface satisfaction. A type implements an interface when it
has the required methods.

```go
var _ authz.Evaluator = (*AuditEvaluator)(nil)
```

That compile-time assertion fails if `AuditEvaluator` stops satisfying the
interface.

## Decorator Pattern

`AuditEvaluator` wraps another evaluator:

```text
caller -> AuditEvaluator -> inner evaluator
```

It adds logging but returns the same result shape.

## Interface Embedding

`ReadOnlyStore` demonstrates interface embedding. Embedding promotes methods
from the embedded interface onto the wrapper type.

This is a language lesson only. The production ReBAC path is under
`internal/`.
