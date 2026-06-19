# Go Generics

This optional chapter uses `examples/generics/result.go` to show Go type
parameters.

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

`T any` means "`T` may be any type." These are different concrete types:

```go
generics.Result[int]
generics.Result[string]
```

They share one implementation, but an `int` result cannot accidentally be used
as a string result.

## Functions With Type Parameters

`Map` introduces both an input and output type:

```go
func Map[T, U any](r Result[T], f func(T) U) Result[U]
```

Read it as: if `r` contains a `T`, apply `f` and return a result containing `U`.
If `r` contains an error, preserve the failure without calling `f`.

Go methods cannot declare additional type parameters, which is why `Map` is a
function rather than a method on `Result[T]`.

## When Not To Use This Pattern

Idiomatic Go normally returns `(value, error)` directly. `Result[T]` is a
language lesson, not a recommendation to wrap every function result. A generic
abstraction earns its place when it removes meaningful duplication or captures
a reusable operation—not merely because generics are available.

## Try It

```bash
go test -v ./examples/generics
```

Then add a `Map` from `int` to `bool`. Predict the inferred types before running
the compiler.

This code is not part of the authorization path. It lives under `examples/`
so it can be deleted without changing the ReBAC implementation.

## Checkpoint

What does the compiler know about `Result[T]` that it would not know about a
container using `any` fields? It knows the exact value type selected by the
caller and checks every operation involving that type.
