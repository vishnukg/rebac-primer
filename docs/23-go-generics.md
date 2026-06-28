# Go Generics

This chapter uses `examples/generics/result.go` to teach type parameters.
Generics are powerful, but Go keeps them deliberately plain. They are a tool for
removing real duplication while preserving static types, not a contest to see
how much punctuation can fit in a function signature.

## The Problem

Suppose the same container should hold an `int`, a document, or a slice without
copying the implementation for every type. A type parameter lets the caller
choose the contained type while the compiler still checks it.

`Result[T]` is a small value-or-error container:

```go
type Result[T any] struct {
    value T
    err   error
    ok    bool
}
```

Read `T any` as "`T` may be any type." These are different concrete types:

```go
generics.Result[int]
generics.Result[string]
```

They share one implementation, but a `Result[int]` cannot be passed where a
`Result[string]` is required.

## Type Parameters

Type parameters live in square brackets:

```go
func OK[T any](v T) Result[T] {
    return Result[T]{value: v, ok: true}
}
```

The caller can write the type argument explicitly:

```go
r := generics.OK[int](42)
```

Most of the time the compiler infers it:

```go
r := generics.OK(42) // T is int
```

Type inference is a convenience, not a dynamic escape hatch. The compiler still
knows the exact type selected for `T`.

## Constraints

A constraint says what operations are legal for a type parameter. The most open
constraint is `any`, which is an alias for `interface{}`:

```go
func Collect[T any](results []Result[T]) Result[[]T]
```

Inside `Collect`, Go knows nothing about `T` except that values of type `T` can
be stored, moved, returned, and appended to a `[]T`. You cannot compare two
values of type `T` unless the constraint says comparison is valid.

Use `comparable` when values must be valid map keys or support `==`:

```go
func Contains[T comparable](values []T, target T) bool {
    for _, v := range values {
        if v == target {
            return true
        }
    }
    return false
}
```

This works for strings, numbers, pointers, booleans, channels, and structs or
arrays made entirely of comparable fields. It does not work for slices, maps, or
functions.

## Type Sets

Constraints can describe a set of allowed underlying types:

```go
type Integer interface {
    ~int | ~int64 | ~uint
}

func Sum[T Integer](values []T) T {
    var total T
    for _, v := range values {
        total += v
    }
    return total
}
```

The `|` means "one of these types." The `~` means "this type or a named type
whose underlying type is this."

Without `~int`, a custom type like this would not satisfy the constraint:

```go
type Count int
```

Use type sets sparingly. They are useful for small numeric helpers and generic
collections. They are usually the wrong choice for domain behavior.

## Generic Functions

`Map` introduces both an input and output type:

```go
func Map[T, U any](r Result[T], f func(T) U) Result[U]
```

Read it as:

```text
If r contains a T, apply f and return a Result containing U.
If r contains an error, preserve the failure without calling f.
```

Example:

```go
r := generics.OK(3)
mapped := generics.Map(r, func(n int) string {
    return fmt.Sprintf("n=%d", n)
})
```

Here `T` is `int` and `U` is `string`.

## Generic Types And Methods

`Result[T]` is a generic type. Its methods can use the type's parameter:

```go
func (r Result[T]) Value() (T, bool) {
    return r.value, r.ok
}
```

Go methods cannot declare their own additional type parameters. That is why
`Map` is a top-level function rather than a method like this:

```go
// This is not valid Go.
func (r Result[T]) Map[U any](f func(T) U) Result[U]
```

The valid shape is:

```go
func Map[T, U any](r Result[T], f func(T) U) Result[U]
```

## Zero Values Still Matter

Generic code still follows Go's zero-value rules:

```go
var zero T
```

When `Result[T]` fails, `Value` returns the zero value of `T` and `false`:

```go
func (r Result[T]) Value() (T, bool) {
    return r.value, r.ok
}
```

That is the same comma-ok shape used by maps and type assertions:

```go
value, ok := m[key]
value, ok := maybeString.(string)
```

The `ok` boolean is the contract. The zero value alone cannot tell success from
failure.

## Generics Versus Interfaces

Use a type parameter when the operation is the same for many value types and the
compiler should preserve the exact type:

```go
func CloneSlice[T any](values []T) []T
```

Use an interface when you need behavior:

```go
type Checker interface {
    Evaluate(context.Context, rebac.CheckRequest) (rebac.CheckResult, error)
}
```

This distinction is the heart of practical Go generics:

```text
type parameter -> same algorithm over many value types
interface      -> any value with this behavior
```

## When Not To Use This Pattern

Idiomatic Go normally returns `(value, error)` directly:

```go
doc, err := service.Read(ctx, id, actor)
if err != nil {
    return err
}
```

`Result[T]` is a language lesson, not a recommendation to wrap every function
result. It is useful here because it gives us a small generic type, generic
functions, methods, zero-value behavior, and tests in one file.

A generic abstraction earns its place when it:

- removes meaningful duplication
- keeps static type information
- has a small, obvious constraint
- makes call sites clearer than the non-generic version

It does not earn its place merely because it is reusable in theory.

## Try It

Run:

```bash
go test -v ./examples/generics
```

Then make these changes one at a time:

1. Add a `Contains[T comparable]` helper and tests for `[]int` and `[]string`.
2. Try to call `Contains` with a slice of slices. Read the compiler error.
3. Add a `Map` from `int` to `bool`. Predict `T` and `U` before running tests.
4. Add a `Collect` test where the first item fails. Confirm later values are not
   inspected by adding a value that would panic if touched.
5. Try to write a method with an extra type parameter. Let the compiler reject
   it, then rewrite it as a function.

This code is not part of the authorization path. It lives under `examples/` so
the ReBAC implementation stays focused.

## Checkpoint

You are ready to continue when you can explain:

- what `T any` means
- why `Result[int]` and `Result[string]` are distinct types
- when `comparable` is needed
- what `~int` means in a constraint
- why `Map` is a function instead of a method
- when generics are worse than plain `(value, error)` Go

Next: [Go Interfaces and Embedding](24-go-interfaces-embedding.md).
