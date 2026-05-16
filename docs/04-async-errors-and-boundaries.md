# Async, errors, and boundaries

Backend TypeScript is mostly about boundaries:

- HTTP boundaries
- database boundaries
- SDK boundaries
- authorization boundaries

Those boundaries are usually asynchronous and failure-prone. This repo keeps
the async model small so the important ideas stay visible.

## `Promise<T>`

An async function returns a `Promise`.

```ts
async findById(id: DocumentId): Promise<CollaborativeDocument | undefined> {
  return this.documents.get(id);
}
```

Even though the in-memory repository returns immediately, the interface is async
because a real repository would probably talk to a database.

That lets the service keep the same shape when the storage implementation
changes.

## Await at the boundary

From `DocumentService`:

```ts
const existing = await this.requireDocument(input.id);
await this.requireAllowed(input.actor, "can_edit", documentObject(input.id), "edit");
```

This reads like a checklist:

1. find the document
2. check permission
3. update state

Readable async code often looks like synchronous code with explicit `await`
points.

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

Open `src/authz/openfga-client.ts`.

The OpenFGA SDK is useful, but the rest of the app should not be covered in SDK
details. The adapter keeps those details in one place:

```ts
export class OpenFgaAuthorizer implements Authorizer {
  async check(request: CheckRequest): Promise<CheckResult> {
    const response = await this.client.check({
      user: request.user,
      relation: request.relation,
      object: request.object
    });

    return {
      allowed: response.allowed === true,
      trace: ["OpenFGA evaluated the relationship graph remotely"]
    };
  }
}
```

The service depends on `Authorizer`, not `OpenFgaClient`.

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
await this.requireAllowed(actor, "can_edit", documentObject(id), "edit");
```

The code now says what the business action needs.

## Error handling strategy

This repo follows a simple rule:

- domain code throws domain errors
- infrastructure adapters let unexpected SDK/network errors bubble up
- an application boundary can translate errors into HTTP responses later

Avoid swallowing errors too early. If you catch an error, either add context or
convert it into a meaningful domain/application error.

## Exercise

Write a small fake authorizer for a test:

```ts
class DenyAllAuthorizer implements Authorizer {
  async check(): Promise<CheckResult> {
    return { allowed: false, trace: ["test deny"] };
  }
}
```

Use it to prove `DocumentService.create` rejects unauthorized actors.

Then compare that test with the existing graph-based tests. Which one teaches
more about ReBAC? Which one isolates the service more tightly?
