# Course map

Welcome to **TS ReBAC Primer**.

This repo is meant to be more than a code sample. It is a practical TypeScript
course wrapped around a real authorization problem: implementing
relationship-based access control with OpenFGA.

The learning loop is:

```text
read a concept -> inspect real code -> run tests -> change one thing -> explain what changed
```

If a topic does not help you read, write, test, or maintain this codebase, it is
kept out of the main path.

## What you will build

The project domain is collaborative documents:

- users belong to teams
- teams get workspace access
- documents belong to workspaces
- document permissions are inherited through a relationship graph
- TypeScript types keep the authorization vocabulary explicit

The important idea is that TypeScript and ReBAC support each other. ReBAC gives
you a precise domain language. TypeScript lets you encode that language so
mistakes are caught while the code is still cheap to fix.

## Track 1: TypeScript primer

Read these first if you want this repo to be your TypeScript source of truth.

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 01 | TypeScript mental model, `strict`, project setup | `tsconfig.json`, `package.json` |
| 02 | Types, unions, narrowing, template literal types | `src/authz/types.ts` |
| 03 | Functions, modules, classes, interfaces | `src/domain/service.ts`, `src/domain/repository.ts` |
| 04 | Async TypeScript, errors, and service boundaries | `src/domain/service.ts`, `src/authz/openfga-client.ts` |
| 05 | Testing TypeScript with Vitest | `test/*.test.ts` |
| 06 | Coding style for maintainable TypeScript | `docs/06-typescript-code-style.md` |
| 07 | Node ESM, module loading, module patterns, singletons | `package.json`, `tsconfig.json`, `src/main.ts` |

## Track 2: ReBAC with OpenFGA

Read these after Track 1, or in parallel if authorization is your main goal.

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 10 | ReBAC concepts and relationship graphs | `src/authz/graph-authorizer.ts` |
| 11 | OpenFGA model DSL | `src/authz/model.ts` |
| 12 | TypeScript OpenFGA implementation | `src/authz/openfga-client.ts` |

## Suggested pace

### Day 1: Make TypeScript feel less mysterious

1. Read `01-typescript-foundations.md`.
2. Run `npm run build`.
3. Break one type in `src/authz/types.ts`.
4. Read the compiler error carefully.
5. Restore the code and run `npm test`.

### Day 2: Learn the type system through the ReBAC vocabulary

1. Read the type aliases in `src/authz/types.ts`.
2. Read `02-types-and-values.md`.
3. Add a new permission name to `DocumentRelation`.
4. Watch which files need to change.

### Day 3: Read the service layer like production code

1. Read `03-functions-modules-classes.md`.
2. Inspect `DocumentService`.
3. Trace how an update request becomes an authorization check.

### Day 4: Tests as executable documentation

1. Read `05-testing-with-vitest.md`.
2. Run `npm test`.
3. Change `tutorialTuples()` and predict which tests fail.

### Day 5: Understand Node modules

1. Read `07-node-esm-and-module-patterns.md`.
2. Inspect the `.js` extensions in TypeScript imports.
3. Explain why `src/main.ts` performs actions but `src/authz/types.ts` does not.

### Day 6+: ReBAC and OpenFGA

1. Read `10-rebac-concepts.md`.
2. Read `11-openfga-model.md`.
3. Run `npm run dev` and inspect the graph trace.

## Repo commands

```bash
npm install
npm run build
npm test
npm run dev
npm run coverage
```

## How to study this repo

Do not read passively. TypeScript becomes useful when you make the compiler do
work for you.

Good study moves:

- rename a relation and follow the compiler errors
- remove a tuple and predict authorization behavior
- add one test before changing implementation
- replace a broad type with a narrower union
- explain every permission as a graph path

Bad study moves:

- memorizing syntax without running code
- adding abstractions before the problem is visible
- treating `as` casts as a normal escape hatch
- testing mocks instead of behavior

The goal is not to write fancy TypeScript. The goal is to write TypeScript that
keeps important business rules obvious.
