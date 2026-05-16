# TS ReBAC Primer: learn TypeScript by building OpenFGA authorization

This repository is a TypeScript course for programmers who want to learn two
things together:

- practical TypeScript for backend services
- relationship-based access control (ReBAC) with OpenFGA

The project domain is a collaborative document workspace. Workspaces contain
documents, teams get workspace access, and users inherit permissions through a
relationship graph. That small domain is enough to teach TypeScript types,
interfaces, async code, testing, and the core OpenFGA mental model.

The docs are designed to be read as a TypeScript primer, not as thin notes
beside the code. Start with the course map and work through the TypeScript track
before going deep on ReBAC.

## Start here

1. Read [docs/00-course-map.md](docs/00-course-map.md).
2. Install dependencies: `npm install`
3. Run tests: `npm test`
4. Run the tutorial demo: `npm run dev`
5. Read docs in order while jumping into the referenced code.
6. Keep new code aligned with [docs/06-typescript-code-style.md](docs/06-typescript-code-style.md).

## Repository map

- `src/domain`: TypeScript domain model and service layer
- `src/authz/model.ts`: OpenFGA authorization model DSL
- `src/authz/types.ts`: strongly typed tuple, object, and relation helpers
- `src/authz/memory-store.ts`: in-memory tuple graph used by unit tests
- `src/authz/graph-authorizer.ts`: small evaluator that explains graph traversal
- `src/authz/openfga-client.ts`: real OpenFGA SDK adapter
- `test`: Vitest tests that double as executable lessons
- `docs`: ordered course notes for TypeScript and ReBAC
- `deployments`: local OpenFGA docker-compose setup
- `practice/collab-docs-lite`: capstone exercise

## Why this layout

The repo keeps the teaching loop short. Most tests use an in-memory graph so
you can learn the model without running infrastructure. The OpenFGA adapter is
still included so the same application boundary can talk to a real OpenFGA
server when you are ready.

## Commands

```bash
npm install
npm test
npm run build
npm run dev
```

To start OpenFGA locally:

```bash
docker compose -f deployments/docker-compose.yml up -d
```

Then use `src/authz/openfga-client.ts` as the production adapter for the same
`Authorizer` interface used in the tests.
