// Package generics is a Go-language teaching example, NOT part of the production
// ReBAC path. It demonstrates generic type parameters via a value-or-error
// container. See docs/23-go-generics.md.
package generics

import "fmt"

// Result[T] is a generic value-or-error container.
//
// Go generics use type parameters declared in square brackets: [T any] means
// "T can be any type." The constraint 'any' is an alias for interface{}.
//
// Compare with TypeScript's generic syntax:
//
//	type Result<T> = { ok: true; value: T } | { ok: false; error: string }
//
// Go achieves the same idea with a struct and a bool field.
type Result[T any] struct {
	value T
	err   error
	ok    bool
}

// OK wraps a successful value in a Result.
func OK[T any](v T) Result[T] { return Result[T]{value: v, ok: true} }

// Fail wraps an error in a Result.
func Fail[T any](err error) Result[T] { return Result[T]{err: err, ok: false} }

// Failf creates a Fail Result from a format string, matching the fmt.Errorf API.
func Failf[T any](format string, args ...any) Result[T] {
	return Fail[T](fmt.Errorf(format, args...))
}

// IsOK reports whether the Result holds a value.
func (r Result[T]) IsOK() bool { return r.ok }

// Value returns the contained value and true, or the zero value and false.
func (r Result[T]) Value() (T, bool) { return r.value, r.ok }

// Unwrap returns the value or panics if the Result is a failure.
// Use only where a failure is truly a programming error, not a recoverable state.
func (r Result[T]) Unwrap() T {
	if !r.ok {
		panic(fmt.Sprintf("graph.Result.Unwrap on a failure: %v", r.err))
	}
	return r.value
}

// Err returns the contained error, or nil for a success.
func (r Result[T]) Err() error {
	if r.ok {
		return nil
	}
	return r.err
}

// Map applies f to the value inside r and returns a new Result.
// If r is a failure, Map returns that failure unchanged without calling f.
//
// Map is a top-level function (not a method) because Go does not yet support
// adding new type parameters in methods — only in functions.
func Map[T, U any](r Result[T], f func(T) U) Result[U] {
	if !r.ok {
		return Fail[U](r.err)
	}
	return OK(f(r.value))
}

// Collect gathers a slice of Results into a single Result containing a slice.
// The first failure short-circuits and is returned as the overall failure.
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
