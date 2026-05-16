# TypeScript OpenFGA implementation

The application does not talk to OpenFGA directly from the domain service.

Instead, it depends on a small interface:

```ts
export interface Authorizer {
  check(request: CheckRequest): Promise<CheckResult>;
}
```

That one interface is the boundary between business logic and authorization
infrastructure.

## Scene

The service needs one answer: allowed or denied. OpenFGA has stores, model ids,
SDK request shapes, network errors, and tuple writes. The adapter keeps those
details from spilling into the domain.

## Why the interface matters

`DocumentService` should not know about:

- OpenFGA API URLs
- store ids
- authorization model ids
- SDK response shapes
- HTTP retries

It should know the business rule:

```text
to update a document, the actor must have can_edit on that document
```

That rule appears in code as:

```ts
await this.requireAllowed(input.actor, "can_edit", documentObject(input.id), "edit");
```

The implementation behind `Authorizer` can change without rewriting the service.

## Two implementations

This repo has two implementations:

```text
GraphAuthorizer   -> local evaluator for learning and unit tests
OpenFgaAuthorizer -> SDK adapter for real OpenFGA
```

They share the same interface.

That is why the domain code is easy to test and still has a production path.

## The teaching implementation

`GraphAuthorizer` is not trying to be OpenFGA.

It is a readable evaluator for this repo's model. It exists so you can see the
graph traversal in plain TypeScript and run tests without infrastructure.

This is the useful part:

```ts
const result = await authorizer.check({
  user: alice,
  relation: "can_edit",
  object: roadmap
});

console.log(result.trace);
```

The trace explains why access was allowed or denied.

## The real OpenFGA adapter

Open `src/authz/openfga-client.ts`.

```ts
export class OpenFgaAuthorizer implements Authorizer {
  private readonly client: OpenFgaClient;

  constructor(config: OpenFgaConfig) {
    this.client = new OpenFgaClient(config);
  }
}
```

The adapter owns the SDK client. The rest of the app does not.

The `check` method converts from this repo's request shape:

```ts
type CheckRequest = {
  user: RebacObject<"user">;
  relation: Relation;
  object: RebacObject;
};
```

to the SDK call:

```ts
await this.client.check({
  user: request.user,
  relation: request.relation,
  object: request.object
});
```

Then it converts the SDK response back into this repo's `CheckResult`.

That conversion is the adapter pattern in its simplest form.

## Writing tuples

The adapter also has:

```ts
async writeTuples(tuples: readonly TupleKey[]): Promise<void>
```

The app can write relationship facts without leaking SDK tuple shapes
everywhere.

Example tuple:

```ts
tuple(workspace("acme"), "editor", subjectSet(team("platform"), "member"))
```

This becomes an OpenFGA write request.

## Running OpenFGA locally

Start OpenFGA:

```bash
docker compose -f deployments/docker-compose.yml up -d
```

You then need to:

1. create a store
2. write the model from `src/authz/model.ts`
3. write tuples
4. configure `OpenFgaAuthorizer` with `apiUrl`, `storeId`, and optionally
   `authorizationModelId`

This repo does not hide those concepts because learning them is part of the
point.

## TypeScript lesson

The OpenFGA SDK is an external dependency. External dependencies should have a
small contact surface with your domain code.

That gives you:

- easier tests
- less churn when SDK types change
- clearer business logic
- one place for infrastructure error handling later

This is a general TypeScript backend habit, not only an OpenFGA habit.

## Exercise

Add a `deleteTuples` method to `OpenFgaAuthorizer`.

Guidelines:

1. accept `readonly TupleKey[]`
2. map from repo tuple shape to SDK tuple shape inside the adapter
3. do not expose SDK-specific types in `DocumentService`
4. add a focused unit test around the tuple mapping if you introduce a helper

## Checkpoint

Why does `DocumentService` depend on `Authorizer` instead of `OpenFgaClient`?

Good answer: the service owns business rules; the adapter owns SDK details.
