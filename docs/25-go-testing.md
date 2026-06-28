# Go Testing

Go tests live beside production code in `_test.go` files and run with:

```bash
go test ./...
```

The testing story is one of Go's strengths: no separate test language, no
required framework, no build step mystery. A test is just Go code that calls
your code and reports what happened.

## Test Files And Packages

A test function starts with `Test`:

```go
func TestParseObject(t *testing.T) {
    typ, id, err := ParseObject("document:roadmap")
    if err != nil {
        t.Fatalf("ParseObject returned error: %v", err)
    }
    if typ != ObjectTypeDocument {
        t.Errorf("type = %q, want %q", typ, ObjectTypeDocument)
    }
    if id != "roadmap" {
        t.Errorf("id = %q, want roadmap", id)
    }
}
```

Use `t.Fatal` or `t.Fatalf` when the rest of the test cannot continue. Use
`t.Error` or `t.Errorf` when more assertions can still produce useful output.

Tests can use the same package:

```go
package rebac
```

or an external test package:

```go
package rebac_test
```

Same-package tests can access unexported names. External tests see the package
as consumers see it. Use external tests when you want to protect the public API
from accidental assumptions.

## Focused Commands

Run the whole suite:

```bash
go test ./...
```

Run one package:

```bash
go test ./internal/authz
```

Run one test:

```bash
go test -run TestGraphEvaluator_TeamMemberCanEditDocument ./internal/authz
```

Run one subtest:

```bash
go test -run 'TestGraphEvaluator_PermissionMatrix/alice_can_edit' ./internal/authz
```

Add `-v` when test logs or subtest names matter:

```bash
go test -v -run TestTrace ./internal/authz
```

## Table-Driven Tests

Table tests make repeated input/output rules visible:

```go
func TestParseObject(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {name: "document", input: "document:roadmap"},
        {name: "missing separator", input: "document", wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, _, err := ParseObject(tt.input)
            if (err != nil) != tt.wantErr {
                t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

Each row should describe behavior, not mirror implementation details. A table
with twenty vague rows is worse than five rows with names that explain the rule.

## Helpers

Move noisy setup into helpers when it makes the behavior clearer:

```go
func newEvaluator(t *testing.T, tuples ...rebac.TupleKey) *authz.GraphEvaluator {
    t.Helper()
    store := authz.NewInMemoryStore(tuples...)
    return authz.NewGraphEvaluator(store)
}
```

`t.Helper()` tells the test runner to report failures at the call site, not
inside the helper. Use helpers for setup and repeated assertions. Do not hide
the essential behavior of the test behind a helper name.

Useful helpers from `testing.T`:

| Helper | Use |
|---|---|
| `t.Helper()` | mark helper functions |
| `t.Cleanup(fn)` | register cleanup for this test |
| `t.TempDir()` | create an automatically cleaned temporary directory |
| `t.Setenv(k, v)` | set an environment variable for this test only |
| `t.Logf(...)` | print diagnostics when `-v` is used or the test fails |

## Fakes And Stubs

Small consumer-owned interfaces make hand-written fakes easy:

```go
type fakeChecker struct {
    result rebac.CheckResult
    err    error
}

func (f fakeChecker) Check(
    context.Context,
    rebac.CheckRequest,
) (rebac.CheckResult, error) {
    return f.result, f.err
}
```

Prefer fakes and stubs when they keep the test obvious. Avoid tests that assert
every internal call unless the call itself is the behavior being promised.

Good service tests usually prove:

- the right authorization question is asked
- allowed users reach the repository operation
- denied users do not reach the repository operation
- meaningful errors are preserved

## Contract Tests

Contract tests run the same behavior suite against multiple implementations.

This repo uses that idea for authorization. The in-process graph evaluator and
OpenFGA model must agree on the allow/deny truth table. That catches drift when
one implementation changes and the other does not.

Contract tests are useful when:

- a concrete implementation may be swapped
- an adapter talks to an external service
- several stores should obey the same semantics
- behavior matters more than implementation details

The contract should describe externally visible behavior. It should not demand
that every implementation use the same private algorithm.

## HTTP Tests

Use `httptest` for handlers:

```go
request := httptest.NewRequest(http.MethodGet, "/documents/roadmapDocument", nil)
response := httptest.NewRecorder()

handler.ServeHTTP(response, request)

if response.Code != http.StatusOK {
    t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
}
```

Test status codes, important headers, decoded bodies, and whether malformed
input is rejected before domain dependencies are called.

Authorization HTTP tests should distinguish:

```text
token lacks documents:write     -> OAuth scope denial
token has scope, ReBAC denies   -> graph authorization denial
```

Both may produce HTTP 403. A status-only test can pass while proving the wrong
thing.

## Fuzzing

Fuzz tests generate inputs for code that accepts arbitrary data. They are
excellent for parsers and boundary code:

```bash
go test -fuzz=FuzzParseObject -fuzztime=30s ./internal/rebac
```

A fuzz target should check invariants:

- invalid input returns an error, not a panic
- valid parsed values round-trip when appropriate
- no input causes unbounded work

Fuzzing is not a substitute for named examples. It finds edge cases; it does
not explain the product rule to the next reader.

## Benchmarks

Benchmarks answer performance questions:

```bash
go test -bench=. -benchtime=5s ./internal/authz
```

A benchmark function starts with `Benchmark` and uses `b.N`:

```go
func BenchmarkGraphEvaluator_Evaluate(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _, _ = evaluator.Evaluate(context.Background(), request)
    }
}
```

Use benchmarks before adding caches, indexes, or concurrency. Performance
guesses are cheap and often wrong.

## Race Detector

The race detector catches unsafe concurrent access at runtime:

```bash
go test -race ./...
```

A data race happens when two goroutines access the same memory location, at
least one access is a write, and there is no synchronization. Race tests are
slower, but they are exactly the right tool for in-memory stores, goroutines,
and shared maps.

## Coverage

Coverage reports which statements were executed:

```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Coverage is a map, not a grade. High coverage can still miss the important
denial case. Low coverage can expose an untested package boundary. Use it to
ask better questions.

## Useful Commands

```bash
go test ./internal/authz
go test -v -run TestTrace ./internal/authz
go test -run TestGraphEvaluator_PermissionMatrix ./internal/authz
go test -bench=. -benchtime=5s ./internal/authz
go test -fuzz=FuzzParseObject -fuzztime=30s ./internal/rebac
go test -race ./...
go test -count=1 ./...       # bypass the test cache
go test -shuffle=on ./...    # expose order dependencies
```

The repository-level loop is:

```bash
gofmt -w .
go test ./...
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck ./...
go test -race ./...
```

`make check` runs the containerized version of this loop.

## Patterns Used Here

| Pattern | Example |
|---|---|
| Arrange/Act/Assert | `internal/authz/evaluator_test.go` |
| table-driven tests | permission matrix tests |
| subtests | parser and matrix tests |
| fakes/stubs | document service tests |
| contract tests | `internal/authz/contract` |
| fuzz tests | parser fuzz target |
| benchmarks | graph evaluator benchmark |
| race detector | in-memory stores and concurrency examples |
| HTTP handler tests | `internal/api/handler_test.go` |

## Try It

Do these in order:

1. Add a table row to a parser or validator test.
2. Add `t.Helper()` to a noisy setup helper and observe failure locations.
3. Run `TestTrace` with `-v` and explain the graph walk.
4. Add a denied authorization case that would catch over-granting.
5. Run the parser fuzz target for 30 seconds.
6. Run an evaluator benchmark and write down what question it answers.
7. Run `go test -race ./examples/concurrency ./internal/authz`.

## Checkpoint

You are ready to continue when you can explain:

- when to use `Fatal` versus `Error`
- why table rows need meaningful names
- why `t.Helper()` changes failure reporting
- why contract tests are useful for OpenFGA parity
- why allow-only authorization tests are insufficient
- what fuzzing and benchmarks prove, and what they do not prove
- when to run the race detector

Next: [Guided Go Feature Lab](29-go-guided-feature-lab.md).
