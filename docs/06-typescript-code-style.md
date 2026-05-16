# TypeScript code style

This repo values simplicity, readability, and maintainability over cleverness.

## Defaults

- Keep modules small and named after the domain concept they own.
- Prefer plain functions, classes, and interfaces before adding libraries.
- Use `type` aliases for data shapes and unions.
- Use `interface` for behavior contracts such as `Authorizer` and repositories.
- Keep `strict` TypeScript enabled.
- Let `npm run build` catch unused code and missing returns.

## Types

- Model domain language explicitly: `RebacObject`, `TupleKey`, `CheckRequest`.
- Use union types for closed vocabularies such as relations.
- Avoid `as` casts in application code. Prefer type guards such as
  `isObjectOfType`.
- Validate string parsing at module boundaries.

## Code shape

- Keep authorization checks near the business action they protect.
- Extract a helper only when it removes repeated logic without hiding intent.
- Prefer readable loops over dense functional chains when explaining graph
  traversal.
- Write tests that describe behavior, not implementation details.

## ReBAC-specific rule

Every permission should be explainable as a graph path:

```text
user -> relation -> object -> inherited relation -> permission
```

If you cannot explain the path in one or two lines, simplify the model before
adding more code.
