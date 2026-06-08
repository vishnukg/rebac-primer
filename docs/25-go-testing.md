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
