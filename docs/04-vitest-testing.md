# Testing with Vitest

Authorization tests should be concrete. Avoid tests that only assert mocks were
called; they do not prove your graph grants and denies the right access.

This repo uses two test styles:

- `test/graph-authorizer.test.ts` tests ReBAC graph behavior directly.
- `test/document-service.test.ts` tests application behavior at the service
  boundary.

Run:

```bash
npm test
npm run coverage
```

Useful test shape:

```ts
const authorizer = new GraphAuthorizer(new MemoryTupleStore(tutorialTuples()));

await expect(
  authorizer.check({ user: alice, relation: "can_edit", object: roadmap })
).resolves.toMatchObject({ allowed: true });
```

The in-memory evaluator is not a replacement for OpenFGA. It is a teaching and
unit-testing tool. Integration tests can use `OpenFgaAuthorizer` against the
server from `deployments/docker-compose.yml`.
