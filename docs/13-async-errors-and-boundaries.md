# Async, errors, and boundaries

Backend TypeScript is mostly about boundaries:

- HTTP boundaries
- database boundaries
- SDK boundaries
- authorization boundaries

Those boundaries are usually asynchronous and failure-prone. To write
maintainable backend TypeScript, you need to understand what `async` and `await`
actually do.

## Scene

The app asks OpenFGA a question. OpenFGA might answer quickly, answer slowly, or
fail. Meanwhile, Node should keep serving other requests. This chapter is about
writing that flow clearly without pretending the network is reliable.

## TypeScript does not invent async

Async behavior comes from JavaScript and the runtime, usually Node.

TypeScript adds types:

```ts
Promise<CheckResult>
```

But the runtime behavior is JavaScript:

```ts
const check: Authorizer["check"] = async request => {
  const response = await client.check({
    user: request.user,
    relation: request.relation,
    object: request.object
  });

  return {
    allowed: response.allowed === true,
    trace: ["OpenFGA evaluated the relationship graph remotely"]
  };
};
```

The type tells readers and the compiler what will eventually be produced. Node
does the actual scheduling.

## `Promise<T>`

A `Promise<T>` represents work that will eventually either:

- fulfill with a `T`
- reject with an error

Example:

```ts
Promise<CollaborativeDocument | undefined>
```

means:

```text
this async operation eventually returns a document, returns undefined, or fails
```

In this repo:

```ts
const findById = async (id: DocumentId): Promise<CollaborativeDocument | undefined> => {
  const found = store.get(id);
  return found ? { ...found } : undefined;
};
```

Even though the in-memory repository returns immediately, the interface is async
because a real repository would probably talk to a database.

That lets the service keep the same shape when storage changes.

## What `async` does

An `async` function always returns a promise.

This:

```ts
async function allowed(): Promise<boolean> {
  return true;
}
```

is conceptually like:

```ts
function allowed(): Promise<boolean> {
  return Promise.resolve(true);
}
```

If an `async` function throws, the returned promise rejects:

```ts
async function fail(): Promise<void> {
  throw new Error("nope");
}
```

Callers handle it with `await`, `.catch`, or a test assertion like
`rejects.toThrow`.

## What `await` does

`await` pauses the current async function until the promise settles.

It does not block the whole Node process.

This service method:

```ts
const update = async (input: UpdateDocumentInput): Promise<CollaborativeDocument> => {
  const existing = await repository.findById(input.id);
  if (!existing) throw new DocumentNotFoundError(input.id);

  const decision = await authorizer.check({
    user: input.actor,
    relation: "can_edit",
    object: document(input.id)
  });
  if (!decision.allowed) {
    throw new ForbiddenError(`${input.actor} cannot edit document:${input.id}`);
  }

  const updated = { ...existing, body: input.body, updatedBy: input.actor };
  await repository.save(updated);
  return updated;
};
```

reads like a checklist:

1. find the document
2. check authorization
3. save the updated document
4. return the result

Readable async code often looks synchronous, but each `await` is a scheduling
point where other work can continue.

## Event loop and microtasks

> **Optional deep dive.** The rest of this chapter is what you need for ReBAC
> work. The next two sections (event loop, concurrency vs parallelism) are
> background mechanics — useful, but skip to **Sequential awaits** if you want
> to keep moving and come back when an async bug forces you to.

Node runs JavaScript on an event loop.

For this primer, use this mental model:

```text
call stack:       currently running JavaScript
task queue:       timers, IO callbacks, incoming requests
microtask queue:  promise continuations
```

When an awaited promise resolves, the rest of the async function is scheduled as
a microtask.

Example:

```ts
console.log("A");

Promise.resolve().then(() => console.log("B"));

console.log("C");
```

Output:

```text
A
C
B
```

The promise callback runs after the current stack finishes.

You do not need to memorize every event-loop phase to write good backend code,
but you should know this: `await` yields control; it does not freeze the server.

## Concurrency vs parallelism

Concurrency means multiple tasks are in progress during the same period.

Parallelism means multiple tasks are literally executing at the same time.

Node async IO is usually concurrent, not parallel:

```text
request A waits for database
request B waits for OpenFGA
request C waits for disk
Node resumes each one when its promise settles
```

CPU-heavy JavaScript is different. A long CPU loop blocks the event loop.

Bad:

```ts
while (true) {
  calculateForever();
}
```

No request can progress while that loop owns the thread.

For this repo, most async work is IO-shaped: repository calls and OpenFGA SDK
calls.

## Sequential awaits

Sequential code is easiest to read when each step depends on the previous one.

```ts
const existing = await repository.findById(input.id);
const decision = await authorizer.check({
  user: input.actor,
  relation: "can_edit",
  object: document(input.id)
});
await repository.save(updated);
```

This order is intentional:

- you cannot update a missing document
- you should not save before authorization succeeds
- the returned value depends on the old document

Sequential awaits are not a problem when they represent real business order.

## Parallel awaits

If operations are independent, start them together:

```ts
const [document, decision] = await Promise.all([
  repository.findById(id),
  authorizer.check({ user: actor, relation: "can_read", object: document(id) })
]);
```

This can be faster because both operations run concurrently.

But do not parallelize blindly. In authorization code, order can be part of the
security model. For example, you may intentionally check whether a document
exists before deciding whether to reveal authorization details.

Default to clarity. Optimize when you know the operations are independent.

## Promise helpers

Common helpers:

```ts
await Promise.all([...])
```

All must succeed. Rejects as soon as one rejects.

```ts
await Promise.allSettled([...])
```

Waits for every promise and gives you fulfilled/rejected results.

```ts
await Promise.race([...])
```

Resolves or rejects with the first settled promise.

```ts
await Promise.any([...])
```

Resolves with the first fulfilled promise. Rejects only if all reject.

For most backend service code, `Promise.all` is the one you will use most.

## Error propagation

Inside an async function:

```ts
throw new ForbiddenError("not allowed");
```

is equivalent to returning a rejected promise.

Callers can use:

```ts
try {
  await service.update(input);
} catch (error) {
  // translate to HTTP response later
}
```

Tests can use:

```ts
await expect(service.update(input)).rejects.toBeInstanceOf(ForbiddenError);
```

Always `await` async expectations. If you forget, the test can pass before the
promise rejects.

## Domain errors

The domain layer defines its own errors:

```ts
export class DocumentNotFoundError extends Error {
  constructor(id: DocumentId) {
    super(`Document not found: ${id}`);
  }
}
```

```ts
export class ForbiddenError extends Error {
  constructor(message: string) {
    super(message);
  }
}
```

These are small, but useful. A future HTTP layer can map them cleanly:

- `DocumentNotFoundError` -> `404`
- `ForbiddenError` -> `403` or intentionally masked `404`

Do not throw generic strings. Throw errors that communicate intent.

## SDK adapters

Open `src/adapters/authz/makeOpenFgaAuthorizer.ts`.

The OpenFGA SDK is useful, but the rest of the app should not be covered in SDK
details. The adapter keeps those details in one place:

```ts
const check: Authorizer["check"] = async request => {
  const response = await client.check({
    user: request.user,
    relation: request.relation,
    object: request.object
  });

  return {
    allowed: response.allowed === true,
    trace: ["OpenFGA evaluated the relationship graph remotely"]
  };
};
```

The document domain depends on `Authorizer`, not `OpenFgaClient`.

That is the maintainable boundary.

## Why adapters matter

Without an adapter, service code tends to become noisy:

```ts
await openFgaClient.check({
  user,
  relation,
  object,
  authorization_model_id: process.env.FGA_MODEL_ID,
  store_id: process.env.FGA_STORE_ID
});
```

That mixes business rules with infrastructure details.

With an adapter, the service stays focused:

```ts
await authorizer.check({ user: actor, relation: "can_edit", object: document(id) });
```

The code now says what the business action needs.

## Error handling strategy

This repo follows a simple rule:

- domain code throws domain errors
- infrastructure adapters let unexpected SDK/network errors bubble up
- an application boundary can translate errors into HTTP responses later

Avoid swallowing errors too early. If you catch an error, either add context or
convert it into a meaningful domain/application error.

Bad:

```ts
try {
  await this.client.check(request);
} catch {
  return { allowed: false, trace: ["failed"] };
}
```

That turns an outage into an authorization decision without making the tradeoff
explicit.

Better:

```ts
try {
  return await this.client.check(request);
} catch (error) {
  throw new Error("OpenFGA check failed", { cause: error });
}
```

Now the application boundary can decide whether to fail closed, retry, or return
an operational error.

## Top-level await

ES modules support top-level `await`:

```ts
const config = await loadConfig();
```

Use it carefully. Top-level await delays module evaluation for every importer.

It is reasonable in an application entrypoint. It is risky in shared library
modules because importing the module can unexpectedly perform async work.

This repo keeps async work inside functions and methods.

## Async anti-patterns

### Forgetting `await`

```ts
service.update(input);
```

This starts the async operation but does not wait for it. In tests and service
code, that is usually a bug.

### Using `forEach` with async callbacks

```ts
items.forEach(async (item) => {
  await save(item);
});
```

`forEach` does not wait for the async callbacks.

Prefer sequential:

```ts
for (const item of items) {
  await save(item);
}
```

Or parallel:

```ts
await Promise.all(items.map((item) => save(item)));
```

Choose based on whether order matters.

### Catching too broadly

```ts
try {
  await service.update(input);
} catch {
  return undefined;
}
```

This hides important failures. Catch errors where you can do something useful
with them.

## Exercise

Write a small fake authorizer for a test:

```ts
const denyAllAuthorizer: Authorizer = {
  check: async () => ({ allowed: false, trace: ["test deny"] })
};
```

Use it to prove `documents.create` rejects unauthorized actors.

Then compare that test with the existing graph-based tests. Which one teaches
more about ReBAC? Which one isolates the service more tightly?

## Exercise: sequential vs parallel

Imagine a future `readMany(ids, actor)` method.

Ask:

1. Should document lookups happen sequentially or with `Promise.all`?
2. Should authorization checks happen before or after confirming documents
   exist?
3. What error should the method return if one document is forbidden?

There is no universal answer. The maintainable answer is the one that makes the
security behavior explicit.

## Checkpoint

Explain the difference:

```text
await pauses the current async function.
await does not block the whole Node process.
```

If that sentence makes sense, the event-loop model is starting to click.
