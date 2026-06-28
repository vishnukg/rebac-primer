# Go Interfaces and Embedding

This chapter uses `examples/middleware/middleware.go` to show three common Go
ideas:

- implicit interface satisfaction
- decorators that wrap behavior
- embedding as composition

Interfaces are where many experienced developers new to Go overbuild first. Go
interfaces are most useful when they are small, local, and boring.

## Interface Satisfaction

An interface is a method set:

```go
type Checker interface {
    Evaluate(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error)
}
```

A type implements an interface by having the required methods. There is no
`implements` declaration:

```go
func (a *AuditEvaluator) Evaluate(
    ctx context.Context,
    req rebac.CheckRequest,
) (rebac.CheckResult, error) {
    // ...
}
```

That method means `*AuditEvaluator` satisfies `Checker`.

A compile-time assertion documents that intent:

```go
var _ Checker = (*AuditEvaluator)(nil)
```

The right side is a typed nil pointer used only for type checking. It does not
allocate an evaluator.

## Small Interfaces At The Point Of Use

`AuditEvaluator` needs only one capability: evaluate a permission check. It
does not need to know whether the inner implementation is:

- the in-memory graph evaluator
- an OpenFGA adapter
- a fake used by a test
- another decorator

So the package declares the interface it consumes:

```go
type Checker interface {
    Evaluate(context.Context, rebac.CheckRequest) (rebac.CheckResult, error)
}
```

This is a common Go shape:

```text
consumer declares small interface -> provider returns concrete type -> wiring
connects them
```

Do not create a large provider-owned interface just because a mocking framework
in another language would ask for one.

## Method Sets And Pointer Receivers

Method sets matter when satisfying interfaces.

If a method has a pointer receiver:

```go
func (a *AuditEvaluator) Evaluate(...)
```

then `*AuditEvaluator` has that method. A plain `AuditEvaluator` value does not
necessarily satisfy the interface in every assignment context.

This is why constructors commonly return pointers:

```go
func NewAuditEvaluator(inner Checker, w io.Writer) *AuditEvaluator
```

The returned value has the full pointer method set, and callers do not copy the
wrapper by accident.

## Nil Interfaces

An interface value contains a dynamic type and a dynamic value. It is nil only
when both are absent:

```go
var c Checker // nil interface
```

This is different:

```go
var a *AuditEvaluator = nil
var c Checker = a
fmt.Println(c == nil) // false
```

The interface has a dynamic type (`*AuditEvaluator`) even though the dynamic
value is nil. This is the same trap discussed for `error` in the foundations
chapters. Avoid returning typed nil pointers as interface values.

## Decorator Pattern

`AuditEvaluator` wraps another evaluator:

```text
caller -> AuditEvaluator -> inner evaluator
```

It adds logging but preserves the same external behavior:

```go
func (a *AuditEvaluator) Evaluate(ctx context.Context, req rebac.CheckRequest) (
    rebac.CheckResult,
    error,
) {
    start := time.Now()
    result, err := a.inner.Evaluate(ctx, req)
    elapsed := time.Since(start)

    a.logger.Printf("check user=%s relation=%s object=%s ...", ...)
    return result, err
}
```

This is the classic Go middleware shape:

```text
accept interface -> wrap behavior -> return a value satisfying same interface
```

It works because callers depend on behavior, not concrete implementation.

## Embedding Is Composition

Struct embedding promotes fields and methods from an embedded value:

```go
type ReadOnlyStore struct {
    authz.TupleReader
}
```

`ReadOnlyStore` now exposes the methods of `TupleReader`. It does not expose
write methods because it embeds only the read interface, not the full repository
interface.

That makes a capability boundary visible to the compiler:

```go
ro := middleware.NewReadOnlyStore(store)
ro.Has(ctx, object, relation, user) // ok
ro.Write(ctx, tuples)               // does not compile
```

Embedding is not inheritance. The outer type can add methods, override promoted
methods by declaring its own method with the same name, or embed a different
capability entirely.

## Interface Embedding

Interfaces can embed other interfaces:

```go
type TupleRepository interface {
    TupleReader
    TupleWriter
}
```

This means `TupleRepository` includes every method from both embedded
interfaces. Use this to build larger capabilities from small ones.

Be careful when embedding broad interfaces. Embedding `io.ReadWriter` is often
fine. Embedding a giant application interface can smuggle methods through a
boundary you meant to keep narrow.

## Type Assertions And Type Switches

Most code should call interface methods directly. Sometimes a boundary receives
a broad value and must inspect the dynamic type.

Type assertion:

```go
writer, ok := value.(io.Writer)
if !ok {
    return errors.New("value cannot write")
}
```

Type switch:

```go
switch v := value.(type) {
case nil:
    return "missing"
case string:
    return v
case fmt.Stringer:
    return v.String()
default:
    return fmt.Sprintf("%v", v)
}
```

Use these at boundaries, adapters, and generic plumbing. If domain code is full
of type switches, you may be missing a real interface.

## Interfaces Versus Generics

Interfaces describe behavior:

```go
type Checker interface {
    Evaluate(context.Context, rebac.CheckRequest) (rebac.CheckResult, error)
}
```

Generics describe type-checked reuse over values:

```go
func Collect[T any](results []Result[T]) Result[[]T]
```

If you need to call a method, an interface is probably the right shape. If you
need the same algorithm over `[]int`, `[]string`, and `[]Document`, a type
parameter may be better.

## What To Avoid

Avoid these patterns unless you have a concrete reason:

- interfaces with one implementation and no package boundary benefit
- interfaces named after their implementation, such as `GraphEvaluatorInterface`
- broad interfaces returned from constructors
- embedding just to save a few keystrokes
- type assertions used to recover behavior a narrow interface hid
- test fakes that assert every internal call instead of visible behavior

The goal is not fewer lines. The goal is code where capabilities are obvious.

## Try It

Run:

```bash
go test -v ./examples/middleware
```

Then try these experiments:

1. Call `Write` on `ReadOnlyStore`. It should not compile.
2. Change `ReadOnlyStore` to embed `authz.TupleRepository`. `Write` becomes
   available. Decide whether that is a better contract.
3. Add a second decorator that counts checks instead of logging them.
4. Wrap an `AuditEvaluator` around another `AuditEvaluator`. It works because
   both satisfy `Checker`.
5. Create a fake `Checker` in a test that always denies. Confirm the decorator
   preserves the denial.

## Checkpoint

You are ready to continue when you can explain:

- why Go does not need an `implements` keyword
- why the consumer often owns the interface
- how pointer receivers affect interface satisfaction
- why a typed nil inside an interface is not a nil interface
- why `ReadOnlyStore` exposes read methods but not write methods
- when type assertions are a useful boundary tool

Next: [Go Testing](25-go-testing.md).
