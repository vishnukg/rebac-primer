# Go concurrency: goroutines, channels, and WaitGroups

Go's concurrency model is famously simple to write and famously easy to get wrong.
This chapter grounds every concept in `examples/concurrency/parallel.go`, which you
can run right now.

## The problem concurrency solves here

When a UI renders a permissions panel it needs to know all four computed
permissions for a document at once: can_read, can_comment, can_edit, can_delete.

If checks are independent network calls, sequential execution roughly adds
their latencies, while concurrent execution can approach the latency of the
slowest call. That can matter for an OpenFGA-backed UI.

For this repository's tiny in-memory evaluator, concurrency is likely slower
because goroutine and channel overhead dominates. This chapter uses permission
checks because they are familiar—not because every authorization check should
be parallelized. Measure before choosing concurrency.

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

A channel is a typed pipe. One goroutine sends; another receives. On an
unbuffered channel, a send blocks until a receiver is ready. On any channel, a
receive blocks while no value is available and the channel remains open. That
blocking is not a bug — it is how goroutines synchronise.

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
relations:

```go
// examples/concurrency/parallel.go
ch := make(chan outcome, len(relations))
```

Each goroutine sends exactly one value into the channel and then exits. Because
the channel is buffered to match the number of goroutines, no goroutine ever
blocks waiting for the collector — the sends always succeed immediately. The
collector runs the loop after all goroutines have been started:

```go
for _, rel := range relations {
    go func(rel rebac.Relation) {
        result, err := auth.Evaluate(ctx, rebac.CheckRequest{User: user, Relation: rel, Object: object})
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
buffer matters: once a worker returns, its final send cannot leak merely because
the collector stopped receiving.

## Why the goroutine passes `rel` as an argument

Look at this pattern:

```go
for _, rel := range relations {
    go func(rel Relation) {  // rel is a parameter, not a closed-over variable
        // use rel
    }(rel)
}
```

Since Go 1.22, range variables declared by the loop are created per iteration,
so closing over `rel` is safe in this example. Passing it explicitly is still a
reasonable teaching style because the goroutine's inputs are visible at the call
site. It is clarity, not a correctness requirement for this module's Go version.

## WaitGroups: wait without collecting

`BulkCheck` uses `sync.WaitGroup` instead of a channel:

```go
// examples/concurrency/parallel.go
func BulkCheck(ctx context.Context, auth Checker, reqs []rebac.CheckRequest) []BulkResult {
    results := make([]BulkResult, len(reqs))
    var wg sync.WaitGroup

    for i, req := range reqs {
        wg.Add(1)
        go func(i int, req rebac.CheckRequest) {
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
| **Signals first error** | Natural — send an error result and cancel remaining work | Awkward — needs shared state or an extra channel |
| **Use when** | results are all the same shape and order does not matter | you need results in input order |

`AllPermissions` uses channels because results come back unordered and we build
a map.

`BulkCheck` uses WaitGroup because callers need results in the same position as
their input requests.

It also starts one goroutine per request. That is fine for a small teaching
slice, but a production bulk API should bound concurrency with a worker pool or
semaphore so a huge request cannot create an unbounded number of goroutines.

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

`GraphEvaluator` honours the context: `hasRelation` checks `ctx.Err()` at the
top of every recursive call, so a cancelled or timed-out check stops mid-walk
and returns the context error instead of finishing work nobody is waiting for
(`internal/authz/evaluator_errors_test.go` proves it). A network-backed
evaluator like the OpenFGA adapter additionally passes `ctx` to its HTTP
client, which aborts the request in flight.

```go
// context.Background() in tests means "never cancel"
result, err := ev.Evaluate(context.Background(), req)
```

`AllPermissions` acts on the context at its own level too — it stops collecting
and returns as soon as the caller cancels. It does that with `select`.

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

This still depends on `auth.Evaluate` eventually returning—normally because it
honors the canceled context. A backend that ignores context and hangs forever
can still leak its worker. Buffering solves the abandoned-send problem, not an
uncooperative dependency.

## Race detector

Go ships a built-in race detector. Run your tests with `-race` to catch unsafe
concurrent access:

```bash
go test -race ./examples/concurrency ./internal/authz
```

A race condition occurs when two goroutines access the same memory location and
at least one is a write. The race detector instruments memory access at runtime
and reports violations immediately. Run it on every PR.

## Try it

**Measure instead of guessing:**

Add a benchmark that compares four sequential in-memory checks with
`AllPermissions`. The sequential version should usually win at this scale.
Then replace the evaluator with a fake that sleeps for 20 ms per check; the
concurrent version should finish much closer to 20 ms than 80 ms.

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
