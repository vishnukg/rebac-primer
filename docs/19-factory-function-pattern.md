# The factory function pattern

This repo uses factory functions everywhere. Understanding the pattern by name
makes it easier to read code you haven't seen before and to explain the
architecture to others.

## Scene

You keep seeing `make*` functions throughout the codebase. They all look the
same: take a config object, return an object with operations. This chapter
names that pattern, explains where it comes from, and shows why it is the
right fit for a ports-and-adapters authorization system.

## What it is called

The pattern has several names depending on context:

| Name | Emphasis |
|---|---|
| **Factory Function with Closure-Based DI** | most precise formal name |
| **Functional Dependency Injection** | emphasises the FP origin |
| **Module Factory Pattern** | emphasises the module boundary |
| **Closure-Based DI** | shorthand among practitioners |

It is a deliberate evolution of the **Revealing Module Pattern** — the same
closure idea, made reusable: call the factory many times to get independent
instances, each with its own captured dependencies.

The `class` keyword does not appear in this repo. That is not an accident.

## The two-call shape

Every factory in this repo has the same structure:

```text
make*(dependencies)  →  { operation }
                              ↓
                       operation(runtimeArgs)  →  result
```

The factory is called **once at startup** to capture dependencies in a
closure. The returned operation is called **once per request** at runtime.

```ts
// Startup — called once in compose.ts
const documents = makeDocuments({ repository, authzClient });

// Runtime — called once per HTTP POST /documents
await documents.create({ id, title, body, workspace, actor });
```

This separates two very different concerns:

| Phase | When | Who calls it |
|---|---|---|
| Construction | Startup | `compose.ts` |
| Operation | Per request | HTTP handler, test |

## From class to factory

A class mixes construction and operation in the same `this`:

```ts
class DocumentsService {
    constructor(
        private repository: DocumentRepository,
        private authzClient: AuthzClient,
    ) {}

    async create(input: CreateDocumentInput) {
        // repository and authzClient accessed via this
    }
    // read, update — same shape, same `this`
}
```

A factory separates them cleanly:

```ts
const makeDocuments = ({ repository, authzClient }: MakeDocumentsCfg): Documents => {
    const create: Documents["create"] = async input => {
        // repository and authzClient captured in closure — no this
    };
    // read, update defined the same way, inline
    return { create, read, update };
};
```

The factory wins on three counts in this codebase:

**No `this` binding bugs.** `this` in JavaScript is context-dependent.
Destructuring a method from a class instance can silently break it. Closures
never have this problem.

**Dependencies are visible at the boundary.** `CreateDocumentCfg` declares
exactly what the operation needs. Nothing can sneak in through a global or an
implicit import.

**Testing is honest.** Pass test doubles to the factory. No mocking
framework, no class instantiation. The factory is just a function call.

```ts
// In a test — wire it directly
const documents = makeDocuments({
    repository:  makeInMemoryDocumentRepository(),
    authzClient: makeInProcessAuthzClient(seedPolicyTuples()),
});

await documents.create({ id, title, body, workspace, actor });
```

## Connection to functional programming

### Partial application

Calling `makeDocuments({ repository, authzClient })` is **partial
application**: you fix the dependency arguments now so the returned operations
only need the runtime arguments later.

Strict currying (`fn(a)(b)(c)`) is the mathematical version. Partial
application via a config object is the practical version. This repo uses
partial application.

### The Reader monad

In typed functional languages (Haskell, F#), the formal equivalent is the
**Reader monad**: a computation that depends on a shared environment injected
once. Your factory is the same idea without the monad machinery.

```text
makeDocuments({ repository, authzClient })
  ≈ Reader.ask(env => useEnv(env))
```

The environment is injected once; the computation sees it from then on.

### Mark Seemann — dependency rejection

Mark Seemann (author of *Dependency Injection Principles, Practices, and
Patterns*) argues that functional code does not need DI the way OOP code
does. In OOP, DI frameworks exist to work around hidden state. In FP,
dependencies are explicit function arguments — the type signature is the
contract. He calls this **dependency rejection**: the function rejects the
idea of hidden state and requires every dependency to be passed explicitly.

This repo is that idea applied to TypeScript and ReBAC.

## The naming convention

**Inputs** always use a named config object:

```ts
// Good — names visible at the call site
const documents = makeDocuments({ repository, authzClient });

// Bad — positional, caller must read the signature to know order
const documents = makeDocuments(repository, authzClient);
```

**`make*` functions return one port, with their operations defined inline.**
A `make*` receives its dependencies ready-made and returns a single port — a
function, or an object whose keys are the *methods of one interface*. Crucially,
it does **not** build those operations from other factories; it defines them
inline. The return type annotation names the port:

```ts
const makeGraphEvaluator   = ({ repository }: ...): Evaluator      => { ... }
const makeAuthzHttpHandler = ({ authz }: ...): AuthzHttpHandler    => { ... }

// A multi-method port is still a make* — as long as every method is inline.
// Both domains are exactly this: one factory, one noun, operations as methods.
const makeAuthzService = ({ repository, evaluator }: ...): AuthzService => {
    // check / writeTuples / deleteTuples / listTuples defined inline — calls no factory
    return { check, writeTuples, deleteTuples, listTuples };
};
const makeDocuments = ({ repository, authzClient }: ...): Documents => {
    // create / read / update defined inline — calls no factory
    return { create, read, update };
};
const makeInMemoryTupleRepository = ({ seed }: ...): TupleRepository => {
    // write / delete / findAll defined inline over a private Map — calls no factory
    return { write, delete: del, findAll };
};
```

#### The domain is one module — operations are its methods

`makeAuthzService` defines all four operations (`check`, `writeTuples`,
`deleteTuples`, `listTuples`) in **one** closure; `makeDocuments` defines its
three (`create`, `read`, `update`) the same way — rather than a separate `make*`
factory per operation that a `compose*` then re-bundles. That is deliberate:

- **One factory, one noun.** The domain _is_ the `AuthzService` (or `Documents`).
  Its operations are verbs that belong to it, so they live inside as methods — not
  as four standalone nouns that have to be reassembled.
- **One rule to apply.** "Build the noun; verbs are its methods." There is no
  second decision about whether each operation gets its own file and factory.
- **The seam is already there.** When an operation needs more — caching on
  `check`, paging on `listTuples` — you add it _inside_ the factory; every call
  site is untouched because they only ever called `authz.check(...)`.

The trade-off is honest: you can no longer build a single operation in isolation
(a hypothetical `makeCheck(...)`) — you build the whole service and call the one
method. Tests do exactly that (`const { check } = makeAuthzService(...)`). The alternative — one
`make*` per operation assembled by a `compose*` — is equally valid; this repo
chose the single-module form for simplicity, matching `makeRestaurant` in the
ModulePattern reference repo.

**`compose*` functions build their own collaborators and wire them together.**
A `compose*` calls `make*` factories (and may select a concrete adapter from the
environment), then assembles the results. *That* — building your own
collaborators — is the deciding difference, not the return shape. A `compose*`
may return either of two shapes:

- **A single named port**, when its only job is to select an adapter and/or
  build one port from smaller factories. `composeAuthzBackend` picks the backend
  from the environment and returns an `AuthzService` either way:

  ```ts
  const composeAuthzBackend = (seedTuples: TupleKey[]): AuthzService => {
      if (process.env.AUTHZ_BACKEND === "openfga") {
          return makeOpenFgaAuthzService({ apiUrl, storeId, modelId });
      }
      const repository = makeInMemoryTupleRepository({ seed: seedTuples });
      const evaluator  = makeGraphEvaluator({ repository });
      return makeAuthzService({ repository, evaluator }); // one named port — BUILT from make* → compose*
  };
  ```

- **A named bag of peers**, *if* an entry point genuinely drives more than one
  capability. In this repo none do — each service root returns the single thing
  its `index.ts` runs (`composeAuthzService` → `{ listen }`,
  `composeDocumentsService` → `{ listen }`, `composeCliApp` → `{ run }`):

  ```ts
  const composeDocumentsService = ({ port?, authzUrl?, tokens?, seedDocuments? } = {}) => {
      const authzClient   = makeAuthzServiceClient({ baseUrl: authzUrl });
      const authenticator = makeDemoTokenVerifier({ tokens });
      const repository    = makeInMemoryDocumentRepository();
      const documents     = makeDocuments({ repository, authzClient });
      const handler = makeDocumentsHttpHandler({ authenticator, documents });
      const server  = makeDocumentsHttpServer({ handler });
      // seedDocuments are created inside listen() at startup, so the domain is
      // never handed back out — return only what the entry point drives.
      return { listen };
  };
  ```

  **Return only what the entry point actually drives — never expose the domain
  for a startup side-task or a test.** Startup data comes *in* as config
  (`seedTuples`, `seedDocuments`) and the root seeds it internally. This is a
  deliberate decision; see [ADR 0001](./adr/0001-composition-roots-return-only-what-is-driven.md).

### Which one am I writing? The one-question test

> _Does the function build its own collaborators — call `make*` factories (or
> select a concrete adapter) and wire them together?_
>
> - **No** — it receives its deps ready-made and returns one port with its
>   operations defined inline → it is a **`make*`** (`makeAuthzService`,
>   `makeDocuments`, `makeGraphEvaluator`, `makeAuthzHttpServer`).
> - **Yes** — it assembles pieces built elsewhere → it is a **`compose*`**
>   (`composeAuthzBackend`, `composeAuthzService`, `composeDocumentsService`,
>   `composeCliApp`).

The deciding factor is **building your own collaborators**, *not* the return
type. `makeAuthzService` defines all four operations inline (a `make*`);
`composeAuthzBackend` selects a backend and calls `makeInMemoryTupleRepository` /
`makeGraphEvaluator` / `makeAuthzService` (or `makeOpenFgaAuthzService`), handing
back an `AuthzService` (a `compose*`) — both produce the same port, but only one
builds its parts. This is the same rule the ModulePattern reference repo uses to
separate `makeRestaurant` (the domain, operations inline) from `composeServerApp`
(builds the restaurant, router, and server, then wires them).

#### This repo, function by function

| Function | Kind | Why |
| --- | --- | --- |
| `makeAuthzService` | `make*` | the authz domain — `check`/`writeTuples`/`deleteTuples`/`listTuples` defined inline |
| `makeDocuments` | `make*` | the documents domain — `create`/`read`/`update` defined inline |
| `makeGraphEvaluator`, `makeAuthzHttpHandler`/`Server`, `makeInMemory*`, … | `make*` | define their behaviour inline |
| `composeAuthzBackend` | `compose*` | selects the backend, calls `makeInMemoryTupleRepository` / `makeGraphEvaluator` / `makeAuthzService` (or `makeOpenFgaAuthzService`) |
| `composeAuthzService`, `composeDocumentsService`, `composeCliApp` | `compose*` | call the above + adapters; return only what the entry point drives (`{ listen }`, `{ listen }`, `{ run }`) |

#### A third kind: plain functions

Not every function is a factory. A function that takes data and returns data — a
transform, a formatter, a validator (e.g. `readPort`, `toErrorResponse`) — is
just an ordinary function. Don't give it a `make*` / `compose*` prefix; those are
only for the wiring layer (building and connecting ports).

> Helpers shared between composition roots (such as `readPort`, used by both the
> authz and documents services) live in `src/shared/` so the parsing logic is
> defined once.

## Where this lives in the repo

```text
core/ports/      ← interfaces (what each factory depends on)
core/domain/     ← business operations (pure domain logic)
adapters/        ← concrete implementations (db, http, graph)
compose.ts       ← the one place all factories are wired
index.ts         ← starts the process, calls listen()
```

`compose.ts` is the **composition root** — the only file that knows which
concrete adapters are used. The domain code depends on ports. Adapters
implement ports. The factory pattern is what makes the wiring possible
without a framework.

## Trade-offs

**Not a singleton by default.** Each call to a factory creates a new
closure. If you need one shared instance, call the factory once in
`compose.ts` and pass the result around — which is exactly what this repo
does.

**No magic wiring.** Unlike NestJS or InversifyJS there is no container
scanning decorators. You wire manually in `compose.ts`. More verbose in a
large codebase, but completely transparent. You can always trace every
dependency by reading the composition root.

**Closure memory.** Each factory call allocates a new closure scope. For
service-level objects created once at startup this is irrelevant. For
short-lived objects created thousands of times per second, prefer plain
functions with explicit arguments.

## Checkpoint

Explain this in one sentence:

```ts
// compose.ts
const documents = makeDocuments({ repository, authzClient });

// HTTP handler
await documents.create({ id, title, body, workspace, actor });
```

Good answer: `makeDocuments` captures the infrastructure dependencies once at
startup and returns the `Documents` port, whose `create`/`read`/`update` methods
are defined inline in the closure; `documents.create` is then called once per
request with only the runtime data it needs. The closure separates wiring from
execution.

Also explain why the authz service has **both** a `makeAuthzService` and a
`composeAuthzService`:

Good answer: `makeAuthzService` is a `make*` — the **domain**. It receives its
ports (`repository`, `evaluator`) ready-made, defines all four operations
(`check`/`writeTuples`/`deleteTuples`/`listTuples`) inline, and calls no other
factory. `composeAuthzService` is a `compose*` — the **composition root**. It
**builds its collaborators by calling other factories** (`composeAuthzBackend`,
which itself builds `makeAuthzService`; `makeAuthzHttpHandler`;
`makeAuthzHttpServer`) and wires them together, returning only `{ listen }`.
Calling other factories — not the shape of what it returns — is what makes it a
compose. The `compose.ts` file is the composition root: the one place in the
codebase that knows which concrete adapter goes behind each port.

## Further reading

**Factory functions and the Module Pattern**
- [From the Module Pattern to Factory Functions](https://medium.com/programming-essentials/from-the-module-pattern-to-factory-functions-a741cfbe818e) — Cristian Salcescu. Traces the evolution from IIFE → Revealing Module → reusable factory.
- [Factory Functions and the Module Pattern](https://www.theodinproject.com/lessons/node-path-javascript-factory-functions-and-the-module-pattern) — The Odin Project. Practical walkthrough with closure examples.
- [Factory functions](https://medium.com/@_ericelliott/factory-functions-b50d041bb023) — Eric Elliott. The primary advocate for replacing classes with factory functions in JavaScript.

**Functional Dependency Injection**
- [Functional Dependency Injection in TypeScript](https://hassannteifeh.medium.com/functional-dependency-injection-in-typescript-4c2739326f57) — Hassan Nteifeh. Walks through the exact pattern this repo uses.
- [TypeScript FP Dependency Injection Is Easy!](https://dev.to/tareksalem/typescript-fp-dependency-injection-is-easy-18pn) — DEV Community.
- [Dependency Injection in TypeScript](https://mateuszsuchon.com/articles/dependency-injection-in-typescript) — Mateusz Suchoń. Contrasts functional and OOP approaches.
- [7 Ways to do Dependency Injection in Functional JavaScript](https://happy-css.com/articles/dependency-injection-in-java-script/) — Comprehensive comparison of DI styles in JS.
- [Dependency Injection, Currying and Partial Application](https://medium.com/@curtistatewilkinson/dependency-injection-currying-and-partial-application-for-easy-unit-tests-ded40c39016c) — Curtis Tate Wilkinson.

**The Reader Monad connection**
- [Dependency Injection and Reader Monad](https://dev.to/napicella/dependency-injection-and-reader-monad-5ap4) — DEV Community. Shows how factory functions are a practical Reader monad.
- [Purely functional dependency injection in TypeScript](https://anttih.com/articles/2018/07/05/purely-functional-di) — Antti Holvikari. Deep dive into the FP underpinnings.

**Mark Seemann — Dependency Rejection**
- [From Dependency Injection to Dependency Rejection](https://www.youtube.com/watch?v=cxs7oLGrxQ4) — Talk arguing that FP makes DI containers unnecessary. Highly recommended.

**Ports and Adapters (Hexagonal Architecture)**
- [Hexagonal Architecture](https://jmgarridopaz.github.io/content/hexagonalarchitecture.html) — Juan Manuel Garrido de Paz. The original pattern this wiring style implements.
- [Ports and Adapters Architecture](https://medium.com/the-software-architecture-chronicles/ports-adapters-architecture-d19f2d476eca) — Herberto Graça.
