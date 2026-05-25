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

From `src/shared/rebac.ts`:

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
src/shared/rebac.ts                      shared types and value constructors
src/authz-service/core/domain/           AuthZ domain: check, writeTuples, listTuples
src/authz-service/core/ports/            TupleRepository + Evaluator interfaces
src/authz-service/adapters/graph/        in-process graph traversal + permission model
src/authz-service/adapters/db/           in-memory tuple store
src/authz-service/adapters/http/         AuthZ HTTP handler and server
src/documents-service/core/domain/       document create/read/update + domain errors
src/documents-service/core/ports/        Authenticator + AuthzClient + DocumentRepository
src/documents-service/adapters/authn/    token verification adapter
src/documents-service/adapters/authz/    HTTP client calling the AuthZ service
src/documents-service/adapters/db/       in-memory document repository
src/documents-service/adapters/http/     Documents HTTP handler and server
src/documents-service/adapters/client/   HTTP and terminal client adapters
src/cli                                  terminal-client entrypoint and composition root
test/fixtures.ts                         shared demo actors, tokens, and seed tuples
```

Good module boundaries answer this question:

> If I change this file, what part of the system am I changing?

If the answer is "a little bit of everything," the module is probably doing too
much.

## Interfaces describe behavior

From `src/documents-service/core/ports/authzClient.ts`:

```ts
export interface AuthzClient {
    check:       (request: CheckRequest) => Promise<CheckResult>;
    writeTuples: (tuples: TupleKey[]) => Promise<void>;
}
```

The document domain does not care whether authorization is handled by:

- the real AuthZ service over HTTP (`makeAuthzServiceClient`)
- the in-process graph evaluator used in tests (`makeInProcessAuthzClient`)
- a hand-written stub returning fixed values in a focused test

It only cares that the dependency satisfies `AuthzClient`.

## Dependency injection without a framework

From `src/documents-service/core/domain/makeDocuments.ts`:

```ts
const makeDocuments = ({ repository, authzClient }: DocumentsCfg): Documents => ({
    create: makeCreateDocument({ repository, authzClient }),
    read:   makeReadDocument({ repository, authzClient }),
    update: makeUpdateDocument({ repository, authzClient }),
});
```

This is dependency injection in plain TypeScript. No container is required. No
decorators are required. The factory simply receives the dependencies the domain
needs.

That keeps tests simple:

```ts
const repository  = makeInMemoryDocumentRepository();
const authzClient = makeInProcessAuthzClient(seedPolicyTuples());
const documents   = makeDocuments({ repository, authzClient });
```

## Factories are enough for stateful adapters

The in-memory tuple repository owns a private map, but it does not need a class:

```ts
const makeInMemoryTupleRepository = (seed: TupleKey[] = []): TupleRepository => {
    const store = new Map<string, TupleKey>();

    const keyFor = (t: TupleKey): string => `${t.object}|${t.relation}|${t.user}`;
    const write  = (t: TupleKey): void   => { store.set(keyFor(t), t); };
    seed.forEach(write);

    return { has, findByObjectRelation, findAll, write, delete: deleteFn };
};
```

The closure hides the map. The returned object exposes a small port:

- `has`
- `findByObjectRelation`
- `findAll`
- `write`
- `delete`

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

- `makeCreateDocument` depends on `DocumentRepository` and `AuthzClient`.
- `makeGraphEvaluator` is the in-process AuthZ implementation.
- `makeAuthzServiceClient` is the HTTP AuthZ implementation.
- `makeInMemoryDocumentRepository` implements `DocumentRepository`.
- `makeDocumentsServer` wires document domain to the HTTP server.
- `composeCliApp` wires the terminal client to the HTTP client.
- HTTP handlers depend on `Documents`, not document implementation details.

The rule is simple:

```text
High-level policy should not import low-level infrastructure.
The composition root is where concrete choices are made.
```

## Entry points and composition roots

An entrypoint is the file Node runs directly:

```text
src/authz-service/index.ts      starts the AuthZ service (port 4100)
src/documents-service/index.ts  starts the Documents service (port 4000)
src/cli/index.ts                runs the terminal demo client
```

These files should be boring. They should start the app, print output, listen on
a port, or close a terminal.

A composition root is the small module that builds the object graph for an
entrypoint:

```text
src/authz-service/compose.ts    -> graph evaluator + HTTP server
src/documents-service/compose.ts -> authz client + document domain + HTTP server
src/cli/compose.ts              -> API client + terminal client
```

The entrypoint performs the action. The composition root chooses concrete
implementations. The domain code still depends on ports.

## Import style

This repo separates runtime imports from type-only imports:

```ts
// From src/authz-service/adapters/graph/makeGraphEvaluator.ts
import {
    isObjectOfType, isSubjectSet, parseObject, parseSubjectSet,
} from "../../../shared/rebac.ts";
import type {
    CheckRequest, CheckResult, RebacObject, Relation,
} from "../../../shared/rebac.ts";
```

`import type` is a good habit. It tells TypeScript (and bundlers) that the
import is erased at runtime — no runtime cost, no circular-dependency risk.
Both lines import from the same barrel (`shared/rebac.ts`), but the split makes
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
makeUpdateDocument depends on AuthzClient.
makeAuthzServiceClient implements AuthzClient.
makeUpdateDocument does not import makeAuthzServiceClient.
```

Good answer: the domain depends on behavior (the port), not infrastructure (the
adapter), so it stays testable and the backing AuthZ service can be swapped
without touching document logic.

For the formal name of this factory pattern, its FP connections (partial
application, Reader monad), and known trade-offs, see
[19-factory-function-pattern.md](./19-factory-function-pattern.md).
