# TypeScript foundations

TypeScript adds static checks to JavaScript. This repo uses strict mode because
authorization code is security-sensitive and should reject vague data shapes.

Start with `src/authz/types.ts`.

Key ideas:

- `type` aliases name domain concepts such as `TupleKey` and `CheckRequest`.
- string literal unions restrict values to known relations such as `can_edit`.
- template literal types model OpenFGA ids like `user:alice` and `document:roadmap`.
- functions like `user("alice")` keep object-id construction consistent.

Example:

```ts
const alice = user("alice");        // type: RebacObject<"user">
const roadmap = document("roadmap"); // type: RebacObject<"document">
```

The compiler now helps catch confused object types before tests run.
