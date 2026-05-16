# Functions, modules, classes, and interfaces

TypeScript does not require a heavy object-oriented style. Good TypeScript often
looks like clear JavaScript with better boundaries.

## Scene

You are deciding where behavior belongs. Should everything be a class? Should
everything be a pure function? This repo chooses a middle path: object-oriented
boundaries for services and adapters, simple functions for small value work.

This repo uses:

- functions for small transformations and constructors
- modules for ownership
- interfaces for dependencies
- classes where state and behavior belong together

That is a hybrid style. It is not "functional programming" in the academic
sense, and it is not classic Java/C# style object orientation either. It is
plain TypeScript: classes for services, stores, and adapters; functions for
small value constructors and parsing helpers.

The guiding rule is:

```text
Use objects when behavior and state belong together.
Use functions when transforming values is the whole job.
```

## Functions should say what they need

From `src/authz/types.ts`:

```ts
export function tuple(objectId: RebacObject, relation: Relation, subject: Subject): TupleKey {
  return { object: objectId, relation, user: subject };
}
```

This function is intentionally boring. It takes three values and returns a tuple
key.

The useful part is the signature:

- `objectId` must be an OpenFGA-style object id
- `relation` must be a known relation
- `subject` must be an allowed subject
- the return value is immutable

Readable code often comes from boring functions with precise types.

## Modules are boundaries

Each folder has a job:

```text
src/authz    authorization model, tuple vocabulary, authorizer implementations
src/domain   document domain model and service logic
src/testing  shared fixtures for tests and demos
```

Good module boundaries answer this question:

> If I change this file, what part of the system am I changing?

If the answer is "a little bit of everything," the module is probably doing too
much.

This chapter focuses on modules as design boundaries. For the deeper Node
runtime model, including ESM loading, `.js` import extensions, module caching,
and singleton patterns, read `07-node-esm-and-module-patterns.md`.

## Interfaces describe behavior

From `src/authz/types.ts`:

```ts
export interface Authorizer {
  check(request: CheckRequest): Promise<CheckResult>;
}
```

This says the domain service does not care whether authorization is handled by:

- the in-memory teaching evaluator
- the real OpenFGA SDK
- a fake implementation in a focused test

It only cares that the dependency can answer `check`.

## Dependency injection without a framework

From `src/domain/service.ts`:

```ts
export class DocumentService {
  constructor(
    private readonly repository: DocumentRepository,
    private readonly authorizer: Authorizer
  ) {}
}
```

This is dependency injection in plain TypeScript.

No container is required. No decorators are required. The constructor simply
receives the dependencies the service needs.

That keeps tests simple:

```ts
const service = new DocumentService(
  new InMemoryDocumentRepository(),
  new GraphAuthorizer(store)
);
```

## Classes are useful when they own state

`MemoryTupleStore` is a class because it owns a private map of tuples:

```ts
export class MemoryTupleStore {
  private readonly tuples = new Map<string, TupleKey>();
}
```

The map should not be exposed directly. The class gives callers a small API:

- `write`
- `delete`
- `has`
- `findByObjectRelation`
- `all`

That is a good reason to use a class.

## Classes are useful at service boundaries

`DocumentService` is a class because it coordinates dependencies and exposes
business actions:

```ts
export class DocumentService {
  constructor(
    private readonly repository: DocumentRepository,
    private readonly authorizer: Authorizer
  ) {}

  async update(input: UpdateDocumentInput): Promise<CollaborativeDocument> {
    const existing = await this.requireDocument(input.id);
    await this.requireAllowed(input.actor, "can_edit", documentObject(input.id), "edit");

    const updated = { ...existing, body: input.body, updatedBy: input.actor };
    await this.repository.save(updated);
    return updated;
  }
}
```

This is a good object-oriented shape:

- dependencies are injected once in the constructor
- public methods map to business actions
- private methods hide repeated mechanics
- stateful collaborators are explicit
- tests can create a fresh service per test

`OpenFgaAuthorizer` follows the same pattern. It owns an SDK client and exposes
the smaller `Authorizer` interface to the rest of the app.

## Classes are not required for everything

These are plain functions:

```ts
user("alice");
team("platform");
workspace("acme");
document("roadmap");
```

They do not need classes because they do not own state. They are small
constructors for typed ids.

Do not turn every noun into a class. Use the simplest construct that expresses
the idea.

For example, these could become classes:

```ts
class UserId {}
class TeamId {}
class WorkspaceId {}
class DocumentId {}
```

That may be useful in a larger domain with rich validation and behavior. In this
primer, it would add ceremony without teaching much. Template literal types and
small constructor functions keep the ReBAC vocabulary visible with less code.

## What a more OO version would look like

A more object-oriented tuple model might look like this:

```ts
class TupleKey {
  constructor(
    readonly object: RebacObject,
    readonly relation: Relation,
    readonly user: Subject
  ) {}

  toString(): string {
    return `${this.object}|${this.relation}|${this.user}`;
  }
}
```

That is not wrong. It becomes attractive if tuples gain behavior:

- canonical serialization
- validation rules
- comparison methods
- conversion to SDK request types

But if the class only stores data, a `Readonly` object is usually simpler:

```ts
export type TupleKey = Readonly<{
  user: Subject;
  relation: Relation;
  object: RebacObject;
}>;
```

The maintainability question is not "class or function?" The question is
"where does the behavior live most clearly?"

## The style used in this repo

This repo leans object-oriented for application structure:

- `DocumentService` coordinates domain actions.
- `MemoryTupleStore` owns mutable tuple state.
- `GraphAuthorizer` owns graph traversal behavior over a store.
- `OpenFgaAuthorizer` adapts the OpenFGA SDK behind an interface.
- `InMemoryDocumentRepository` owns document persistence state.

It uses functions and type aliases for value-level vocabulary:

- `user("alice")`
- `workspace("acme")`
- `document("roadmap")`
- `tuple(...)`
- `parseObject(...)`
- `isObjectOfType(...)`

That split is deliberate. It keeps the architecture familiar to OO developers
without wrapping every small value in a class.

## My recommendation

For maintainable TypeScript backend code, prefer **object-oriented boundaries
with simple value types inside them**.

That means:

- services, repositories, and infrastructure adapters are classes
- behavior contracts are interfaces
- domain values are type aliases or small immutable objects until they need
  behavior
- pure parsing and construction helpers stay as functions
- avoid inheritance-first designs
- prefer composition over inheritance

Inheritance is rarely the first tool this repo should reach for. Most of the
time, an interface plus a class implementation is clearer:

```ts
interface Authorizer {
  check(request: CheckRequest): Promise<CheckResult>;
}

class OpenFgaAuthorizer implements Authorizer {}
class GraphAuthorizer implements Authorizer {}
```

That is object-oriented enough to give you polymorphism and encapsulation,
without creating a deep class hierarchy.

## Clean composition model

The dependency direction in this repo is:

```text
domain service -> interfaces
infrastructure -> implements interfaces
entrypoints    -> compose concrete objects
```

Concrete examples:

- `DocumentService` depends on `DocumentRepository` and `Authorizer`.
- `GraphAuthorizer` implements `Authorizer`.
- `OpenFgaAuthorizer` implements `Authorizer`.
- `InMemoryDocumentRepository` implements `DocumentRepository`.
- `createServices()` wires concrete implementations together.
- HTTP handlers depend on `DocumentWorkflow`, not `DocumentService` internals.

That separation keeps code testable:

```text
HTTP handler test -> uses DocumentWorkflow
service test      -> uses Authorizer interface
authorizer test   -> uses TupleReader interface
SDK adapter test  -> mocks SDK boundary
```

The rule is simple:

```text
High-level policy should not import low-level infrastructure.
The composition root is where concrete choices are made.
```

## Private methods can improve reading flow

`DocumentService` exposes public business actions:

- `create`
- `read`
- `update`

It hides repetitive implementation details:

```ts
private async requireAllowed(
  actor: RebacObject<"user">,
  relation: Relation,
  object: RebacObject,
  action: string
): Promise<void>
```

This helper is worth having because it removes repeated check/throw code while
keeping authorization visible in each public method.

If a helper makes callers harder to understand, do not extract it.

## Import style

This repo separates runtime imports from type-only imports:

```ts
import { document as documentObject } from "../authz/types.js";
import type { Authorizer, RebacObject, Relation } from "../authz/types.js";
```

`import type` is a good habit. It tells readers and tooling that the import is
used only by TypeScript and does not need to exist at runtime.

## Exercise

Add a `delete` method to `DocumentService`.

Requirements:

1. the actor must have `can_delete` on the document
2. the repository should expose a delete method
3. tests should cover owner allowed and viewer denied

Keep the shape consistent with `read` and `update`. The goal is not novelty; the
goal is making the new behavior feel like it belongs.

## Checkpoint

Explain this design in one sentence:

```text
DocumentService has an Authorizer.
OpenFgaAuthorizer implements Authorizer.
DocumentService does not import OpenFgaAuthorizer.
```

Good answer: the service depends on behavior, not infrastructure, so it stays
testable and easy to swap from the teaching graph to real OpenFGA.
