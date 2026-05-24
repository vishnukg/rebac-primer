# Go concurrency: goroutines, channels, and WaitGroups

Go's concurrency model is famously simple to write and famously easy to get wrong.
This chapter grounds every concept in `go/internal/authzservice/adapters/graph/parallel.go`, which you
can run right now.

## The problem concurrency solves here

When a UI renders a permissions panel it needs to know all four computed
permissions for a document at once: can_read, can_comment, can_edit, can_delete.

Running four checks sequentially takes four times as long. Running them
concurrently takes as long as the slowest one. For a UI that renders eagerly,
the difference is visible.

`AllPermissions` and `BulkCheck` in `parallel.go` solve this with two different
Go concurrency primitives. Read both, because each one teaches something the
other does not.

## Goroutines are not threads

A goroutine is a function running concurrently with the caller. Start one with
`go`:

```go
go func() {
    fmt.Println("I run concurrently")
}()
```

The Go runtime schedules goroutines across OS threads, so they are cheap: you
can start thousands without worrying about the OS thread limit. But "cheap" does
not mean "free" — goroutines you start must finish.

## Channels: communication as coordination

A channel is a typed pipe. One goroutine sends; another receives. The send
blocks until a receiver is ready, and the receive blocks until a sender is ready.
That blocking is not a bug — it is how goroutines synchronise.

```go
ch := make(chan int)     // unbuffered: send blocks until received
ch := make(chan int, 5)  // buffered: first 5 sends do not block
```

`AllPermissions` uses a **buffered channel** whose capacity equals the number of
relations (`parallel.go:32`):

```go
// go/internal/authzservice/adapters/graph/parallel.go
ch := make(chan outcome, len(relations))
```

Each goroutine sends exactly one value into the channel and then exits. Because
the channel is buffered to match the number of goroutines, no goroutine ever
blocks waiting for the collector — the sends always succeed immediately. The
collector runs the loop after all goroutines have been started:

```go
for _, rel := range relations {
    go func(rel Relation) {
        result, err := auth.Check(ctx, CheckRequest{User: user, Relation: rel, Object: object})
        ch <- outcome{relation: rel, allowed: result.Allowed, err: err}
    }(rel)
}

summary := make(PermissionSummary, len(relations))
for range len(relations) {
    out := <-ch
    // ...
}
```

The outer loop starts N goroutines; the inner loop collects N results. When the
inner loop finishes, every goroutine has sent its result and exited. No goroutine
leaks.

## Why the goroutine captures `rel` as an argument

Look at this pattern:

```go
for _, rel := range relations {
    go func(rel Relation) {  // rel is a parameter, not a closed-over variable
        // use rel
    }(rel)
}
```

If `rel` were closed over directly (not passed as an argument), all goroutines
would share the same variable — and by the time they run, the loop may have
already advanced it to the last value. Passing it as an argument gives each
goroutine its own copy, frozen at the moment the goroutine was started.

In Go 1.22+ this is less of a trap because loop variables are now per-iteration,
but the explicit argument form is still the clearer idiom — it documents intent.

## WaitGroups: wait without collecting

`BulkCheck` uses `sync.WaitGroup` instead of a channel (`parallel.go:60`):

```go
// go/internal/authzservice/adapters/graph/parallel.go
func BulkCheck(ctx context.Context, auth Authorizer, reqs []CheckRequest) []BulkResult {
    results := make([]BulkResult, len(reqs))
    var wg sync.WaitGroup

    for i, req := range reqs {
        wg.Add(1)
        go func(i int, req CheckRequest) {
            defer wg.Done()
            result, err := auth.Check(ctx, req)
            results[i] = BulkResult{Request: req, Result: result, Err: err}
        }(i, req)
    }

    wg.Wait()
    return results
}
```

`WaitGroup` is a counter:

- `wg.Add(1)` increments it before each goroutine starts.
- `defer wg.Done()` decrements it when the goroutine returns.
- `wg.Wait()` blocks until the counter reaches zero.

Because each goroutine writes to a unique index (`results[i]`), no two goroutines
touch the same memory location — so no mutex is needed. This is safe.

## Channel vs WaitGroup: when to use which

| | Channel | WaitGroup |
|---|---|---|
| **Collects values** | Yes — receive from channel | No — goroutines write shared state |
| **Order preserved** | No — first finished, first received | Yes — write by index |
| **Signals first error** | Easy — close channel or sentinel value | Awkward — needs an extra channel |
| **Use when** | results are all the same shape and order does not matter | you need results in input order |

`AllPermissions` uses channels because results come back unordered and we build
a map.

`BulkCheck` uses WaitGroup because callers need results in the same position as
their input requests.

## Context cancellation: letting callers abort

Both functions accept a `context.Context` as their first parameter. Context
carries a cancellation signal — when the caller cancels, goroutines that check
`ctx.Done()` can stop early.

`GraphAuthorizer.Check` currently ignores the context (it is an in-memory
traversal, so cancellation is not worth the complexity). A real network call
would pass `ctx` to the HTTP client and abort automatically. The interface
requires the parameter so swapping in the real OpenFGA client later needs no
changes at the call site.

```go
// context.Background() in tests means "never cancel"
result, err := auth.Check(context.Background(), req)
```

## `select`: waiting on multiple channels

For completeness — `select` is Go's channel-aware switch. It blocks until one
of its cases can proceed:

```go
select {
case result := <-resultCh:
    // process result
case <-ctx.Done():
    return nil, ctx.Err()  // caller cancelled
}
```

`AllPermissions` does not need `select` because it is collecting a fixed number
of results and uses a buffered channel (no blocking). You would add `select` to
handle early cancellation in a production service where the context might time
out mid-flight.

## Race detector

Go ships a built-in race detector. Run your tests with `-race` to catch unsafe
concurrent access:

```bash
go test -race ./internal/authz/...
```

A race condition occurs when two goroutines access the same memory location and
at least one is a write. The race detector instruments memory access at runtime
and reports violations immediately. Run it on every PR.

## Try it

**Break the goroutine capture pattern:**

In `AllPermissions`, change the goroutine from:

```go
go func(rel Relation) {
    // uses the parameter rel
}(rel)
```

to:

```go
go func() {
    // closes over loop variable rel
}()
```

Run `go test -race ./internal/authz/...`. The race detector will flag the
closure reading the loop variable concurrently.

**Observe ordering:**

Add a `t.Logf` inside each goroutine in `BulkCheck` and run with `-v`. The
goroutines finish in a different order each run, but `results` always comes back
sorted by input index.

## Checkpoint

Two questions. Both require reading `parallel.go`.

1. Why is the channel in `AllPermissions` buffered? What would happen if it were
   unbuffered?
2. `BulkCheck` writes `results[i]` from a goroutine. Why is that safe without a
   mutex?

If you can answer both without guessing, you understand the chapter.
