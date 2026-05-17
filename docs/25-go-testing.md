# Go testing patterns: AAA, table-driven tests, benchmarks, and fuzz

Go ships a testing package in the standard library. No framework required. This
chapter covers the four patterns that appear in this repo, in order from most
common to most specialised.

Code references: `go/internal/authz/graph_test.go`, `go/internal/authz/parallel_test.go`,
`go/internal/authz/middleware_test.go`, `go/internal/authz/result_test.go`.

## The testing package basics

Every test file ends in `_test.go`. Every test function has this signature:

```go
func TestName(t *testing.T) { ... }
```

Run all tests in a package:

```bash
go test ./internal/authz/...
```

Run a specific test by name:

```bash
go test -run TestGraphAuthorizer_TeamMemberCanEditDocument ./internal/authz/...
```

Run with verbose output (print `t.Log` lines even on success):

```bash
go test -v ./internal/authz/...
```

## Pattern 1: Arrange / Act / Assert (AAA)

Every test in this repo follows the AAA style. The comments are literal:

```go
// go/internal/authz/graph_test.go
func TestGraphAuthorizer_CaseyIsDenied(t *testing.T) {
    // Arrange: Casey has no tuples in the graph.
    store := seedStore()
    auth := authz.NewGraphAuthorizer(store)
    req := authz.CheckRequest{
        User:     fixtures.Casey,
        Relation: authz.RelationDocumentCanEdit,
        Object:   fixtures.RoadmapDocument,
    }

    // Act
    result, err := auth.Check(context.Background(), req)

    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.Allowed {
        t.Error("expected Casey can_edit=false but got true")
    }
}
```

### `t.Fatal` vs `t.Error`

- `t.Fatal` stops the test immediately. Use it when a failure makes the rest of
  the test meaningless — for example, if the function returned an unexpected
  error and the next assertion would panic on a nil pointer.
- `t.Error` records the failure and continues. Use it when multiple assertions
  are independent and you want all failures reported in one run.

### `t.Logf` for diagnostic context

Printed only when the test fails (or with `-v`):

```go
for _, line := range result.Trace {
    t.Logf("  trace: %s", line)
}
```

This makes failing tests self-documenting. You see the graph trace alongside the
failure message without adding it to the always-visible output.

### Shared setup helpers

`seedStore` in `graph_test.go` is a helper that builds the fixture store:

```go
func seedStore(extra ...authz.TupleKey) *authz.InMemoryTupleStore {
    all := append(fixtures.SeedRelationshipTuples(), extra...)
    return authz.NewInMemoryTupleStore(all...)
}
```

The variadic `extra` parameter lets individual tests add tuples without forking
the fixture. This is the Go equivalent of a Vitest `beforeEach` factory.

## Pattern 2: Table-driven tests

When the same behaviour needs verifying against many inputs, a table-driven test
is cleaner than N duplicate functions.

```go
// go/internal/authz/graph_test.go
func TestGraphAuthorizer_PermissionMatrix(t *testing.T) {
    store := seedStore()
    auth := authz.NewGraphAuthorizer(store)

    rows := []struct {
        name     string
        user     authz.Object
        relation authz.Relation
        want     bool
    }{
        {"editor_can_read", fixtures.Alice, authz.RelationDocumentCanRead, true},
        {"viewer_cannot_edit", fixtures.Bob, authz.RelationDocumentCanEdit, false},
        // ...
    }

    for _, row := range rows {
        t.Run(row.name, func(t *testing.T) {
            result, err := auth.Check(context.Background(), authz.CheckRequest{
                User:     row.user,
                Relation: row.relation,
                Object:   fixtures.RoadmapDocument,
            })
            if err != nil { t.Fatalf(...) }
            if result.Allowed != row.want {
                t.Errorf("got allowed=%v, want %v", result.Allowed, row.want)
            }
        })
    }
}
```

### Why `t.Run`

`t.Run(name, func)` creates a sub-test. Failures appear as:

```
--- FAIL: TestGraphAuthorizer_PermissionMatrix/viewer_cannot_edit
```

You can run a single row:

```bash
go test -run TestGraphAuthorizer_PermissionMatrix/viewer_cannot_edit ./internal/authz/...
```

That is invaluable when debugging one failing case in a matrix of ten.

### Anonymous struct for the table

The table is a slice of anonymous structs — not a map, not a named type. The
fields are visible at declaration; you do not need to hunt for a type definition
elsewhere. This is idiomatic Go: prefer local, obvious structure over distant
abstractions.

## Pattern 3: Benchmarks

A benchmark measures how fast code runs. The signature:

```go
func BenchmarkName(b *testing.B) { ... }
```

```go
// go/internal/authz/graph_test.go
func BenchmarkGraphAuthorizer_Check(b *testing.B) {
    store := seedStore()
    auth := authz.NewGraphAuthorizer(store)
    req := authz.CheckRequest{
        User:     fixtures.Alice,
        Relation: authz.RelationDocumentCanEdit,
        Object:   fixtures.RoadmapDocument,
    }
    ctx := context.Background()

    b.ResetTimer()
    for range b.N {
        auth.Check(ctx, req)
    }
}
```

`b.N` is set by the testing framework — it runs the loop enough times to get a
stable measurement. `b.ResetTimer()` excludes setup time from the measurement.

Run benchmarks:

```bash
go test -bench=. -benchtime=5s ./internal/authz/...
```

Sample output:

```
BenchmarkGraphAuthorizer_Check-10    500000    2345 ns/op
```

`ns/op` is nanoseconds per call. Use benchmarks before and after an optimisation
to confirm the change helped.

## Pattern 4: Fuzz tests

A fuzz test generates random inputs and looks for panics or invariant violations.
The signature:

```go
func FuzzName(f *testing.F) { ... }
```

```go
// go/internal/authz/graph_test.go
func FuzzParseObject(f *testing.F) {
    // Seed corpus: the fuzzer mutates these inputs.
    f.Add("user:alice")
    f.Add("")
    f.Add("user:")

    f.Fuzz(func(t *testing.T, s string) {
        typ, id, err := authz.ParseObject(s)
        if err != nil {
            return // invalid input — fine
        }
        // If parsing succeeded, round-tripping must hold.
        var obj authz.Object
        switch typ {
        case authz.ObjectTypeUser:
            obj = authz.User(id)
        // ...
        }
        if string(obj) != s {
            t.Errorf("round-trip failed: %q -> %s/%s -> %q", s, typ, id, obj)
        }
    })
}
```

The invariant here: if `ParseObject` succeeds, constructing the same object from
the parsed parts must reproduce the original string.

Run fuzzing for 30 seconds:

```bash
go test -fuzz=FuzzParseObject -fuzztime=30s ./internal/authz/...
```

The fuzzer saves any input that triggers a new code path into
`testdata/fuzz/FuzzParseObject/`. These become part of the corpus and run as
regular tests with `go test`. Commit the corpus alongside your code.

Without `-fuzz`, the fuzz test runs only the seed corpus — it is a normal test.
This means CI always runs the known interesting inputs.

## The race detector

Run all tests with the race detector enabled:

```bash
go test -race ./internal/...
```

The race detector instruments memory access at runtime and reports concurrent
reads/writes. It catches bugs like:

```go
// BulkCheck: two goroutines writing to results[i] and results[j] — safe.
// But if i == j, that would be a data race.
```

The race detector is not free: it slows tests by 2–20x. Run it in CI. Accept
the slowdown as the price of knowing your concurrent code is correct.

## `testdata/` — corpus and fixtures

Go recognises `testdata/` directories specially: they are excluded from
compilation but included in test runs. The fuzz corpus lives in
`testdata/fuzz/<FuzzName>/`. You can also put fixture files (JSON, YAML,
expected output) in `testdata/`.

## Comparing to Vitest

| Concept | Go | Vitest (TypeScript) |
|---|---|---|
| Test function | `func TestFoo(t *testing.T)` | `test("foo", () => { ... })` |
| Sub-tests | `t.Run("name", func(t *testing.T))` | `describe` / `test` nesting |
| Setup | helper function called in test | `beforeEach`, `beforeAll` |
| Assertions | `t.Error`, `t.Errorf`, `t.Fatal` | `expect(...).toBe(...)` |
| Mocks | write a fake struct satisfying an interface | `vi.fn()`, `vi.mock()` |
| Benchmarks | `func BenchmarkFoo(b *testing.B)` | no built-in equivalent |
| Fuzz | `func FuzzFoo(f *testing.F)` | no built-in equivalent |

The biggest difference: Go has no assertion library in the standard package. You
write `if got != want { t.Errorf(...) }` explicitly. This is verbose but it
means every assertion reads as plain Go — no magic matcher syntax to learn.

## Try it

**Run the permission matrix with a filter:**

```bash
go test -run TestGraphAuthorizer_PermissionMatrix/outside ./internal/authz/...
```

Only the rows matching `outside` run. Add a new row for a direct document owner
(add an extra tuple, set `want: true` for `can_delete`).

**Run the benchmark, then break it:**

```bash
go test -bench=BenchmarkGraphAuthorizer_Check -benchtime=3s ./internal/authz/...
```

Note the `ns/op`. Then modify `GraphAuthorizer` to add a `time.Sleep(1*time.Microsecond)` inside the loop and run the benchmark again. The number should jump.

**Run the fuzz test:**

```bash
go test -fuzz=FuzzParseObject -fuzztime=10s ./internal/authz/...
```

Look at what the fuzzer finds in `testdata/fuzz/FuzzParseObject/`. Add a
`t.Logf` inside the fuzz function to watch which inputs exercise new code paths.

## Checkpoint

Four questions, one per pattern:

1. What is the difference between `t.Fatal` and `t.Error`? When should you use each?
2. Why does `t.Run` make table-driven tests better than 10 separate test functions?
3. What does `b.ResetTimer()` do and why does it matter?
4. If you commit a file to `testdata/fuzz/FuzzParseObject/`, what happens when
   someone runs `go test` (without `-fuzz`)?

Good answers:
1. `t.Fatal` stops the test; `t.Error` continues. Use `t.Fatal` when failure
   makes subsequent assertions meaningless; use `t.Error` when they are independent.
2. Each row is its own named sub-test (`t.Run`), so failures identify themselves
   by name and you can re-run a single row with `-run`. Ten functions would
   require 10 names and you could not re-run them by row.
3. It excludes setup time (building the store, the authorizer) from the per-op
   measurement. Without it, the benchmark counts setup once and attributes its
   cost to `b.N` iterations, underreporting per-call cost.
4. The fuzz corpus runs as a normal test — the seeds are checked without fuzzing.
   This is how discovered bugs become permanent regression tests.
