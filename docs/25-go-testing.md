# Go Testing

Go tests live beside production code in `_test.go` files and run with:

```bash
go test ./...
```

## Patterns Used Here

| Pattern | Example |
|---|---|
| Arrange/Act/Assert | `internal/authz/evaluator_test.go` |
| table-driven tests | permission matrix tests |
| contract tests | `internal/authz/contract` |
| fuzz tests | `FuzzParseObject` |
| benchmarks | `BenchmarkGraphEvaluator_Evaluate` |
| race detector | `go test -race ./...` |

## Useful Commands

```bash
go test ./internal/authz
go test -v -run TestTrace ./internal/authz
go test -run TestGraphEvaluator_PermissionMatrix ./internal/authz
go test -bench=. -benchtime=5s ./internal/authz
go test -fuzz=FuzzParseObject -fuzztime=30s ./internal/authz
go test -race ./...
```

Authorization tests should cover both allowed and denied paths. The denied cases
are where many authorization bugs hide.

## The Test Pyramid in This Repository

```text
parsers and stores       -> small unit tests
services                 -> behavior and error propagation
authorization contract   -> shared allow/deny truth table
HTTP handlers            -> authn + scope + ReBAC + response mapping
OpenFGA contract         -> optional live-backend parity test
```

The shared contract is particularly important. The in-process evaluator and
OpenFGA model encode the same policy in different forms; running the same truth
table against both catches model drift.

## Test the Reason for a Denial

Two requests may both return HTTP 403 for different reasons:

```text
token lacks documents:write  -> OAuth scope denial
token has scope, user is viewer -> ReBAC denial
```

Keep separate tests for those paths. A status-only test can pass while the
request is being rejected by the wrong layer.

## Fuzzing and Benchmarks

Fuzz parsers and boundary code where arbitrary input can reveal panics or broken
invariants. Benchmarks answer performance questions; they are not pass/fail
correctness tests. Compare alternatives before adding caches, indexes, or
concurrency.

## Checkpoint

Why are allow-only authorization tests insufficient? Because over-granting and
rejecting a request at the wrong layer are both security bugs, and neither is
proven safe by a successful allow case.
