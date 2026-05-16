# Course map

Welcome to TypeScript ReBAC Primer: one concrete project, two learning tracks.

The strategy is to learn TypeScript by building authorization code that has a
real reason to exist. Each doc points at code and tests you can run.

## Track 1: TypeScript foundations

| Doc | Topic | Code |
|-----|-------|------|
| 01 | TypeScript mental model, strict mode, literals | `src/authz/types.ts` |
| 02 | Objects, unions, template literal types | `src/authz/types.ts` |
| 03 | Interfaces, dependency injection, async services | `src/domain/service.ts` |
| 04 | Testing with Vitest | `test/*.test.ts` |

## Track 2: ReBAC with OpenFGA

| Doc | Topic | Code |
|-----|-------|------|
| 10 | Authorization fundamentals | conceptual |
| 11 | ReBAC concepts and relationship graphs | `src/authz/graph-authorizer.ts` |
| 12 | OpenFGA model DSL | `src/authz/model.ts` |
| 13 | TypeScript OpenFGA implementation | `src/authz/openfga-client.ts` |

## How to use this repo

1. Run `npm test`.
2. Read one doc.
3. Inspect the referenced code.
4. Change one tuple in `src/testing/fixtures.ts`.
5. Run the tests again and explain the changed graph path.

The tests are intentionally small. They are executable explanations of the
authorization model.
