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
const create = makeCreateDocument({ repository, authzClient });

// Runtime — called once per HTTP POST /documents
await create({ id, title, body, workspace, actor });
```

This separates two very different concerns:

| Phase | When | Who calls it |
|---|---|---|
| Construction | Startup | `compose.ts` |
| Operation | Per request | HTTP handler, test |

## From class to factory

A class mixes construction and operation in the same `this`:

```ts
class CreateDocumentService {
    constructor(
        private repository: DocumentRepository,
        private authzClient: AuthzClient,
    ) {}

    async create(input: CreateDocumentInput) {
        // repository and authzClient accessed via this
    }
}
```

A factory separates them cleanly:

```ts
const makeCreateDocument = ({ repository, authzClient }: CreateDocumentCfg) => {
    const create: CreateDocumentFn = async input => {
        // repository and authzClient captured in closure — no this
    };
    return create;
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

Calling `makeCreateDocument({ repository, authzClient })` is **partial
application**: you fix the dependency arguments now so the returned function
only needs the runtime arguments later.

Strict currying (`fn(a)(b)(c)`) is the mathematical version. Partial
application via a config object is the practical version. This repo uses
partial application.

### The Reader monad

In typed functional languages (Haskell, F#), the formal equivalent is the
**Reader monad**: a computation that depends on a shared environment injected
once. Your factory is the same idea without the monad machinery.

```text
makeCreateDocument({ repository, authzClient })
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
const create = makeCreateDocument({ repository, authzClient });

// Bad — positional, caller must read the signature to know order
const create = makeCreateDocument(repository, authzClient);
```

**`make` functions always carry an explicit return type annotation.**
The annotation names the port or type the factory builds. This is the
clearest signal that the function returns one thing:

```ts
const makeGraphEvaluator   = ({ repository }: ...): Evaluator          => { ... }
const makeAuthzDomain      = ({ repository, evaluator }: ...): AuthzService    => { ... }
const makeAuthzHttpHandler = ({ authz }: ...): AuthzHttpHandler        => { ... }
const makeAuthzHttpServer  = ({ handler }: ...): Server                => { ... }
const makeCreateDocument   = ({ repository, authzClient }: ...): CreateDocumentFn => { ... }
```

The return statement may look like an object literal with multiple keys —
but those keys are the *methods of a single interface*, not independent
peers. The return type annotation is what makes that explicit:

```ts
// The object literal IS the AuthzService — the type annotation proves it.
const makeAuthzDomain = (...): AuthzService => {
    return { check, writeTuples, deleteTuples, listTuples };
};
```

Returning the value directly (no wrapper) means the variable at the call
site names the thing, and the function name already tells you what it makes:

```ts
const repository = makeInMemoryTupleRepository({ seed: seedTuples });
const evaluator  = makeGraphEvaluator({ repository });
const domain     = makeAuthzDomain({ repository, evaluator });
const handler    = makeAuthzHttpHandler({ authz: domain });
const server     = makeAuthzHttpServer({ handler });
```

**`compose` functions have no return type annotation** — because no named
type exists for the bag they return. That absence is itself a signal:
these functions wire together independent peers, not a single interface.
The returned bag exists purely for the caller's convenience:

```ts
// No return type annotation — the bag has no name.
const composeAuthzService = ({ port?, seedTuples? } = {}) => {
    // ...wiring...
    return { listen, domain };   // independent peers; callers pick what they need
};
```

The rule of thumb:

| Prefix | Has return type annotation? | Returns |
|---|---|---|
| `make*` | ✓ always | the thing itself — one named type |
| `compose*` | ✗ never | a bag of independent peers |

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
const create = makeCreateDocument({ repository, authzClient });

// HTTP handler
await create({ id, title, body, workspace, actor });
```

Good answer: `makeCreateDocument` captures the infrastructure dependencies
once at startup and returns the `create` function directly; `create` is then
called once per request with only the runtime data it needs. The closure
separates wiring from execution.

Also explain why the file is named `compose.ts` and the function is named
`composeAuthzService` rather than `makeAuthzService`:

Good answer: `make*` functions build one thing and return it directly —
they always have an explicit return type annotation naming the port or type
they produce. `compose*` functions wire multiple independent peers together,
return a named bag, and carry no return type annotation (because no single
named type covers the bag). The `compose.ts` file is the composition root —
the one place in the codebase that knows which concrete adapter goes behind
each port.

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
