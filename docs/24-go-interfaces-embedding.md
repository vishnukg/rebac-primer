# Go interfaces, embedding, and the decorator pattern

This chapter covers three ideas that reinforce each other: implicit interface
satisfaction, struct embedding, and the decorator pattern. All three appear in
`go/internal/authzservice/adapters/graph/middleware.go`. Read that file alongside this doc.

## Interfaces are satisfied implicitly

In TypeScript, you declare that a class implements an interface:

```ts
class AuditAuthorizer implements Authorizer {
    check(req: CheckRequest): Promise<CheckResult> { ... }
}
```

In Go, you never write `implements`. If a type has all the methods the interface
requires, it satisfies the interface — automatically, silently, and for free:

```go
// AuditAuthorizer has a Check method with the right signature.
// It satisfies Authorizer without any declaration.
func (a *AuditAuthorizer) Check(ctx context.Context, req CheckRequest) (CheckResult, error) { ... }
```

This is implicit interface satisfaction, and it matters:

1. **Decoupling**: `AuditAuthorizer` can live in a different package from
   `Authorizer`. It does not import the interface definition; the interface is
   satisfied by shape.
2. **Retroactive satisfaction**: you can make an existing type satisfy a new
   interface without touching the type.
3. **Discovery via compile-time assertion** (see below).

## The compile-time assertion

Because satisfaction is implicit, it is easy to drift. You rename a method and
silently stop satisfying an interface. The repo uses this idiom to catch it:

```go
// go/internal/authzservice/adapters/graph/middleware.go
var _ Authorizer = (*AuditAuthorizer)(nil)
```

This line assigns a nil `*AuditAuthorizer` to the blank identifier `_` typed as
`Authorizer`. If the method set does not match, the compiler rejects it here
rather than at the distant call site. Think of it as a standing unit test for
your interface contract.

The `(*T)(nil)` form creates a nil pointer of type `*T`. The assignment is never
executed at runtime — the variable is blank, so it vanishes. The compiler
evaluates the types without generating code.

## The decorator pattern

A decorator wraps a value, intercepts calls, and adds behaviour around them:

```
caller → AuditAuthorizer.Check → GraphAuthorizer.Check → result
                              ↑
                     writes audit log here
```

In Go, a decorator is a struct that holds the inner value and implements the same
interface:

```go
// go/internal/authzservice/adapters/graph/middleware.go
type AuditAuthorizer struct {
    inner  Authorizer
    logger *log.Logger
}

func (a *AuditAuthorizer) Check(ctx context.Context, req CheckRequest) (CheckResult, error) {
    start := time.Now()
    result, err := a.inner.Check(ctx, req)
    // ... log outcome ...
    return result, err
}
```

The key: `AuditAuthorizer.inner` is an `Authorizer` interface, not a concrete
type. You can wrap `GraphAuthorizer`, `OpenFGAAuthorizer`, or another
`AuditAuthorizer`. The decorators compose without knowing each other's types.

In `app.go` (the composition root), wiring one in is one line:

```go
authorizer := authz.NewAuditAuthorizer(authz.NewGraphAuthorizer(tupleStore), os.Stderr)
```

Nothing else changes. `DocumentService` depends on `Authorizer` and will never
know an audit layer was added. That separation — "what the service needs" from
"how it is implemented" — is the payoff of interface-based dependency injection.

Compare with TypeScript middleware:

```ts
class AuditAuthorizer implements Authorizer {
    constructor(private inner: Authorizer, private log: Logger) {}
    async check(req: CheckRequest): Promise<CheckResult> {
        const result = await this.inner.check(req);
        this.log.info(...);
        return result;
    }
}
```

The Go and TypeScript patterns are structurally identical. Go just spells it with
a method and an interface instead of a class and `implements`.

## Interface composition

Interfaces can embed other interfaces. `store.go` uses this:

```go
// go/internal/authzservice/adapters/db/store.go
type TupleStore interface {
    TupleReader   // embeds
    TupleWriter   // embeds
    All() []TupleKey
}
```

A `TupleStore` is anything that satisfies `TupleReader`, `TupleWriter`, and
`All()`. `*InMemoryTupleStore` satisfies all three. This composes interfaces
without repeating method signatures.

## Struct embedding

Struct embedding is Go's form of composition-as-inheritance. Embed a type and
its method set is promoted onto the outer struct:

```go
// go/internal/authzservice/adapters/graph/middleware.go
type ReadOnlyStore struct {
    TupleReader  // embedded interface
}
```

Because `TupleReader` is embedded, `ReadOnlyStore` automatically has `Has` and
`FindByObjectRelation` methods — promoted from the embedded field. No delegation
boilerplate required:

```go
ro := authz.NewReadOnlyStore(store)
ro.Has(...)                  // promoted from TupleReader
ro.FindByObjectRelation(...) // promoted from TupleReader
```

`ReadOnlyStore` does not embed `TupleWriter`, so `Write` and `Delete` are
simply absent from its method set. The compiler enforces the restriction — you
cannot call `ro.Write(...)` because the method does not exist.

This is a structural alternative to visibility modifiers. TypeScript uses
`readonly` and `private`. Go restricts by controlling which interface is
embedded.

## Method promotion rules

When you embed type `T`:

1. All exported methods of `T` are promoted onto the outer struct.
2. If the outer struct defines a method with the same name, the outer method
   wins — it shadows the promoted one.
3. Promoted methods can satisfy interfaces. `ReadOnlyStore` satisfies
   `TupleReader` because the embedded `TupleReader` field promotes the required
   methods.

```go
auth := authz.NewGraphAuthorizer(ro)  // ro satisfies TupleReader — compiles
```

`NewGraphAuthorizer` accepts a `TupleReader`. `ReadOnlyStore` satisfies
`TupleReader` through embedding. No conversion, no casting.

## Interface values: the two-word secret

An interface value in Go is secretly two words: a pointer to the type descriptor,
and a pointer to the data. When you assign a concrete value to an interface:

```go
var a authz.Authorizer = authz.NewGraphAuthorizer(store)
```

Go stores `(*GraphAuthorizer)(pointer)` inside the interface. The compiler erases
the concrete type from the caller's perspective — callers only see `Authorizer`.

This is the same as TypeScript's structural typing: callers program to the
interface shape, not the concrete class.

## Layers of wrapping in practice

The composition root (`app.go`) is where you decide the order:

```go
tupleStore := authz.NewInMemoryTupleStore(...)
ro         := authz.NewReadOnlyStore(tupleStore)   // restrict writes
base       := authz.NewGraphAuthorizer(ro)
audited    := authz.NewAuditAuthorizer(base, os.Stderr)
docs       := domain.NewDocumentService(audited)
```

Each layer adds one concern. Reading the composition root tells you the full
stack without reading each layer's internals. This is the composition root
pattern: one place where all the wiring happens.

## Try it

**Add a caching decorator:**

Write `CachingAuthorizer` that wraps `Authorizer` and caches the last result for
each `(user, relation, object)` triple using a `sync.Map`:

```go
type CachingAuthorizer struct {
    inner authz.Authorizer
    cache sync.Map
}
```

Add the compile-time assertion (`var _ Authorizer = (*CachingAuthorizer)(nil)`).
Wire it between `AuditAuthorizer` and `GraphAuthorizer` in the composition root.

Run `go test -race ./internal/...` to confirm it is thread-safe.

## Checkpoint

Three short questions:

1. Go has no `implements` keyword. How does the compiler know `AuditAuthorizer`
   satisfies `Authorizer`?
2. `ReadOnlyStore` has `Has` and `FindByObjectRelation` methods but you did not
   write them. Where do they come from?
3. What does `var _ Authorizer = (*AuditAuthorizer)(nil)` actually do at runtime?

Good answers:
1. By checking that `AuditAuthorizer` has all the methods `Authorizer` requires
   with the right signatures. The check is structural, not declarative.
2. They are promoted from the embedded `TupleReader` field.
3. Nothing. The blank identifier discards the value; the assignment generates no
   code. Only the compiler evaluates it, at compile time.
