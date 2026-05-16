# Node ESM and module patterns

Modules are one of the most important parts of modern TypeScript backend code.

They answer practical questions:

- How does Node find and load files?
- Why do imports in this repo end with `.js`?
- What is the difference between ESM and CommonJS?
- When is a module evaluated?
- How do singleton values work?
- When should you avoid singletons?

This doc gives you the working mental model.

## Scene

The app works in tests, but Node refuses to load a file because an import path is
wrong. Or a singleton leaks state between tests. Or importing a module starts a
server by accident. Module systems feel boring until they break your day.

This chapter helps you predict what Node will load, when it will run, and what
state will be shared.

## What a module is

A module is a file with its own scope.

In old browser JavaScript, top-level variables could accidentally become global.
In modern Node modules, top-level declarations stay inside the module unless you
export them.

```ts
const secret = "only this file can see me";

export const publicValue = "other files can import me";
```

That one rule is a huge maintainability win. It means each file can own a small
piece of the system without leaking everything.

## ESM vs CommonJS

Node has two major module systems.

CommonJS is the older Node module system:

```js
const fs = require("node:fs");

module.exports = {
  readConfig
};
```

ES modules are the modern JavaScript standard:

```ts
import { readFile } from "node:fs/promises";

export function readConfig() {}
```

This repo uses ESM.

You can see that in `package.json`:

```json
{
  "type": "module"
}
```

With `"type": "module"`, Node treats `.js` files in this package as ES modules.
Since TypeScript compiles `.ts` to `.js`, the emitted code runs as ESM.

## Why imports use `.js` in `.ts` files

This repo uses imports like:

```ts
import { MemoryTupleStore } from "./memory-store.js";
```

The source file is `memory-store.ts`, so why not import `./memory-store.ts`?

Because Node runs the compiled JavaScript, not the TypeScript source.

At runtime the file is:

```text
dist/src/authz/memory-store.js
```

Node ESM expects the runtime extension in relative imports. TypeScript with
`moduleResolution: "NodeNext"` understands this pattern and maps the `.js`
specifier back to the `.ts` source during type checking.

So this is correct in a Node ESM TypeScript project:

```ts
import { tuple } from "./types.js";
```

This is usually wrong for this repo:

```ts
import { tuple } from "./types";
```

Node ESM does not guess extensions the way CommonJS historically did.

## The relevant TypeScript settings

Open `tsconfig.json`.

```json
{
  "module": "NodeNext",
  "moduleResolution": "NodeNext"
}
```

These options tell TypeScript to follow modern Node module rules.

That means TypeScript checks imports the way Node will load them. This reduces
the annoying class of bugs where code type-checks but cannot be loaded by Node.

## How Node loads an ES module

When Node sees:

```ts
import { GraphAuthorizer } from "./authz/graph-authorizer.js";
```

it roughly does this:

```text
1. resolve the module specifier to a file path
2. load that file's dependencies first
3. evaluate each module once
4. make exported bindings available to importers
```

The "once" part matters.

If three files import `src/authz/model.ts`, the module body is evaluated once.
All importers share the same module instance.

## Module evaluation order

Suppose you have this graph:

```text
main.ts
  imports graph-authorizer.ts
    imports memory-store.ts
    imports types.ts
```

Node must evaluate dependencies before the importer can use them.

The practical rule:

```text
top-level code runs when the module is first imported
```

That is why top-level work should be cheap and unsurprising.

Good top-level code:

```ts
export const openFgaModel = `...`;
```

Risky top-level code:

```ts
const connection = await connectToProductionDatabase();
```

The first one defines data. The second one performs infrastructure work as a
side effect of importing a file.

## Live bindings

ESM exports are live bindings, not copied snapshots.

```ts
// counter.ts
export let count = 0;

export function increment(): void {
  count += 1;
}
```

```ts
// main.ts
import { count, increment } from "./counter.js";

increment();
console.log(count); // 1
```

This is powerful, but mutable exported state can make code hard to reason about.
Prefer exporting functions, constants, classes, or factory functions.

## Default exports vs named exports

Default export:

```ts
export default class DocumentService {}
```

Import:

```ts
import AnythingYouWant from "./service.js";
```

Named export:

```ts
export class DocumentService {}
```

Import:

```ts
import { DocumentService } from "./service.js";
```

This repo prefers named exports.

Named exports are easier to search, easier to refactor, and harder to rename
accidentally at the import site.

## Type-only imports

TypeScript has imports that exist only for type checking:

```ts
import type { Authorizer, RebacObject, Relation } from "../authz/types.js";
```

At runtime, this import disappears from the emitted JavaScript.

Use `import type` when:

- the imported name is only used as a type
- the module has runtime side effects you do not want to trigger accidentally
- you want readers to know the dependency is type-only

This is a small habit that keeps module dependencies honest.

## Barrel files

A barrel file re-exports from many files:

```ts
// src/authz/index.ts
export * from "./types.js";
export * from "./memory-store.js";
export * from "./graph-authorizer.js";
```

Then callers import from one place:

```ts
import { GraphAuthorizer, MemoryTupleStore } from "../authz/index.js";
```

Barrels can be useful in libraries. They can also hide dependencies and create
accidental import cycles.

This repo mostly avoids barrels because it is a teaching project. Direct imports
make ownership clearer.

## Module side effects

A module has a side effect when importing it changes the world:

```ts
console.log("loaded");
process.env.DEBUG = "true";
startServer();
```

Side effects are not always wrong, but they should be obvious.

This is fine for an application entrypoint:

```ts
// src/main.ts
const authorizer = new GraphAuthorizer(new MemoryTupleStore(tutorialTuples()));
```

This would be a poor surprise inside `src/authz/types.ts`:

```ts
startHttpServer();
```

Rule of thumb:

```text
library modules export capabilities
entrypoints perform actions
```

## Module caching and singletons

Node caches modules after evaluating them.

That means this module creates one shared store per process:

```ts
// singleton-store.ts
import { MemoryTupleStore } from "./memory-store.js";

export const tupleStore = new MemoryTupleStore();
```

Every importer gets the same `tupleStore` instance.

```ts
import { tupleStore } from "./singleton-store.js";
```

That is the simplest singleton pattern in Node ESM.

## Singleton pattern 1: exported instance

```ts
export const authorizer = new GraphAuthorizer(new MemoryTupleStore());
```

Pros:

- very simple
- no framework
- works because modules are cached

Cons:

- hard to reset in tests
- hidden shared mutable state
- configuration must be known at import time
- importing the module creates the instance whether you need it or not

Use this for stateless constants or process-wide infrastructure that is truly
global.

Avoid it for tutorial domain state.

## Singleton pattern 2: lazy getter

```ts
let authorizer: GraphAuthorizer | undefined;

export function getAuthorizer(): GraphAuthorizer {
  authorizer ??= new GraphAuthorizer(new MemoryTupleStore());
  return authorizer;
}
```

Pros:

- delays creation until first use
- can include runtime configuration
- can expose a test reset if you really need one

Cons:

- still global state
- callers cannot see dependencies clearly

This is useful for expensive infrastructure clients, but it should not become
the default for everything.

## Singleton pattern 3: explicit composition

This repo usually prefers explicit composition:

```ts
const store = new MemoryTupleStore(tutorialTuples());
const authorizer = new GraphAuthorizer(store);
const repository = new InMemoryDocumentRepository();
const service = new DocumentService(repository, authorizer);
```

Pros:

- dependencies are visible
- tests can create fresh instances
- no hidden shared state
- easy to understand

Cons:

- a little more wiring code

For learning, explicit composition is best. You see the object graph being
created, and tests stay independent.

## Singleton pattern 4: dependency container

Larger apps sometimes build a small container:

```ts
export type AppServices = {
  authorizer: Authorizer;
  documents: DocumentService;
};

export function createServices(): AppServices {
  const store = new MemoryTupleStore(tutorialTuples());
  const authorizer = new GraphAuthorizer(store);
  const repository = new InMemoryDocumentRepository();

  return {
    authorizer,
    documents: new DocumentService(repository, authorizer)
  };
}
```

This is still plain TypeScript. It gives the app one composition point without
using a dependency injection framework.

That is often enough.

## Import cycles

An import cycle happens when modules depend on each other:

```text
a.ts imports b.ts
b.ts imports a.ts
```

ESM can handle some cycles, but they are easy to make confusing.

Common causes:

- barrel files that re-export too much
- domain modules importing infrastructure modules
- utility modules becoming dumping grounds

Avoid cycles by keeping dependency direction clear:

```text
domain types       <- services use these
interfaces         <- services depend on these
infrastructure     -> implements interfaces
entrypoint         -> wires everything together
```

In this repo:

```text
DocumentService depends on Authorizer
OpenFgaAuthorizer implements Authorizer
DocumentService does not import OpenFgaAuthorizer
```

That is deliberate.

## Dynamic import

Most imports should be static:

```ts
import { GraphAuthorizer } from "./authz/graph-authorizer.js";
```

Dynamic import loads a module at runtime:

```ts
const { GraphAuthorizer } = await import("./authz/graph-authorizer.js");
```

Use dynamic import when:

- loading optional code
- delaying expensive startup work
- selecting a module based on runtime configuration

Do not use it just to avoid organizing dependencies.

## CommonJS interop

Many npm packages still publish CommonJS. TypeScript and Node can interoperate,
but the edges can be awkward.

This repo has:

```json
"esModuleInterop": true
```

That helps TypeScript generate friendlier code for CommonJS-style default
imports.

Still, prefer reading the package's docs and examples. Import shape depends on
how the package publishes its module formats.

## What this repo recommends

Use these defaults:

- ESM only
- named exports
- explicit `.js` extensions for relative imports
- `import type` for type-only dependencies
- direct imports instead of barrels
- no top-level infrastructure side effects
- explicit composition over singletons
- singleton exported constants only for immutable data

For example, this is good:

```ts
export const openFgaModel = `...`;
```

This is risky:

```ts
export const openFgaClient = new OpenFgaClient(loadConfigFromEnv());
```

The model string is immutable data. The client is environment-dependent
infrastructure.

## Exercise

Create a small composition module:

```ts
// src/app/create-services.ts
export function createServices() {
  const store = new MemoryTupleStore(tutorialTuples());
  const authorizer = new GraphAuthorizer(store);
  const repository = new InMemoryDocumentRepository();

  return {
    authorizer,
    documents: new DocumentService(repository, authorizer)
  };
}
```

Then update `src/main.ts` to use it.

After that, ask:

- Did the entrypoint become easier to read?
- Did any module start doing work just because it was imported?
- Can tests still create fresh services?

The right answer is the one that keeps dependencies visible.

## Checkpoint

Explain this rule:

```text
library modules export capabilities
entrypoints perform actions
```

Good answer: importing `types.ts` should define helpers, not start servers or
open network connections. `server.ts` is allowed to perform actions because it is
an entrypoint.
