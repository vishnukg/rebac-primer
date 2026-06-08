# Go interfaces, embedding, and the decorator pattern

This chapter covers three ideas that reinforce each other: implicit interface
satisfaction, struct embedding, and the decorator pattern. All three appear in
`go/examples/middleware/middleware.go`. Read that file alongside this doc.

## Interfaces are satisfied implicitly

In TypeScript, you declare that a class implements an interface:

```ts
class AuditEvaluator implements Evaluator {
    evaluate(req: CheckRequest): Promise<CheckResult> { ... }
}
```

In Go, you never write `implements`. If a type has all the methods the interface
requires, it satisfies the interface — automatically, silently, and for free:

```go
// AuditEvaluator has an Evaluate method with the right signature.
// It satisfies the Checker interface without any declaration.
func (a *AuditEvaluator) Evaluate(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error) { ... }
```

This is implicit interface satisfaction, and it matters:

1. **Decoupling**: `AuditEvaluator` can live in a different package from
   `Checker`. It does not import the interface definition; the interface is
   satisfied by shape.
2. **Retroactive satisfaction**: you can make an existing type satisfy a new
   interface without touching the type.
3. **Discovery via compile-time assertion** (see below).

## The compile-time assertion

Because satisfaction is implicit, it is easy to drift. You rename a method and
silently stop satisfying an interface. The repo uses this idiom to catch it:

```go
// go/examples/middleware/middleware.go
var _ Checker = (*AuditEvaluator)(nil)
```

This line assigns a nil `*AuditEvaluator` to the blank identifier `_` typed as
`Checker`. If the method set does not match, the compiler rejects it here
rather than at the distant call site. Think of it as a standing unit test for
your interface contract.

The `(*T)(nil)` form creates a nil pointer of type `*T`. The assignment is never
executed at runtime — the variable is blank, so it vanishes. The compiler
evaluates the types without generating code.

## The decorator pattern

A decorator wraps a value, intercepts calls, and adds behaviour around them:

```
caller → AuditEvaluator.Evaluate → GraphEvaluator.Evaluate → result
                                ↑
                       writes audit log here
```

In Go, a decorator is a struct that holds the inner value and implements the same
interface:

```go
// go/examples/middleware/middleware.go
//
// Checker is a local alias for authz.Evaluator — the interface that both
// GraphEvaluator and AuditEvaluator satisfy.
type AuditEvaluator struct {
    inner  Checker
    logger *log.Logger
}

func (a *AuditEvaluator) Evaluate(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error) {
    start := time.Now()
    result, err := a.inner.Evaluate(ctx, req)
    // ... log outcome ...
    return result, err
}
```

The key: `AuditEvaluator.inner` is a `Checker` interface, not a concrete
type. You can wrap `GraphEvaluator`, an OpenFGA evaluator, or another
`AuditEvaluator`. The decorators compose without knowing each other's types.

In `cmd/server/main.go` (the composition root), wiring one in is one line:

```go
evaluator := middleware.NewAuditEvaluator(graph.NewGraphEvaluator(tupleStore), os.Stderr)
```

Nothing else changes. `documents.Service` depends on `authz.AuthzClient` and will
never know an audit layer was added. That separation — "what the service needs"
from "how it is implemented" — is the payoff of interface-based dependency injection.

Compare with TypeScript middleware:

```ts
class AuditEvaluator implements Evaluator {
    constructor(private inner: Evaluator, private log: Logger) {}
    async evaluate(req: CheckRequest): Promise<CheckResult> {
        const result = await this.inner.evaluate(req);
        this.log.info(...);
        return result;
    }
}
```

The Go and TypeScript patterns are structurally identical. Go just spells it with
a method and an interface instead of a class and `implements`.

## Interface composition

Interfaces can embed other interfaces. This is how you would model a tuple store
that separates read and write capabilities:

```go
// Conceptual split — illustrates interface composition
type TupleReader interface {
    Has(ctx context.Context, object rebac.Object, relation rebac.Relation, user rebac.Subject) (bool, error)
    FindByObjectRelation(ctx context.Context, object rebac.Object, relation rebac.Relation) ([]rebac.TupleKey, error)
}

type TupleWriter interface {
    Write(ctx context.Context, tuple rebac.TupleKey) error
    Delete(ctx context.Context, tuple rebac.TupleKey) error
}

// TupleRepository composes both — satisfying either interface is enough for code
// that only needs that half.
type TupleRepository interface {
    TupleReader  // embeds
    TupleWriter  // embeds
    FindAll(ctx context.Context, filter ...TupleFilter) ([]TupleKey, error)
}
```

In this repo `authz.TupleRepository` (`go/internal/authz/authz.go`) is the
single combined interface. Any code that only needs reads can accept a narrower
interface — splitting is a refactor you can make without changing any caller that
already passes a full `TupleRepository`.

## Struct embedding

Struct embedding is Go's form of composition-as-inheritance. Embed a type and
its method set is promoted onto the outer struct:

```go
// go/examples/middleware/middleware.go
type ReadOnlyStore struct {
    authz.TupleRepository  // embedded interface — all methods are promoted
}
```

Because `authz.TupleRepository` is embedded, `ReadOnlyStore` automatically has
`Has`, `FindByObjectRelation`, `FindAll`, `Write`, and `Delete` — promoted from
the embedded field. No delegation boilerplate required:

```go
ro := middleware.NewReadOnlyStore(store)
ro.Has(...)                  // promoted from TupleRepository
ro.FindByObjectRelation(...) // promoted from TupleRepository
```

The "read-only" name expresses intent — this value is meant to be passed only to
code that reads tuples. At the type level, all `TupleRepository` methods are
available; restricting by interface would require a separate read-only interface.
This is a common Go pattern: use naming and code review to communicate intent
when compiler enforcement is not worth the extra interface definition.

## Method promotion rules

When you embed type `T`:

1. All exported methods of `T` are promoted onto the outer struct.
2. If the outer struct defines a method with the same name, the outer method
   wins — it shadows the promoted one.
3. Promoted methods can satisfy interfaces. `ReadOnlyStore` satisfies
   `authz.TupleRepository` because the embedded `authz.TupleRepository` field
   promotes all the required methods.

```go
ev := graph.NewGraphEvaluator(ro)  // ro satisfies TupleRepository — compiles
```

`NewGraphEvaluator` accepts an `authz.TupleRepository`. `ReadOnlyStore` satisfies
that interface through embedding. No conversion, no casting.

## Interface values: the two-word secret

An interface value in Go is secretly two words: a pointer to the type descriptor,
and a pointer to the data. When you assign a concrete value to an interface:

```go
var e authz.Evaluator = graph.NewGraphEvaluator(store)
```

Go stores `(*GraphEvaluator)(pointer)` inside the interface. The compiler erases
the concrete type from the caller's perspective — callers only see `Evaluator`.

This is the same as TypeScript's structural typing: callers program to the
interface shape, not the concrete class.

## Layers of wrapping in practice

The composition root (`cmd/server/main.go`) is where you decide the order:

```go
tupleStore := authzdb.New(fixtures.SeedRelationshipTuples()...)
ro         := middleware.NewReadOnlyStore(tupleStore)   // signal: this path reads only
base       := graph.NewGraphEvaluator(ro)
audited    := middleware.NewAuditEvaluator(base, os.Stderr)
authzSvc   := authz.New(tupleStore, audited)
docsSvc    := documents.New(docRepo, authzSvc)
```

Each layer adds one concern. Reading the composition root tells you the full
stack without reading each layer's internals. This is the composition root
pattern: one place where all the wiring happens.

## Try it

**Add a caching decorator:**

Write `CachingEvaluator` that wraps `Checker` and caches the last result for
each `(user, relation, object)` triple using a `sync.Map`:

```go
type CachingEvaluator struct {
    inner Checker
    cache sync.Map
}
```

Add the compile-time assertion (`var _ Checker = (*CachingEvaluator)(nil)`).
Wire it between `AuditEvaluator` and `GraphEvaluator` in `buildHandler()`:

```go
base    := graph.NewGraphEvaluator(tupleStore)
cached  := graph.NewCachingEvaluator(base)
audited := middleware.NewAuditEvaluator(cached, os.Stderr)
```

Run `go test -race ./...` to confirm it is thread-safe.

## Checkpoint

Three short questions:

1. Go has no `implements` keyword. How does the compiler know `AuditEvaluator`
   satisfies `Checker`?
2. `ReadOnlyStore` has `Has` and `FindByObjectRelation` methods but you did not
   write them. Where do they come from?
3. What does `var _ Checker = (*AuditEvaluator)(nil)` actually do at runtime?

Good answers:
1. By checking that `AuditEvaluator` has all the methods `Checker` requires
   with the right signatures. The check is structural, not declarative.
2. They are promoted from the embedded `authz.TupleRepository` field.
3. Nothing. The blank identifier discards the value; the assignment generates no
   code. Only the compiler evaluates it, at compile time.
