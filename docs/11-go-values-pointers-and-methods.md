# Go Values, Pointers, Collections, and Methods

Go's default model is value semantics: assignment and function calls copy
values. Pointers, slices, maps, and interfaces add important qualifications.
This chapter makes those rules concrete using this repository.

## Values Are Copied

Assigning a struct copies its fields:

```go
original := CollaborativeDocument{Body: "first"}
updated := original
updated.Body = "second"
```

`original.Body` remains `"first"`.

The document service deliberately uses this pattern:

```go
updated := *existing
updated.Body = input.Body
updated.UpdatedBy = input.Actor
```

`existing` is a pointer. `*existing` dereferences it and copies the document
value, so the loaded object is not mutated before the repository accepts the
save.

## Pointers

`*T` is a pointer to a value of type `T`:

```go
var doc *CollaborativeDocument // nil
```

`&value` takes an address:

```go
doc := CollaborativeDocument{ID: "roadmap"}
ptr := &doc
```

`*ptr` dereferences the pointer:

```go
copyOfDoc := *ptr
```

Go automatically dereferences pointers for field access:

```go
ptr.Body = "updated" // equivalent to (*ptr).Body = "updated"
```

Unlike C, Go does not support pointer arithmetic.

## Pointer or Value?

Use a pointer when at least one of these is true:

- the function or method must mutate the caller's value
- copying the value would be meaningfully expensive
- `nil` communicates absence
- the type contains synchronization primitives and must not be copied
- consistency makes a method set easier to understand

Use a value when it is small, immutable in practice, and naturally copied.

This repository passes `rebac.CheckRequest` and `rebac.TupleKey` by value because
they are small data records. Services and stores use pointers because they have
identity, internal state, or mutexes.

Do not choose pointers merely to imitate reference-oriented languages.

## Methods and Receivers

A method is a function with a receiver:

```go
func (s *Service) Read(ctx context.Context, id string, actor rebac.Object) (
    *CollaborativeDocument,
    error,
) {
    // ...
}
```

A pointer receiver can mutate the receiver and belongs to the method set of
`*Service`. A value receiver receives a copy:

```go
func (o Object) String() string {
    return string(o)
}
```

For a given type, normally keep receivers consistently pointer-based or
value-based. Types containing `sync.Mutex` or `sync.RWMutex` must not be copied
after first use, so their methods should use pointer receivers.

## Zero Values and Constructors

Every Go variable has a value even without explicit initialization:

| Type | Zero value |
|---|---|
| `bool` | `false` |
| number | `0` |
| `string` | `""` |
| pointer, map, slice, function, interface | `nil` |
| struct | each field's zero value |

Useful zero values reduce setup:

```go
var wg sync.WaitGroup
var mu sync.Mutex
```

Not every application type has a useful zero value. `GraphEvaluator` needs a
store and depth limit, so callers use:

```go
evaluator := authz.NewGraphEvaluator(store)
```

Constructors in Go are ordinary functions conventionally named `New` or
`NewType`. The language has no constructor keyword.

## Arrays and Slices

An array's length is part of its type:

```go
var fixed [3]string
```

Most Go code uses slices:

```go
relations := []rebac.Relation{
    rebac.RelationDocumentCanRead,
    rebac.RelationDocumentCanEdit,
}
```

A slice is a small descriptor pointing at an underlying array. It has a length
and capacity:

```go
len(relations)
cap(relations)
```

Append may reuse the existing array or allocate a new one:

```go
relations = append(relations, rebac.RelationDocumentCanDelete)
```

Always assign the result of `append`.

Two slices can share an underlying array, so changing an element through one may
be visible through the other. To make an independent copy:

```go
copyOfScopes := append([]string(nil), scopes...)
```

The final `...` expands a slice into variadic arguments.

A nil slice is safe to range over and append to. It has length zero. For JSON,
however, a nil slice usually encodes as `null`, while a non-nil empty slice
encodes as `[]`.

## Maps

Create a writable map with `make` or a literal:

```go
permissions := make(map[rebac.Relation]bool)

permissions := map[rebac.Relation]bool{
    rebac.RelationDocumentCanRead: true,
}
```

Reading a missing key returns the value type's zero value:

```go
allowed := permissions[relation]
```

Use the comma-ok form when absence differs from a stored zero value:

```go
allowed, exists := permissions[relation]
```

Delete is safe even when the key is absent:

```go
delete(permissions, relation)
```

A nil map can be read but writing to it panics. Maps are reference-like runtime
values; assigning a map does not clone its contents.

Map iteration order is deliberately unspecified. Sort keys when stable output
matters.

## Strings, Bytes, and Runes

Strings are immutable byte sequences, conventionally UTF-8:

```go
s := "Go"
len(s) // bytes
```

Use `[]byte` for mutable bytes or I/O buffers. Use `rune` (an alias for `int32`)
for a Unicode code point:

```go
for _, r := range "Gophér" {
    fmt.Printf("%c\n", r)
}
```

Indexing a string returns a byte, not necessarily a complete character.

## Interfaces and Nil

An interface value conceptually contains a dynamic type and dynamic value. It is
nil only when both are absent:

```go
var err error // nil interface
```

A pointer stored inside an interface can be nil while the interface itself is
non-nil:

```go
var typed *MyError = nil
var err error = typed
fmt.Println(err == nil) // false
```

Avoid returning typed nil pointers as interfaces. Return a literal `nil` error
on success.

## Copying and Concurrency

Slices and maps require care because copies can share data. Mutex-protected
types require care because copying the mutex creates a different lock.

This repository's stores use pointer receivers and return copied slices where
callers should not mutate internal state. Run:

```bash
go test -race ./internal/authz ./internal/documents
```

The race detector finds concurrent unsynchronized access that ordinary tests may
miss.

## Try It

Read `internal/documents/token.go` and find where scope slices are copied.
Temporarily remove one copy, write a test that mutates the returned slice, and
observe how the verifier's internal state can be changed. Restore the copy after
the experiment.

Then read `internal/documents/service.go` and answer why `updated := *existing`
is safer than directly modifying `existing.Body`.

## Checkpoint

You are ready to continue when you can explain:

- why assigning a struct and assigning a map behave differently
- why `append` must be assigned back
- when a pointer receiver is necessary
- the difference between a nil slice and nil map
- why an interface containing a nil pointer may not equal `nil`

Next: [Errors, interfaces, packages, and testing](12-go-errors-interfaces-and-testing.md).
