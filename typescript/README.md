# TypeScript ReBAC Primer

This implementation uses a small ports-and-adapters shape:

```text
src/core/ports/*              interfaces and ReBAC value helpers
src/core/domain/documents/*   document use cases and domain errors
src/adapters/authn/*          bearer-token verification adapter
src/adapters/authz/*          graph/OpenFGA authorization adapters
src/adapters/db/*             document repository adapter
src/adapters/http/*           HTTP request/response adapter
src/adapters/client/*         HTTP and terminal clients
src/server/compose.ts         server wiring
src/cli/compose.ts            terminal-client wiring
src/demo/compose.ts           demo wiring
src/demo/fixtures.ts          shared demo data
```

The dependency direction is the important part:

```text
adapters -> core <- composition roots
```

The core defines the ports and business rules. Adapters translate outside-world
details, such as HTTP, terminal I/O, token parsing, OpenFGA, or in-memory maps,
into those ports. Composition roots choose which adapters to use.

In this repo, a **port** is an interface or function shape the core depends on.
An **adapter** is one concrete implementation of that shape.

| Concept | Port in `core` | Adapter |
|---------|----------------|---------|
| Authentication | `Authenticator` | `adapters/authn/makeDemoTokenVerifier.ts` |
| Authorization | `Authorizer` | `adapters/authz/makeGraphAuthorizer.ts`, `makeOpenFgaAuthorizer.ts` |
| Relationship reads | `TupleStore` read methods | `adapters/authz/makeInMemoryTupleStore.ts` |
| Document persistence | `DocumentRepository` | `adapters/db/makeInMemoryDocumentRepository.ts` |
| HTTP input/output | `Documents` domain API | `adapters/http/makeHttpHandler.ts` |
| Terminal client | `DocumentsClient` | `adapters/client/*` |

The server composition root creates concrete adapters, passes them into core
factories, and then starts the HTTP server. The core never imports from
`src/adapters`.

Learning flow:

1. Authn: `adapters/authn` verifies a demo OAuth2-style bearer token and returns `user:*`.
2. Authz: `core/ports/authz.ts` defines ReBAC objects, relations, tuples, and the `Authorizer` port.
3. Graph authz: `adapters/authz/makeGraphAuthorizer.ts` answers checks from relationship tuples.
4. Documents: `core/domain/documents` protects create/read/update with ReBAC.
5. HTTP: `adapters/http` turns requests into authenticated document calls.
6. Composition: `server/compose.ts` wires concrete adapters together.

Useful commands:

```bash
npm install
npm run check
npm run demo
npm run server
npm run client
```

Demo tokens for HTTP:

```text
demo-token-alice  user:alice, can read and edit
demo-token-bob    user:bob, can read only
demo-token-casey  user:casey, authenticated but denied by ReBAC
```
