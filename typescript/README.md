# TypeScript ReBAC Primer

This implementation is **two small HTTP services** in a ports-and-adapters
shape, sharing one vocabulary module.

```text
src/shared/rebac.ts                ReBAC value helpers (objects, relations, tuples, checks)

src/authz-service/                 AuthZ service (port 4100)
  core/domain/                       AuthzService interface + makeAuthzService
  core/ports/                        TupleRepository, Evaluator
  adapters/db/                       in-memory tuple store
  adapters/graph/                    graph evaluator + permission model
  adapters/http/                     HTTP handler + server
  compose.ts, index.ts               wiring + entrypoint

src/documents-service/             Documents service (port 4000)
  core/domain/                       Documents interface + create/read/update + errors
  core/ports/                        Authenticator, AuthzClient, DocumentRepository
  adapters/authn/                    demo bearer-token verifier
  adapters/authz/                    HTTP client to the AuthZ service
  adapters/db/                       in-memory document repository
  adapters/http/                     HTTP handler + server
  adapters/client/                   HTTP + terminal clients
  compose.ts, index.ts               wiring + entrypoint

src/cli/                           terminal client entrypoint + wiring
src/demo/fixtures.ts               shared demo data (actors, tokens, seed tuples)
test/                              Vitest tests
```

The dependency direction is the important part:

```text
adapters -> core <- composition roots
```

The core defines the ports and business rules. Adapters translate outside-world
details (HTTP, terminal I/O, token parsing, in-memory maps) into those ports.
Composition roots (`compose.ts`) choose which adapters to use. The core never
imports from `adapters/`.

| Concept | Port in `core` | Adapter |
|---------|----------------|---------|
| Authentication | `Authenticator` | `documents-service/adapters/authn/makeDemoTokenVerifier.ts` |
| Authorization (check) | `AuthzClient` | `documents-service/adapters/authz/makeAuthzServiceClient.ts` |
| Permission evaluation | `Evaluator` | `authz-service/adapters/graph/makeGraphEvaluator.ts` |
| Relationship storage | `TupleRepository` | `authz-service/adapters/db/makeInMemoryTupleRepository.ts` |
| Document persistence | `DocumentRepository` | `documents-service/adapters/db/makeInMemoryDocumentRepository.ts` |
| HTTP (documents) | `Documents` domain API | `documents-service/adapters/http/makeDocumentsHttpHandler.ts` |
| Terminal client | `DocumentsClient` | `documents-service/adapters/client/*` |

The documents service depends on the `AuthzClient` port. In production that is
satisfied by `makeAuthzServiceClient` (HTTP to the authz service on :4100); in
tests it is satisfied by `makeInProcessAuthzClient` (the real graph evaluator,
no socket). The domain code is identical either way.

Useful commands:

```bash
npm install
npm run check       # tsc (type-check) + vitest
npm run authz       # AuthZ service on port 4100
npm run documents   # Documents service on port 4000
npm run dev         # both services, watch mode
npm start           # both services (no watch)
npm run client      # terminal client (talks to documents :4000)
```

Demo bearer tokens (both services authenticate via `Authorization: Bearer`):

```text
demo-token-alice  user:alice, can read and edit (platform team -> workspace editor)
demo-token-bob    user:bob, can read only (workspace viewer)
demo-token-casey  user:casey, authenticated but denied by ReBAC
```
