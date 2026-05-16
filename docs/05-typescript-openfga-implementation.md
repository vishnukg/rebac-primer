# TypeScript OpenFGA implementation

The application depends on this interface:

```ts
export interface Authorizer {
  check(request: CheckRequest): Promise<CheckResult>;
}
```

That interface has two implementations:

- `GraphAuthorizer`: local teaching evaluator for unit tests
- `OpenFgaAuthorizer`: real SDK adapter for OpenFGA

The service layer does not know which one it receives:

```ts
const decision = await authorizer.check({
  user: actor,
  relation: "can_edit",
  object: document("roadmap")
});
```

This is dependency injection in plain TypeScript. It keeps business logic
testable while preserving a production path to OpenFGA.

## Running OpenFGA locally

```bash
docker compose -f deployments/docker-compose.yml up -d
```

Then create a store, write the model from `src/authz/model.ts`, write tuples,
and point `OpenFgaAuthorizer` at the resulting store and model ids.
