# Go Learning Path and Practice Plan

This repository is a Go course disguised as a small authorization service.
That is useful: real Go is not learned by memorizing syntax in a blank file.
It is learned by reading small packages, changing behavior, breaking tests,
fixing them, and noticing which choices make the code easier to trust.

If you already program in another language, your biggest adjustment is not
syntax. It is taste:

- Go prefers explicit control flow over clever control flow.
- Go code usually pays for abstraction only after the duplication hurts.
- Errors are ordinary values, so failure paths stay visible.
- Interfaces are tiny behavioral promises, often owned by the consumer.
- Concurrency is a tool, not a seasoning.

Keep this page open while you work through the repo. It is the checklist, gym
program, and field guide for learning Go here.

## The Practice Loop

Use the same loop for every chapter:

1. Read the doc until you reach the first code reference.
2. Open the referenced Go file.
3. Predict what one function or test will do.
4. Run the smallest relevant test.
5. Change one thing.
6. Predict the failure.
7. Run the test again.
8. Restore or improve the code.
9. Explain the result in plain English.

That final explanation matters. If you cannot explain a Go idea without leaning
on vocabulary from another language, you probably copied the motion without
learning the move.

## What To Read

For your goal, do not treat the example chapters as optional. They are optional
for understanding ReBAC, but they are part of the Go curriculum.

| Stage | Read | What you should be able to do afterward |
|---|---|---|
| Orientation | `09` | Know how to use the repo as a Go practice course |
| Foundations | `10`, `11`, `12`, `13` | Read and change basic Go packages, tests, HTTP handlers, and JSON code |
| Idioms | `14` | Recognize what "good Go" means in this project |
| Repository Go | `20`, `21`, `28`, `29` | Follow a real request through packages and add a feature end to end |
| Advanced practice | `22`, `23`, `24`, `25` | Use concurrency, generics, interfaces, embedding, benchmarks, fuzzing, and race tests |

Then read the ReBAC chapters so the code has a real problem to solve:

```text
02 -> 03 -> 04 -> 05 -> 07 -> 27
```

## Coverage Map

This is the practical Go coverage contract for the repo.

| Concept | Where it is taught | Where to practice |
|---|---|---|
| Toolchain, modules, `go.mod`, package names, imports | `10`, `20` | `go test ./...`, `go doc ./internal/rebac` |
| Exported names, `internal`, `cmd`, package boundaries | `10`, `12`, `20` | `internal/rebac`, `cmd/server` |
| Variables, constants, named types, conversions | `10` | `internal/rebac/rebac.go` |
| Functions, multiple returns, closures, variadic calls | `10`, `23` | parser tests, `examples/generics` |
| Control flow, `range`, `switch`, `defer` | `10` | evaluator traversal and store locks |
| Structs, composite literals, constructors, zero values | `10`, `11`, `14` | `internal/authz`, `internal/documents` |
| Pointers, values, receivers, method sets | `11`, `14` | document service copy/update logic |
| Slices, maps, strings, bytes, runes, nil | `11` | token verifier, tuple store |
| Errors, wrapping, sentinels, typed errors | `12`, `14` | service and HTTP error mapping |
| `panic`, `recover`, and when not to use them | `12`, `23` | `Result.Unwrap` as a deliberate teaching example |
| Interfaces, consumer-owned contracts, test doubles | `12`, `14`, `24` | `internal/documents`, `internal/authz` |
| Embedding, decorators, middleware-style wrapping | `12`, `24` | `examples/middleware` |
| Generics, constraints, type inference, when to avoid them | `23` | `examples/generics` |
| Goroutines, channels, `select`, `WaitGroup`, cancellation | `22` | `examples/concurrency` |
| Mutexes, race detector, shared memory | `11`, `13`, `22`, `25` | in-memory stores |
| Context, deadlines, cancellation propagation | `13`, `22`, `28` | HTTP handler -> service -> evaluator |
| `net/http`, `ServeMux`, handlers, middleware shape | `13`, `28`, `33` | `internal/api`, `examples/authzhttp` |
| JSON decoding/encoding and request boundaries | `13` | `internal/api/json.go` |
| Testing, subtests, helpers, fakes, contract tests | `12`, `25` | `internal/*_test.go` |
| Fuzzing, benchmarks, coverage, race tests | `25` | parser and evaluator tests |
| Build, vet, staticcheck, modernize checks | `10`, `12`, `20`, `25` | `make check`, `make modernize` |
| Composition roots and dependency wiring | `12`, `13`, `20`, `28` | `cmd/server/main.go` |
| Ports and adapters, package design, replacement seams | `06`, `20`, `21`, `26`, `34` | in-process evaluator vs OpenFGA adapter |

That is enough Go to work professionally in this codebase. The official Go
spec still exists for edge cases; this course focuses on the concepts you will
actually touch while writing service code.

## How To Practice Without Getting Lost

Use small, reversible drills. Do not start by rewriting architecture. That is
how people build a maze and call it learning.

### Drill 1: Read a Type Like a Contract

Open `internal/rebac/rebac.go`.

Answer:

- Which names are exported?
- Which named types prevent string mixups?
- Which functions construct valid ReBAC IDs?
- Which functions return errors instead of assuming valid input?

Then add one invalid parser test. Run only that package:

```bash
go test ./internal/rebac
```

### Drill 2: Trace a Request

Run:

```bash
go test -v -run TestTrace ./internal/authz
```

Read the output as a graph walk, not as a wall of strings. The evaluator starts
at the requested document permission and works backward through relationships
until it either finds the user or runs out of branches.

### Drill 3: Break Encapsulation On Purpose

Open `internal/documents/token.go`.

Find where scope slices are copied. Temporarily remove a copy, write or adjust a
test that mutates the returned slice, and observe the bug. Then restore the
copy. This teaches slice aliasing faster than a diagram.

### Drill 4: Add One Permission Case

Add a denied case to an authorization table test. Good authorization tests prove
both sides:

- a user with a path is allowed
- a user without that path is denied

Over-granting is the bug you are hunting.

### Drill 5: Add the Delete Feature

Do `29-go-guided-feature-lab.md` after you can read the packages. It forces you
to touch the service, HTTP boundary, authorization check, and tests. That is
the point where Go stops being isolated facts and becomes a workflow.

## What Good Go Looks Like Here

When you wonder whether a change is idiomatic, ask these questions:

- Is the package boundary clear?
- Did I keep the interface near the consumer?
- Did I return a concrete type from the constructor?
- Does the error say what failed and preserve the cause with `%w`?
- Does the test prove behavior instead of implementation trivia?
- Did I pass `context.Context` through request-scoped work?
- Did I avoid shared mutable state, or protect it with a lock?
- Did I use generics or concurrency because they simplify the problem, not
  because they were available?
- Can someone find the wiring in `cmd/server` instead of hunting through domain
  packages for environment variables?

If the answer is mostly yes, the code is probably pointed in the right
direction.

## The "New To Go" Trap List

These are the traps most experienced developers hit when switching to Go:

- Creating interfaces before there are consumers.
- Returning broad interfaces from constructors.
- Hiding errors behind logging instead of returning them with context.
- Treating `nil` slices, empty slices, nil maps, and empty maps as identical.
- Forgetting that slices and maps can share mutable backing data.
- Starting goroutines without knowing how they stop.
- Using a mutex as decoration instead of protecting a specific invariant.
- Using `panic` for expected request failures.
- Building a `util` package because naming the real concept felt hard.
- Writing tests that only prove the happy path.

The repo is intentionally small enough that you can catch each of those traps
in code you can hold in your head.

## A Two-Week Plan

This pace assumes you already program and can spend focused time each day.

| Day | Work |
|---|---|
| 1 | Read `10`; run parser tests; inspect `internal/rebac` |
| 2 | Read `11`; do the slice-copy drill |
| 3 | Read `12`; add one table test and one fake-backed service test |
| 4 | Read `13`; run the HTTP tests and trace one request |
| 5 | Read `14`; refactor one tiny thing, then revert if it was not better |
| 6 | Read `02`-`05`; draw the Alice permission path |
| 7 | Read `07` and `27`; run `make trace` |
| 8 | Read `20`, `21`, `28`; explain package dependencies out loud |
| 9 | Read `22`; run race tests; write the benchmark suggested there |
| 10 | Read `23`; add a constrained generic helper in the example package |
| 11 | Read `24`; experiment with `ReadOnlyStore` capability boundaries |
| 12 | Read `25`; run fuzzing and a benchmark |
| 13 | Start `29`; write failing tests first |
| 14 | Finish `29`; run the full quality loop |

Use Docker-backed `make` commands when you want the pinned environment. Use
local `go` commands when you want faster iteration and your local Go version
matches the toolchain.

## Quality Loop

Before treating an exercise as done:

```bash
gofmt -w .
go test ./...
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck ./...
go test -race ./...
```

The `make check` target runs the same kind of loop in the containerized tool
environment. `make modernize` asks Go 1.26 analyzers whether the code can use
newer standard-library forms.

## Checkpoint

You are ready to leave this page when you can answer:

- Which docs teach core Go, and which docs apply it to the ReBAC service?
- Where would you add a new use case?
- Where would you add a new HTTP route?
- Where would you add a new authorization rule?
- Which command gives you the fastest feedback for the file you just changed?
- Which command gives you the highest confidence before you stop?

Next: [Go Toolchain and Core Syntax](10-go-toolchain-and-syntax.md).
