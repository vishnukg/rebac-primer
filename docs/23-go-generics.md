# Go Generics

This optional chapter uses `examples/generics/result.go` to show Go type
parameters.

`Result[T]` is a small value-or-error container:

```go
type Result[T any] struct {
    value T
    err   error
    ok    bool
}
```

Key helpers:

```text
OK
Fail
Failf
Map
Collect
```

This code is not part of the authorization path. It lives under `examples/`
so it can be deleted without changing the ReBAC implementation.
