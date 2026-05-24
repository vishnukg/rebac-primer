# Functions, modules, factories, and interfaces

TypeScript does not require a heavy object-oriented style. Good TypeScript often
looks like clear JavaScript with better boundaries.

## Scene

You are deciding where behavior belongs. Should everything be a class? Should
everything be a pure function? This repo uses a factory-based ports-and-adapters
style:

- functions for small transformations and typed id constructors
- modules for ownership
- interfaces for ports
- factories for stateful adapters and services

The `class` keyword does not appear anywhere in this repo — not even for errors.
Domain errors use a tagged factory pattern instead (covered below).

The guiding rule is:

```text
Use the smallest construct that makes the dependency boundary obvious.
```

## Functions should say what they need

From `src/core/ports/authz.ts`:

```ts
export const tuple = (
  objectId: RebacObject,
  relation: Relation,
  subject: Subject
): TupleKey => ({ object: objectId, relation, user: subject });
```

This function is intentionally boring. It takes three values and returns a
relationship tuple.

The useful part is the signature:

- `objectId` must be an OpenFGA-style object id
- `relation` must be a known relation
- `subject` must be an allowed subject
- the return value has the shape the authorizers expect

Readable code often comes from boring functions with precise types.

## Modules are boundaries

Each folder has a job:

```text
src/core/ports              interfaces and ReBAC value helpers
src/core/domain/documents   document use cases and domain errors
src/adapters/authn          token verification adapter
src/adapters/authz          graph and OpenFGA authorizer adapters
src/adapters/db             document repository adapter
src/adapters/http           HTTP request/response adapter
src/adapters/client         HTTP and terminal client adapters
src/server                  server entrypoint and composition root
src/cli                     terminal-client entrypoint and composition root
src/demo                    local graph demo entrypoint and composition root
src/demo/fixtures.ts        shared demo data
```

Good module boundaries answer this question:

> If I change this file, what part of the system am I changing?

If the answer is "a little bit of everything," the module is probably doing too
much.

## Interfaces describe behavior

From `src/core/ports/authz.ts`:

```ts
export interface Authorizer {
  check: (request: CheckRequest) => Promise<CheckResult>;
}
```

The document domain does not care whether authorization is handled by:

- the in-memory teaching evaluator
- the real OpenFGA SDK
- a fake implementation in a focused test

It only cares that the dependency can answer `check`.

## Dependency injection without a framework

From `src/core/domain/documents/makeDocuments.ts`:

```ts
const makeDocuments = ({ repository, authorizer }: DocumentsCfg): Documents => {
  const create = makeCreateDocument({ repository, authorizer });
  const read = makeReadDocument({ repository, authorizer });
  const update = makeUpdateDocument({ repository, authorizer });

  return { create, read, update };
};
```

This is dependency injection in plain TypeScript. No container is required. No
decorators are required. The factory simply receives the dependencies the domain
needs.

That keeps tests simple:

```ts
const tupleStore = makeInMemoryTupleStore({ seed: seedRelationshipTuples() });
const authorizer = makeGraphAuthorizer({ tupleStore });
const repository = makeInMemoryDocumentRepository();
const documents = makeDocuments({ repository, authorizer });
```

## Factories are enough for stateful adapters

The in-memory tuple store owns a private map, but it does not need a class:

```ts
const makeInMemoryTupleStore = ({ seed = [] } = {}): TupleStore => {
  const tuples = new Map<string, TupleKey>();

  const write = (tupleKey: TupleKey): void => {
    tuples.set(keyFor(tupleKey), tupleKey);
  };

  seed.forEach(write);

  return { has, findByObjectRelation };
};
```

The closure hides the map. The returned object exposes a small port:

- `has`
- `findByObjectRelation`

That gives the same encapsulation benefit people often reach for classes to get,
with less syntax for this tutorial.

## Tagged error factories — no class needed

A common reason to reach for a class is to create a custom error type that can
be caught precisely. The factory pattern handles this without `class`:

```ts
// A branded type: any Error whose `name` is "ForbiddenError".
export type ForbiddenError = Error & { readonly name: "ForbiddenError" };

// A factory function that builds one.
export const ForbiddenError = (message: string): ForbiddenError =>
    Object.assign(new Error(message), { name: "ForbiddenError" as const });

// A type guard for narrowing in catch blocks.
export const isForbiddenError = (e: unknown): e is ForbiddenError =>
    e instanceof Error && e.name === "ForbiddenError";
```

Usage at the call site is clean and identical to what a class would look like:

```ts
// throwing
throw ForbiddenError(`${actor} cannot edit ${id}`);

// catching precisely
if (isForbiddenError(error)) return json(403, { error: error.message });
```

The type guard works because TypeScript narrows `error.name` — no `instanceof`
dependency on the class constructor. Tests use `toMatchObject({ name: "ForbiddenError" })`
instead of `toBeInstanceOf(...)`.

## The ports-and-adapters direction

The dependency direction in this repo is:

```text
adapters -> core <- composition roots
```

Concrete examples:

- `makeCreateDocument` depends on `DocumentRepository` and `Authorizer`.
- `makeGraphAuthorizer` implements `Authorizer`.
- `makeOpenFgaAuthorizer` implements `Authorizer`.
- `makeInMemoryDocumentRepository` implements `DocumentRepository`.
- `makeServerApp` wires document services to the HTTP server.
- `makeCliApp` wires the terminal client to the HTTP client.
- HTTP handlers depend on `Documents`, not document implementation details.

The rule is simple:

```text
High-level policy should not import low-level infrastructure.
The composition root is where concrete choices are made.
```

## Entry points and composition roots

An entrypoint is the file Node runs directly:

```text
src/server/index.ts
src/cli/index.ts
src/demo/index.ts
```

These files should be boring. They should start the app, print output, listen on
a port, or close a terminal.

A composition root is the small module that builds the object graph for an
entrypoint:

```text
src/server/compose.ts -> document service plus HTTP server
src/cli/compose.ts    -> API client plus terminal client
src/demo/compose.ts   -> graph authorizer plus demo actors
```

The entrypoint performs the action. The composition root chooses concrete
implementations. The domain code still depends on ports.

## Import style

This repo separates runtime imports from type-only imports:

```ts
// From src/adapters/authz/makeGraphAuthorizer.ts
import { isObjectOfType, isSubjectSet, parseObject, parseSubjectSet } from "../../core/index.ts";
import type { Authorizer, CheckRequest, CheckResult, RebacObject, Relation } from "../../core/index.ts";
```

`import type` is a good habit. It tells TypeScript (and bundlers) that the
import is erased at runtime — no runtime cost, no circular-dependency risk.
Both lines import from the same barrel (`core/index.ts`), but the split makes
intent explicit: the first line brings in functions; the second brings in types.

## Exercise

Add a `delete` operation to the document domain.

Requirements:

1. the actor must have `can_delete` on the document
2. the repository should expose a delete method
3. `makeDocuments` should return the new operation
4. tests should cover owner allowed and viewer denied

Keep the shape consistent with `read` and `update`. The goal is making the new
behavior feel like it belongs.

## Checkpoint

Explain this design in one sentence:

```text
makeUpdateDocument has an Authorizer.
makeOpenFgaAuthorizer implements Authorizer.
makeUpdateDocument does not import makeOpenFgaAuthorizer.
```

Good answer: the domain depends on behavior, not infrastructure, so it stays
testable and easy to swap from the teaching graph to real OpenFGA.
