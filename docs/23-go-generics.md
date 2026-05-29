# Go generics: type parameters, constraints, and why they are worth it

Generics landed in Go 1.18. The syntax is deliberate and minimal. This chapter
grounds every concept in `go/internal/authz/adapters/graph/result.go`, which you can read and
run right now.

## The problem generics solve here

Imagine writing `AllPermissions` without generics. You want to return either a
`PermissionSummary` or an error. In Go, the canonical approach is a two-value
return:

```go
func AllPermissions(...) (PermissionSummary, error)
```

That works perfectly, and you should keep using it.

But sometimes you are building a pipeline: collect a bunch of results, transform
them, pass them downstream. Two-value returns do not compose well across higher-
order functions. A `Result[T]` type lets you chain `.Map()` and `Collect()`
without nesting `if err != nil` at every step.

`result.go` in this repo is that type. It is not meant to replace Go's idiomatic
error handling — it demonstrates generics in a real context.

## Declaring a generic type

```go
// go/internal/authz/adapters/graph/result.go
type Result[T any] struct {
    value T
    err   error
    ok    bool
}
```

`[T any]` is the type parameter list. `T` is the name; `any` is the constraint.
`any` is an alias for `interface{}` — it means "T can be any Go type."

Compare with TypeScript:

```ts
type Result<T> = { ok: true; value: T } | { ok: false; error: string }
```

Go uses a struct and a bool field instead of a union type, but the intent is the
same: one container for success or failure.

## Constraints

The constraint limits which types T can be. `any` is the broadest constraint.
You can narrow it:

```go
type Number interface {
    ~int | ~int64 | ~float64
}

func Sum[T Number](values []T) T { ... }
```

The `~` means "any type whose underlying type is int" — so a named type
`type Score int` satisfies `~int`.

`result.go` only needs `any` because it does not call any method on `T` and does
not compare T values. If you added a method that compared `T == T`, you would
need `comparable` instead of `any`.

## Generic constructors

```go
// go/internal/authz/adapters/graph/result.go
func OK[T any](v T) Result[T] {
    return Result[T]{value: v, ok: true}
}

func Fail[T any](err error) Result[T] {
    return Result[T]{err: err, ok: false}
}
```

These are generic functions. The type parameter `[T any]` appears before the
argument list, not after the function name. The compiler infers T from the
argument type at the call site:

```go
r := graph.OK(42)         // T inferred as int
r := graph.OK("hello")    // T inferred as string
r := graph.Fail[int](err) // T must be explicit — Fail has no T-typed argument
```

When the compiler cannot infer T (as in `Fail`), you must supply it explicitly.
This is the one place where Go generics feel slightly more verbose than
TypeScript.

## Generic functions (not methods)

```go
// go/internal/authz/adapters/graph/result.go
func Map[T, U any](r Result[T], f func(T) U) Result[U] {
    if !r.ok {
        return Fail[U](r.err)
    }
    return OK(f(r.value))
}
```

`Map` has two type parameters: `T` (the input type) and `U` (the output type).
Both are constrained by `any`. The compiler infers T from `r` and U from `f`'s
return type.

Go 1.18 does not support adding new type parameters in methods — only in top-
level functions. That is why `Map` and `Collect` are package-level functions
rather than methods on `Result`. This is a known limitation. In TypeScript you
can write `result.map(f)` fluently; in Go you write `graph.Map(r, f)`.

## `Collect` — the Promise.all of Go generics

```go
// go/internal/authz/adapters/graph/result.go
func Collect[T any](results []Result[T]) Result[[]T] {
    out := make([]T, 0, len(results))
    for _, r := range results {
        if !r.ok {
            return Fail[[]T](r.err)
        }
        out = append(out, r.value)
    }
    return OK(out)
}
```

`Collect` takes a slice of `Result[T]` and returns a single `Result[[]T]`. If
any element is a failure, it short-circuits and returns the first error. This is
exactly what `Promise.all` does in TypeScript — succeed only when all succeed.

The TypeScript equivalent:

```ts
function collect<T>(results: Result<T>[]): Result<T[]> {
    const values: T[] = [];
    for (const r of results) {
        if (!r.ok) return { ok: false, error: r.error };
        values.push(r.value);
    }
    return { ok: true, value: values };
}
```

The structure is identical. The syntax is Go.

## Type inference at the call site

Most of the time you do not write type arguments:

```go
mapped := graph.Map(graph.OK(3), func(n int) string {
    return fmt.Sprintf("item-%d", n)
})
// T inferred as int, U inferred as string
```

The compiler figures it out from the arguments. You only write type arguments
when inference fails:

```go
r := graph.Fail[int](errors.New("bad"))
```

`Fail` takes only an `error`, so there is nothing to infer `T` from.

## Generics versus `any` (the old way)

Before generics, Go programmers used `interface{}` (aliased as `any`) as a
universal container:

```go
// pre-generics
func Map(slice []interface{}, f func(interface{}) interface{}) []interface{}
```

This loses type information. The caller must type-assert every element back to
the concrete type, and mistakes panic at runtime. Generics move the check to
compile time.

```go
// with generics
func Map[T, U any](slice []T, f func(T) U) []U
```

No type assertions. No runtime panics from wrong types. The compiler rejects bad
calls before you run anything.

## When not to use generics

Generics add cognitive overhead. Use them when:

1. You are writing a container type (`Result[T]`, `Set[T]`, `Queue[T]`).
2. You have a utility function that works on any type (`Map`, `Filter`, `Reduce`).
3. You are eliminating a type assertion that could panic.

Do not use generics to avoid writing two concrete functions when the concrete
functions are clearer. `AllPermissions` does not use generics — it deals with
`Relation` and `bool` specifically, and the specificity makes it easier to read.

## Try it

**Add an `OrElse` function:**

```go
// OrElse returns the value from r if ok, or fallback if not.
func OrElse[T any](r Result[T], fallback T) T
```

Write the implementation in `result.go` and a test in `result_test.go`. The test
should cover both branches.

**Use `Map` and `Collect` together:**

In a scratch file or test function, build a slice of `Result[int]` from some
inputs, use `Map` to stringify each, then `Collect` them into a
`Result[[]string]`. Trace the types the compiler infers at each step.

## Checkpoint

Explain why `Fail[T]` requires an explicit type argument while `OK` and `Map`
do not.

Good answer: `Fail` takes only an `error` argument. The `T` in `Result[T]`
appears only in the return type, so the compiler has no argument type to infer
it from. `OK` takes a `T` value directly. `Map` takes a `Result[T]` and a
function `func(T) U`, so both T and U are inferrable.
