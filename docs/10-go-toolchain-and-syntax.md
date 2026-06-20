# Go Toolchain and Core Syntax

This chapter assumes you can program in another language but have not used Go.
It provides enough syntax and tooling to read, change, and run this repository
without completing a separate Go course.

## Install or Use the Container

The repository declares its Go version in `go.mod`:

```text
go 1.25.0
toolchain go1.26.4
```

You can use either:

- Docker and `make`, which run the pinned tools in a container
- a local Go installation compatible with the declared toolchain

Check your setup:

```bash
make build
make test
```

With local Go, use:

```bash
go version
go build ./...
go test ./...
```

`./...` means the current package and every package below it.

## Modules, Packages, and Files

`go.mod` defines one module:

```text
module rebac-primer
```

Imports beginning with `rebac-primer/` refer to packages in this module:

```go
import "rebac-primer/internal/rebac"
```

Every `.go` file begins with a package declaration:

```go
package rebac
```

Files in the same directory normally use the same package name and share
declarations. There are no header files. Import cycles are rejected by the
compiler.

Names beginning with an uppercase letter are exported from their package:

```go
type Object string // exported
func ParseObject(...) // exported

func splitSubject(...) // package-private
```

Exported means visible to another Go package, not necessarily to another module.
Directories named `internal` add another boundary: only code beneath the parent
of `internal` can import them.

## Variables and Constants

Use `var` when the type or zero value matters:

```go
var count int                 // 0
var err error                 // nil
var tuples []rebac.TupleKey   // nil slice
var store map[string]string   // nil map
```

Inside a function, `:=` declares and initializes at least one new variable:

```go
object := rebac.Document("roadmapDocument")
result, err := evaluator.Evaluate(ctx, request)
```

Use `=` when the variables already exist:

```go
result, err = evaluator.Evaluate(ctx, secondRequest)
```

Go permits multiple assignment:

```go
left, right = right, left
```

Constants are compile-time values:

```go
const defaultMaxDepth = 100
const RelationDocumentOwner Relation = "owner"
```

Go commonly uses named constants instead of enums:

```go
type ObjectType string

const (
    ObjectTypeUser      ObjectType = "user"
    ObjectTypeDocument  ObjectType = "document"
)
```

## Basic Types and Conversions

Common built-in types include:

```text
bool
string
int, int64
uint, byte
float64
```

Named types are distinct even when they have the same underlying representation:

```go
type Object string
type Subject string
```

Convert explicitly:

```go
object := Object("document:roadmap")
raw := string(object)
```

Go does not perform broad implicit numeric or named-type conversions.

## Functions and Multiple Returns

A function declares parameter and result types:

```go
func ParseObject(s string) (ObjectType, string, error) {
    // ...
}
```

Multiple returns are commonly used for a value plus an error:

```go
typ, id, err := rebac.ParseObject("document:roadmapDocument")
if err != nil {
    return err
}
```

Use `_` to deliberately ignore a value:

```go
typ, _, err := rebac.ParseObject(string(object))
```

Functions are values and can be passed as arguments:

```go
func Map[T, U any](value T, transform func(T) U) U {
    return transform(value)
}
```

## Control Flow

Parentheses around conditions are not used:

```go
if err != nil {
    return err
}
```

An `if` can initialize a value scoped to its branches:

```go
if err := validate(request); err != nil {
    return err
}
```

`for` is Go's only loop keyword:

```go
for i := 0; i < 3; i++ {
    // classic loop
}

for condition {
    // while-style loop
}

for {
    // infinite loop
}
```

`range` iterates over collections:

```go
for index, tuple := range tuples {
    fmt.Println(index, tuple)
}

for _, tuple := range tuples {
    fmt.Println(tuple)
}

for key, value := range permissions {
    fmt.Println(key, value)
}
```

`switch` cases do not fall through by default:

```go
switch typ {
case rebac.ObjectTypeUser:
    return validateUser(id)
case rebac.ObjectTypeDocument:
    return validateDocument(id)
default:
    return fmt.Errorf("unsupported type %q", typ)
}
```

## Structs and Composite Literals

A struct groups named fields:

```go
type CheckRequest struct {
    User     Object
    Relation Relation
    Object   Object
}
```

Prefer keyed literals, especially outside small local types:

```go
request := rebac.CheckRequest{
    User:     rebac.User("alice"),
    Relation: rebac.RelationDocumentCanEdit,
    Object:   rebac.Document("roadmapDocument"),
}
```

Omitted fields receive their zero values.

## `defer`

`defer` schedules a function call for the surrounding function's return. Calls
run in last-in, first-out order:

```go
file, err := os.Open(name)
if err != nil {
    return err
}
defer file.Close()
```

This repository uses `defer` to:

- unlock mutexes
- cancel contexts
- mark `WaitGroup` work complete
- remove graph nodes from the active traversal path

Arguments to a deferred call are evaluated when `defer` executes, not when the
function returns.

## Formatting and Documentation

`gofmt` is the canonical formatter:

```bash
gofmt -w .
```

Comments on exported declarations should begin with the declaration's name:

```go
// GraphEvaluator answers permission checks by walking the relationship graph.
type GraphEvaluator struct {
    // ...
}
```

View package documentation locally:

```bash
go doc ./internal/authz
go doc ./internal/authz.GraphEvaluator
```

## Try It

Run:

```bash
go test ./internal/rebac
go doc ./internal/rebac
```

Then open `internal/rebac/rebac.go` and identify:

1. three named types
2. one constant group
3. one keyed struct literal
4. one function returning multiple values

Next, search the repository for an assignment using `_` to ignore one of those
returned values:

```bash
rg ':=.*_' --glob '*.go'
```

## Checkpoint

You are ready to continue when you can explain:

- the difference between a module, package, and file
- why `Object("x")` is an explicit conversion
- when `:=` is legal
- what `defer` guarantees
- why `ParseObject` returns an error instead of throwing an exception

Next: [Values, pointers, collections, and methods](11-go-values-pointers-and-methods.md).
