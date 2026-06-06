# Go concurrency: goroutines, channels, and WaitGroups

Go's concurrency model is famously simple to write and famously easy to get wrong.
This chapter grounds every concept in `go/internal/authz/adapters/graph/parallel.go`, which you
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

A mental model: a channel is a conveyor belt between workers.

```text
goroutine A  ──ch<- value──►  [ channel ]  ──<-ch──►  goroutine B
   (sender)                  (the belt)              (receiver)
```

- **Send** (`ch <- v`) puts one value on the belt.
- **Receive** (`v := <-ch`) takes one value off the belt.
- An **unbuffered** belt has no space to set things down: the sender must wait
  until the receiver picks the value straight out of their hand (a handoff). This
  is why an unbuffered channel synchronises two goroutines in time — neither
  proceeds until both are ready.
- A **buffered** belt has N slots: the sender can drop up to N values and keep
  working without waiting for anyone. Only when the buffer is full does the next
  send block.

```go
ch := make(chan int)     // unbuffered: send blocks until received (handoff)
ch := make(chan int, 5)  // buffered: first 5 sends do not block
```

Two more facts you will rely on:

- Receiving from a channel that has no value yet **blocks** the receiver until
  one arrives. That is how the collector below waits for results without a busy
  loop.
- A channel can be **closed** (`close(ch)`) to signal "no more values"; ranging
  over a channel (`for v := range ch`) stops when it is closed. We do not need
  close here because we collect an exact, known number of results.

`AllPermissions` uses a **buffered channel** whose capacity equals the number of
relations (`parallel.go:32`):

```go
// go/internal/authz/adapters/graph/parallel.go
ch := make(chan outcome, len(relations))
```

Each goroutine sends exactly one value into the channel and then exits. Because
the channel is buffered to match the number of goroutines, no goroutine ever
blocks waiting for the collector — the sends always succeed immediately. The
collector runs the loop after all goroutines have been started:

```go
for _, rel := range relations {
    go func(rel shared.Relation) {
        result, err := auth.Evaluate(ctx, shared.CheckRequest{User: user, Relation: rel, Object: object})
        ch <- outcome{relation: rel, allowed: result.Allowed, err: err}
    }(rel)
}

summary := make(PermissionSummary, len(relations))
for range len(relations) {
    select {
    case out := <-ch:
        // a result arrived
        summary[out.relation] = out.allowed
    case <-ctx.Done():
        // the caller cancelled or timed out
        return nil, ctx.Err()
    }
}
```

The outer loop starts N goroutines; the inner loop collects N results. Each
iteration of the inner loop blocks in the `select` until **either** the next
result arrives **or** the context is cancelled (see
[`select`](#select-waiting-on-multiple-channels) below). In the happy path the
context is never cancelled, so it collects all N results and every goroutine has
sent its value and exited — no goroutine leaks.

If the caller *does* cancel, the loop returns early. The goroutines that are
still running each send one value into `ch`, but because the channel is buffered
with room for every result, those sends succeed against the buffer instead of
blocking forever on a receiver that has gone away. That is the subtle reason the
buffer matters: it guarantees no goroutine leaks *even when we stop receiving
early*.

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
// go/internal/authz/adapters/graph/parallel.go
func BulkCheck(ctx context.Context, auth Checker, reqs []shared.CheckRequest) []BulkResult {
    results := make([]BulkResult, len(reqs))
    var wg sync.WaitGroup

    for i, req := range reqs {
        wg.Add(1)
        go func(i int, req shared.CheckRequest) {
            defer wg.Done()
            result, err := auth.Evaluate(ctx, req)
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

Both functions accept a `context.Context` as their first parameter. A context
carries a cancellation signal: the caller can cancel it (or set a timeout/
deadline), and code that watches `ctx.Done()` can then stop early instead of
doing work nobody is waiting for anymore.

`ctx.Done()` returns a channel. It is the idiomatic Go cancellation pattern: the
channel stays open (blocks any receive) while the context is live, and is closed
the moment the context is cancelled. A receive on a closed channel returns
immediately — so `<-ctx.Done()` is "block until cancelled." After that,
`ctx.Err()` tells you why (`context.Canceled` or `context.DeadlineExceeded`).

`GraphEvaluator.Evaluate` itself currently ignores the context (it is an
in-memory traversal, so cancellation is not worth the complexity). A real network
call would pass `ctx` to the HTTP client and abort automatically. The interface
requires the parameter so swapping in the real OpenFGA client later needs no
changes at the call site.

```go
// context.Background() in tests means "never cancel"
result, err := ev.Evaluate(context.Background(), req)
```

But `AllPermissions` *does* act on the context, even though the underlying
evaluator ignores it — it stops collecting and returns as soon as the caller
cancels. It does that with `select`.

## `select`: waiting on multiple channels

`select` is Go's channel-aware switch. It blocks until one of its cases can
proceed; if several are ready at once it picks one at random. This is how a
goroutine waits on more than one channel at the same time.

`AllPermissions` uses it in the collector loop to wait on two things at once: the
next result, or the caller giving up.

```go
select {
case out := <-ch:
    // a result arrived — record it
    summary[out.relation] = out.allowed
case <-ctx.Done():
    // the caller cancelled or timed out — stop early
    return nil, ctx.Err()
}
```

Without the `ctx.Done()` case, a slow or hung backend would force the caller to
wait for every check no matter what — even after they timed out. With it, the
function returns the moment the context is cancelled.

Why is returning early still leak-free? The result channel is **buffered** with
one slot per relation, so the goroutines that are still in flight can each finish
their one send into the buffer and exit, even though nobody is receiving anymore.
An *unbuffered* channel would be a bug here: those orphaned goroutines would block
forever on a send with no receiver, leaking one goroutine per outstanding check.
This is the concrete payoff of choosing a buffered channel.

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
