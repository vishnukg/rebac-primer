# Go Idioms and Patterns

This chapter is about writing Go that feels like Go. Syntax tells the compiler
what you mean. Idioms tell the next maintainer whether you meant it clearly.

The examples here point at this repository. Treat them as habits to practice,
not slogans to memorize.

## Small Packages With Plain Names

Go package names are short, lower-case, and usually singular:

```text
rebac
authz
documents
api
```

The package name becomes part of every call site:

```go
rebac.Document("roadmapDocument")
authz.NewGraphEvaluator(store)
documents.New(repo, checker)
```

Avoid package names like `utils`, `helpers`, or `common`. Those names usually
mean "I have not found the real concept yet." If a function parses a ReBAC
object, it belongs with ReBAC vocabulary. If it maps HTTP errors, it belongs at
the HTTP boundary.

## Keep The Dependency Arrow Boring

The domain packages do not reach upward into HTTP or process configuration:

```text
api -> documents -> authz -> rebac
cmd/server -> wires concrete implementations
```

`cmd/server` is allowed to know about everyone because it is the composition
root. It chooses the in-memory evaluator or OpenFGA adapter, reads environment
variables, and starts the server.

Domain packages should not call `os.Getenv`, open network listeners, or choose
global dependencies. Hidden wiring makes tests hard and surprises cheap.

## Constructors Return Concrete Types

In Go, constructors are ordinary functions:

```go
func NewGraphEvaluator(store TupleReader) *GraphEvaluator
func NewInMemoryRepository(seed ...CollaborativeDocument) *InMemoryRepository
```

They usually return concrete types, not interfaces. Returning a concrete type
lets callers use all exported behavior and lets each consumer decide which
small interface it actually needs.

This is common:

```go
store := authz.NewInMemoryStore(tuples...)
evaluator := authz.NewGraphEvaluator(store)
```

This is suspicious unless there is a specific reason:

```go
func NewThing() SomeHugeInterface
```

A broad return interface hides useful behavior and pushes the provider's guess
about abstraction onto every caller.

## Accept Interfaces At The Boundary That Needs Them

The document service does not need to know how authorization is implemented. It
needs one capability:

```go
type AuthorizationService interface {
    Check(context.Context, rebac.CheckRequest) (rebac.CheckResult, error)
}
```

That interface belongs near the consumer. The provider does not need to publish
a giant "mock me" interface.

Rule of thumb:

```text
return concrete, accept small interfaces
```

This does not mean every parameter should be an interface. Use an interface when
the caller benefits from substitutability or when a package boundary needs a
capability contract. For ordinary data, pass ordinary data.

## Prefer Values Until Identity Matters

Small immutable records are usually passed by value:

```go
rebac.CheckRequest
rebac.TupleKey
rebac.CheckResult
```

Services, repositories, and evaluators are usually pointers because they have
identity, internal state, or methods that should not copy locks:

```go
*documents.Service
*authz.InMemoryStore
*authz.GraphEvaluator
```

Use a pointer when:

- the method must mutate the receiver
- copying would be expensive or incorrect
- `nil` is a meaningful absence value
- the type contains a mutex or other synchronization primitive
- consistency of the method set matters

Do not use pointers as a reflex from reference-oriented languages. A pointer is
a tool, not a medal for serious code.

## Make Zero Values Useful When It Fits

Many standard-library types are useful without construction:

```go
var wg sync.WaitGroup
var mu sync.Mutex
var buf bytes.Buffer
```

Not every domain type needs a useful zero value. A graph evaluator without a
store cannot evaluate anything, so construction is explicit:

```go
evaluator := authz.NewGraphEvaluator(store)
```

Both choices can be idiomatic. The question is whether the zero value represents
a real, safe state.

## Errors Need Context, Not Drama

Go error handling is repetitive because failure paths are part of the program.
Keep them visible and make them useful:

```go
if err != nil {
    return fmt.Errorf("check document permission: %w", err)
}
```

Use `%w` when callers should be able to inspect the original error with
`errors.Is` or `errors.As`. Use `%v` when you only want text.

Do not log and return the same error at every layer. That produces noisy logs
and still leaves the caller responsible for deciding what to do. Usually:

- lower layers add context and return
- HTTP boundaries map errors to status codes
- process boundaries log and exit

`panic` is for programmer mistakes and impossible states. A user requesting a
missing document is not an emergency; it is a normal branch.

## Name The Result, Not The Mechanism

Good Go names are often plain:

```go
FindByID
Save
Delete
Evaluate
Check
```

Avoid names that narrate implementation details:

```text
DoTupleMapLookupAndMaybeExpandRules
```

The evaluator's private helpers can be more specific because they describe
steps in an algorithm:

```go
hasRelation
hasTuple
subjectSetContains
expandByRules
```

Names should let a reader skim the outline before reading the body.

## Comments Explain Contracts And Surprises

Go comments are not a place to repeat the code:

```go
// bad: increments i
i++
```

Use comments to explain a contract, invariant, or non-obvious reason:

```go
// Return a copy so callers cannot mutate repository state.
func (v *Verifier) ScopesFor(token string) []string
```

Exported declarations should have comments that start with the declaration
name. That keeps `go doc` useful:

```go
// GraphEvaluator answers permission checks by walking the relationship graph.
type GraphEvaluator struct { ... }
```

## Concurrency Must Have An Exit Plan

Starting a goroutine is easy. Knowing when it stops is the work.

Before writing `go f()`, answer:

- What input does the goroutine need?
- Where does the result go?
- How does cancellation reach it?
- Who waits for it?
- Can it block forever on send, receive, lock, or I/O?

`examples/concurrency.AllPermissions` uses a buffered channel so workers can
finish even if the collector returns early after context cancellation.

`examples/concurrency.BulkCheck` uses `sync.WaitGroup.Go` because it wants to
wait for a known set of independent checks and preserve results by index.

Use concurrency when independent work is slow enough to justify coordination.
The in-memory evaluator is so small that concurrency is often slower. Measure.

## Protect Invariants, Not Lines Of Code

A mutex is not a magic thread-safety sticker. It protects a specific invariant.

For an in-memory store, the invariant is "the map must not be read while another
goroutine is mutating it." That is why reads use `RLock` and writes use `Lock`.

Keep critical sections small:

```go
s.mu.RLock()
defer s.mu.RUnlock()
item, ok := s.items[id]
return item, ok
```

Do not hold a lock while making network calls or calling unknown user-provided
code. You do not control how long that work takes.

## JSON Boundaries Are Contracts

At an HTTP boundary, sloppy input handling becomes user-visible behavior.

This repository's JSON code demonstrates common boundary choices:

- decode into explicit request structs
- set `Content-Type` before writing the response
- reject malformed JSON with a client error
- preserve domain errors for mapping to HTTP statuses
- use `httptest` to test handlers without a real port

For untrusted bodies, use `http.MaxBytesReader`. For strict APIs, use
`decoder.DisallowUnknownFields`. These are policy decisions, not random knobs.

## Tests Should Read Like Examples With Teeth

Good Go tests are boring in the best way:

```go
func TestGraphEvaluator_TeamMemberCanEditDocument(t *testing.T) {
    // arrange
    // act
    // assert
}
```

They should make the behavior visible. Table tests are excellent when the same
rule has many inputs. Separate tests are better when each case tells a different
story.

Use fakes when they make a package boundary easy to test. Avoid mock-heavy tests
that know every internal call. If a refactor breaks a test without changing
behavior, the test may be holding the code too tightly.

## Generics Should Earn Rent

Generics are useful when an operation is genuinely independent of the concrete
type:

```go
func Collect[T any](results []Result[T]) Result[[]T]
```

They are not a replacement for interfaces, and they are not a reason to wrap
simple `(value, error)` returns in production code.

Use generics for reusable data structures, algorithms, and helpers where the
compiler can preserve type information. Use interfaces for behavior.

```text
generic type parameter -> "same operation over many value types"
interface              -> "anything with this behavior"
```

## Standard Library First

Reach for the standard library before dependencies:

- `net/http` for servers and clients
- `encoding/json` for JSON
- `context` for cancellation and deadlines
- `errors` for error chains
- `sync` for locks and coordination
- `testing` and `httptest` for tests
- `slices`, `maps`, and `cmp` for common generic helpers

This repo uses OpenFGA's SDK where it needs an external service contract. Most
other machinery is standard library on purpose.

## Refactoring Recipe

When changing Go code in this repo:

1. Run the focused test first.
2. Make the smallest change that expresses the behavior.
3. Run `gofmt`.
4. Add or adjust tests at the package boundary.
5. Run the package tests.
6. Run the full quality loop before stopping.

Avoid refactoring unrelated packages while learning a concept. Clean code is
not the same thing as a wide diff.

## Checkpoint

You are ready to continue when you can explain:

- why the consumer often owns the interface
- why constructors usually return concrete types
- when a pointer receiver is appropriate
- why errors are wrapped instead of logged everywhere
- how a goroutine in your code stops
- why a generic helper may be worse than a plain function

Next: [Go Language Guide for This Repository](20-go-language-guide.md).
