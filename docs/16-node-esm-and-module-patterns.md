# Node ESM and module patterns

Modules are one of the most important parts of modern TypeScript backend code.

They answer practical questions:

- How does Node find and load files?
- Why do imports in this repo include file extensions?
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

With `"type": "module"`, Node treats JavaScript files in this package as ES
modules. The TypeScript source follows the same ESM rules.

## Why imports include `.ts`

This repo uses imports like:

```ts
import makeGraphAuthorizer from "./adapters/authz/makeGraphAuthorizer.ts";
```

The key rule is not the exact extension. The key rule is that ESM imports are
explicit. Node-style ESM does not guess relative file extensions the way
CommonJS historically did.

This repo executes TypeScript source directly in development and tests, and
`tsconfig.json` enables `allowImportingTsExtensions`. So this is correct here:

```ts
import { tuple } from "./core/index.ts";
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
  "moduleResolution": "NodeNext",
  "allowImportingTsExtensions": true,
  "noEmit": true
}
```

These options tell TypeScript to follow modern Node module rules while checking
source files without emitting JavaScript.

That means TypeScript checks imports the way Node will load them. This reduces
the annoying class of bugs where code type-checks but cannot be loaded by Node.

## How Node loads an ES module

When Node sees:

```ts
import makeGraphAuthorizer from "./adapters/authz/makeGraphAuthorizer.ts";
```

it roughly does this:

```text
1. resolve the module specifier to a file path
2. load that file's dependencies first
3. evaluate each module once
4. make exported bindings available to importers
```

The "once" part matters.

If three files import `src/adapters/authz/model.ts`, the module body is
evaluated once. All importers share the same module instance.

## Module evaluation order

Suppose you have this graph:

```text
demo/index.ts
  imports demo/compose.ts
    imports adapters/authz/makeGraphAuthorizer.ts
    imports core/ports/authz.ts
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
// app.ts
import { count, increment } from "./counter.js";

increment();
console.log(count); // 1
```

This is powerful, but mutable exported state can make code hard to reason about.
Prefer exporting functions, constants, classes, or factory functions.

## Default exports vs named exports

Default export:

```ts
export default function makeService() {}
```

Import:

```ts
import AnythingYouWant from "./service.js";
```

Named export:

```ts
export function makeService() {}
```

Import:

```ts
import { makeService } from "./service.js";
```

This repo uses named exports for shared core APIs and default exports for
single factory functions such as `makeGraphAuthorizer`.

Named exports are easier to search, easier to refactor, and harder to rename
accidentally at the import site.

## Type-only imports

TypeScript has imports that exist only for type checking:

```ts
import type { Authorizer, RebacObject, Relation } from "../core/index.ts";
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
// src/core/index.ts
export * from "./domain/documents/index.ts";
export * from "./ports/index.ts";
```

Then callers import from one place:

```ts
import { makeDocuments, user } from "../core/index.ts";
```

Barrels can be useful when they are small and intentional. This repo has a core
barrel for the public domain API. Adapters are imported directly so the concrete
infrastructure choice remains visible.

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
// src/demo/index.ts
const app = createDemoApp();

for (const actor of app.actors) {
  // print the demo authorization result
}
```

This would be a poor surprise inside `src/core/ports/authz.ts`:

```ts
startHttpServer();
```

Rule of thumb:

```text
library modules export capabilities
composition roots wire dependencies
entrypoints perform actions
```

The repo keeps the wiring in small composition roots. That gives each entrypoint
a clear job:

```text
src/demo/index.ts    -> create demo app, print demo checks
src/server/index.ts  -> create server app, listen on the configured port
src/cli/index.ts     -> create terminal client, run it, close the terminal
```

That keeps imports predictable. Importing a domain module does not start a
server. Running an entrypoint does.

## Module caching and singletons

Node caches modules after evaluating them.

That means this module creates one shared store per process:

```ts
// singleton-store.ts
import makeInMemoryTupleStore from "./makeInMemoryTupleStore.ts";

export const tupleStore = makeInMemoryTupleStore();
```

Every importer gets the same `tupleStore` instance.

```ts
import { tupleStore } from "./singleton-store.ts";
```

That is the simplest singleton pattern in Node ESM.

## Singleton pattern 1: exported instance

```ts
export const authorizer = makeGraphAuthorizer({
  tupleStore: makeInMemoryTupleStore()
});
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
let authorizer: Authorizer | undefined;

export function getAuthorizer(): Authorizer {
  authorizer ??= makeGraphAuthorizer({
    tupleStore: makeInMemoryTupleStore()
  });
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
const tupleStore = makeInMemoryTupleStore({ seed: seedRelationshipTuples() });
const authorizer = makeGraphAuthorizer({ tupleStore });
const repository = makeInMemoryDocumentRepository();
const documents = makeDocuments({ repository, authorizer });
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

## Composition roots

A composition root is the place where an application chooses concrete
implementations for its interfaces.

In this repo:

```text
server/compose.ts  -> domain services plus HTTP server
demo/compose.ts    -> demo authorizer and demo actors
cli/compose.ts     -> HTTP API client plus terminal client
```

Notice the direction:

```text
Documents depends on Authorizer
makeGraphAuthorizer implements Authorizer
server/compose.ts receives the chosen Authorizer
```

The domain does not know which authorizer was chosen. That is the point. The
composition root owns the decision, and the rest of the app talks through
interfaces.

This pattern also avoids accidental singletons. Instead of exporting global
document services, tests and entrypoints ask factories to create a fresh graph.

## Singleton pattern 4: dependency container

Larger apps sometimes build a small container:

```ts
export type AppServices = {
  authorizer: Authorizer;
  documents: Documents;
};

export function createServices(): AppServices {
  const tupleStore = makeInMemoryTupleStore({ seed: seedRelationshipTuples() });
  const authorizer = makeGraphAuthorizer({ tupleStore });
  const repository = makeInMemoryDocumentRepository();

  return {
    authorizer,
    documents: makeDocuments({ repository, authorizer })
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
Documents depends on Authorizer
makeOpenFgaAuthorizer implements Authorizer
core/domain/documents does not import makeOpenFgaAuthorizer
```

That is deliberate.

## Dynamic import

Most imports should be static:

```ts
import makeGraphAuthorizer from "./adapters/authz/makeGraphAuthorizer.ts";
```

Dynamic import loads a module at runtime:

```ts
const { default: makeGraphAuthorizer } = await import("./adapters/authz/makeGraphAuthorizer.ts");
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
- explicit file extensions for relative imports
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

Create a small composition module. Real composition often needs to seed data
through the domain service (so domain rules run), which makes the function
async. Try extracting the setup from `typescript/src/server/index.ts`:

```ts
export type AppServices = Readonly<{
  documents: Documents;
  authorizer: Authorizer;
  tupleStore: TupleStore;
}>;

export async function createServices(): Promise<AppServices> {
  const tupleStore = makeInMemoryTupleStore({ seed: seedRelationshipTuples() });
  const authorizer = makeGraphAuthorizer({ tupleStore });
  const repository = makeInMemoryDocumentRepository();
  const documents = makeDocuments({ repository, authorizer });

  // Seed the initial document through the service so its authz check runs.
  await documents.create({ /* ... */ });

  return { documents, authorizer, tupleStore };
}
```

Then update `src/server/index.ts` to `await createServices()` before passing the
services into `makeServerApp`.

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
open network connections. `src/server/index.ts` is allowed to perform actions
because it is an entrypoint.
