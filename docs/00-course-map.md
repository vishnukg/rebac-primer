# Course map

Welcome to **TS ReBAC Primer**.

This repo is meant to be more than a code sample. It is a practical TypeScript
course wrapped around a real authorization problem: implementing
relationship-based access control with OpenFGA.

You are not reading a manual. You are taking over a small authorization system
and making it easier to understand, test, run, and evolve.

The learning loop is:

```text
read a concept -> inspect real code -> run tests -> change one thing -> explain what changed
```

If a topic does not help you read, write, test, or maintain this codebase, it is
kept out of the main path.

## How each chapter works

Each tutorial chapter is meant to feel like a guided lab:

- **Scene**: the practical problem the chapter is solving
- **Concept**: the TypeScript, ReBAC, Node, or Docker idea
- **Code walk**: where the idea appears in this repo
- **Try it**: a small change that gives you feedback
- **Checkpoint**: a quick question to prove the idea landed

If a chapter starts feeling abstract, jump to the code named in the table and
run the command beside it. The repo is the lesson.

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

Here is the story you will keep coming back to:

```text
The workspace editor edits the roadmap document because she is in the platform team.
The workspace viewer can read the roadmap document but cannot edit it.
The outside collaborator has no path through the graph, so access is denied.
```

That tiny cast of users keeps the examples grounded.

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
| 08 | OAuth/OIDC authentication fundamentals | conceptual |
| 09 | Authorization fundamentals: RBAC, ABAC, ReBAC | conceptual |

## Track 2: ReBAC with OpenFGA

Read these after docs 08, 09, and 10, or in parallel if authorization is your
main goal.

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 10 | Graph theory needed for ReBAC | conceptual |
| 11 | ReBAC concepts and relationship graphs | `src/authz/graph-authorizer.ts` |
| 12 | OpenFGA model DSL | `src/authz/model.ts` |
| 13 | TypeScript OpenFGA implementation | `src/authz/openfga-client.ts` |

## Track 3: Docker and local services

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 20 | Docker fundamentals: images, containers, Dockerfile | `deployments/Dockerfile` |
| 21 | Docker networking: host ports, service names, Compose DNS | `deployments/docker-compose.yml` |
| 22 | Docker Compose local services | `deployments/docker-compose.yml` |
| 23 | Client/server ReBAC demo with terminal client | `src/server.ts`, `src/client/tui.ts` |

## Track 4: Going to production

| Doc | Topic | Code to inspect |
|-----|-------|-----------------|
| 30 | Production readiness: what this repo does not cover | conceptual |

## Suggested pace

### Day 1: Make TypeScript feel less mysterious

1. Read `01-typescript-foundations.md`.
2. Run `npm run build`.
3. Break one type in `src/authz/types.ts`.
4. Read the compiler error carefully.
5. Restore the code and run `npm test`.

Checkpoint: explain why TypeScript catches `can_edti` before Node runs anything.

### Day 2: Learn the type system through the ReBAC vocabulary

1. Read the type aliases in `src/authz/types.ts`.
2. Read `02-types-and-values.md`.
3. Add a new permission name to `DocumentRelation`.
4. Watch which files need to change.

Checkpoint: explain why `Relation` is better than `string`.

### Day 3: Read the service layer like production code

1. Read `03-functions-modules-classes.md`.
2. Inspect `DocumentService`.
3. Trace how an update request becomes an authorization check.

Checkpoint: explain why `DocumentService` depends on `Authorizer`, not
`OpenFgaClient`.

### Day 4: Tests as executable documentation

1. Read `05-testing-with-vitest.md`.
2. Run `npm test`.
3. Change `seedRelationshipTuples()` and predict which tests fail.

Checkpoint: explain why the workspace viewer can read but cannot edit.

### Day 5: Understand Node modules

1. Read `07-node-esm-and-module-patterns.md`.
2. Inspect the `.js` extensions in TypeScript imports.
3. Explain why `src/main.ts` performs actions but `src/authz/types.ts` does not.

Checkpoint: explain why relative ESM imports use `.js` in TypeScript source.

### Day 6+: ReBAC and OpenFGA

1. Run `npm run dev` and read the authorization trace output.
2. Read `08-oauth-authentication.md`.
3. Read `09-authorization-fundamentals.md`.
4. Read `10-graph-theory-for-rebac.md`.
5. Read `11-rebac-concepts.md`.
6. Read `12-openfga-model.md`.
7. Run `npm run dev` again and trace each step against the model.

Checkpoint: draw the path from `user:workspaceEditor` to `document:roadmapDocument#can_edit`.

### Day 7+: Local services and client/server

1. Read `20-docker-fundamentals.md`.
2. Read `21-docker-networking.md`.
3. Read `22-docker-compose-local-services.md`.
4. Start the server with `npm run server`.
5. Run the terminal client with `npm run client`.

Checkpoint: explain what changes when the app runs on your host versus inside
Docker Compose.

### Day 8+: Production readiness

1. Read `30-production-readiness.md`.
2. For each gap listed, write one sentence describing where in this repo the
   production concern would be handled.

Checkpoint: explain why the `Authorizer` interface makes it straightforward to
swap `GraphAuthorizer` for a real OpenFGA client in a production deployment.

## Repo commands

```bash
npm install
npm run build
npm test
npm run dev
npm run server
npm run client
npm run coverage
```

Docker-backed Make targets:

```bash
make deps
make build
make test
make coverage
make check
make server
make client
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

## Keep it fun without making it fluffy

The entertaining part of this repo is the feedback loop:

- break a relation and watch the compiler object
- remove a tuple and watch access disappear
- run the terminal client as the workspace editor, the workspace viewer, then the outside collaborator
- start services locally and make the graph answer real HTTP requests

Every chapter should leave you with something you can run, break, or explain.
