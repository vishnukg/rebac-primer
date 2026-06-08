# Client/server ReBAC demo

This repo includes small client/server examples.

The TypeScript version is intentionally modest:

- Node built-in `http` server
- JSON API
- interactive terminal client
- no Express
- no TUI framework
- ReBAC enforced in the document domain

The Go version exposes the same HTTP shape with the standard library. The goal
is to show the pattern before adding more libraries.

## Scene

So far, many examples run inside one process. Real authorization usually sits
behind a server boundary: a client asks for something, the server checks the
graph, and only then does the action happen.

This demo makes that boundary visible.

## Run the services

TypeScript (starts both authz :4100 and documents :4000):

```bash
npm run dev
```

Or start each individually:

```bash
npm run authz       # AuthZ service on port 4100
npm run documents   # Documents service on port 4000
```

Go:

```bash
make go/server
```

The servers listen on:

```text
TypeScript AuthZ:      http://127.0.0.1:4100
TypeScript Documents:  http://127.0.0.1:4000
Go:                    http://127.0.0.1:4001
```

Health check:

```bash
curl http://127.0.0.1:4000/health
curl http://127.0.0.1:4001/health
```

## Run the client

In another terminal (after services are running):

```bash
npm run client
```

The client is a simple interactive terminal UI. It lets you:

- read the seeded `roadmapDocument`
- update the `roadmapDocument`
- try different actors

Actors:

```text
alice -> Alice, can edit through Platform Team membership
bob   -> Bob, can read as a direct workspace viewer
casey -> Casey, denied by default
```

## API routes

Both servers share the same resource shape **and the same authentication
style**: a standard `Authorization: Bearer <token>` header. The demo tokens are
`demo-token-alice`, `demo-token-bob`, and `demo-token-casey` in both
implementations. (The demo token verifier is a stand-in for real JWT
verification — see doc 01.)

```text
TypeScript (documents :4000)          Go (:4001)
──────────────────────────────────    ──────────────────────────────────
GET   /health                         GET   /health
GET   /whoami                         GET   /whoami
POST  /documents                      POST  /documents
GET   /documents/:id                  GET   /documents/{id}
PATCH /documents/:id                  PATCH /documents/{id}
```

(The TypeScript documents service talks to the AuthZ service on :4100 behind the
scenes; the client only ever calls :4000.)

Example read (Bob is a workspace viewer — 200):

```bash
# TypeScript
curl "http://127.0.0.1:4000/documents/roadmapDocument" \
  -H "Authorization: Bearer demo-token-bob"

# Go
curl "http://127.0.0.1:4001/documents/roadmapDocument" \
  -H "Authorization: Bearer demo-token-bob"
```

Example update (Alice can edit via team → workspace editor — 200):

```bash
# TypeScript
curl -X PATCH "http://127.0.0.1:4000/documents/roadmapDocument" \
  -H "Authorization: Bearer demo-token-alice" \
  -H "content-type: application/json" \
  -d '{"body":"Updated from curl"}'

# Go
curl -X PATCH "http://127.0.0.1:4001/documents/roadmapDocument" \
  -H "Authorization: Bearer demo-token-alice" \
  -H "content-type: application/json" \
  -d '{"body":"Updated from curl"}'
```

Bob can read but cannot update (403):

```bash
# TypeScript
curl -X PATCH "http://127.0.0.1:4000/documents/roadmapDocument" \
  -H "Authorization: Bearer demo-token-bob" \
  -H "content-type: application/json" \
  -d '{"body":"Should fail"}'

# Go
curl -X PATCH "http://127.0.0.1:4001/documents/roadmapDocument" \
  -H "Authorization: Bearer demo-token-bob" \
  -H "content-type: application/json" \
  -d '{"body":"Should fail"}'
```

Verify your identity (both implementations expose `/whoami`):

```bash
# TypeScript
curl "http://127.0.0.1:4000/whoami" -H "Authorization: Bearer demo-token-alice"
# Go
curl "http://127.0.0.1:4001/whoami" -H "Authorization: Bearer demo-token-alice"
# → {"user":"user:alice","scopes":["documents:read","documents:write"]}
```

## Where ReBAC is enforced

The HTTP layer parses requests and maps errors to responses.

The document domain enforces authorization.

TypeScript:

```ts
const { allowed } = await authzClient.check({
    user:     input.actor,
    relation: "can_edit",
    object:   document(input.id),
});
```

Go:

```go
err := s.requireAllowed(ctx, input.Actor, shared.RelationDocumentCanEdit, shared.Document(input.ID), "edit")
```

That is the important boundary.

The client does not decide whether Bob can edit. The server
decides. The server uses the document domain. The document domain uses the
authz client, which calls the AuthZ service.

```text
client -> documents :4000 -> Documents -> AuthzClient -> authz :4100 -> graph
```

## Composition roots in this demo

The executable files stay intentionally thin:

```text
src/authz-service/index.ts     -> composeAuthzService(), then listen()
src/documents-service/index.ts -> composeDocumentsService(), then listen()
src/cli/index.ts               -> composeCliApp(), then run()
go/cmd/server/main.go          -> buildHandler(), then ListenAndServe()
```

The object graphs are assembled in the composition roots:

```text
composeAuthzService (authz-service/compose.ts)
  -> makeInMemoryTupleRepository (seeded with policy tuples)
  -> makeGraphEvaluator
  -> makeAuthzService
  -> makeAuthzHttpHandler + makeAuthzHttpServer

composeDocumentsService (documents-service/compose.ts)
  -> makeAuthzServiceClient (HTTP to authz :4100)
  -> makeDemoTokenVerifier
  -> makeInMemoryDocumentRepository
  -> makeDocuments
  -> makeDocumentsHttpHandler + makeDocumentsHttpServer

composeCliApp (cli/compose.ts)
  -> makeHttpDocumentsClient
  -> Node readline terminal
  -> makeTerminalClient

go buildHandler() (cmd/server/main.go)
  -> authzdb.New (in-memory tuple store, seeded)
  -> graph.NewGraphEvaluator
  -> authz.New (authz service)
  -> docsdb.New (in-memory document repository)
  -> docsauthn.New (demo token verifier)
  -> documents.New (documents service)
  -> docshttp.NewServer (HTTP handler)
```

That split matters because ReBAC code is easier to reason about when business
rules do not create their own infrastructure. The document domain asks an
`AuthzClient` interface for a decision; the composition root wires
`makeAuthzServiceClient` as the concrete implementation.

```text
entrypoint -> composition root -> interfaces + concrete adapters
                          |
                          v
                 domain service uses interfaces
```

## Why this is only "TUI-like"

The client uses Node's built-in `readline/promises`. It is an interactive
terminal client, not a full-screen TUI.

That is intentional for now:

- no extra dependencies
- easy to read
- easy to debug
- enough to demonstrate client/server ReBAC

A future version could use Ink or another terminal UI library after the
client/server boundary is clear.

## Testing note

Some restricted execution environments block opening local listening sockets.
That is why the default test suite focuses on domain and graph behavior.

You can still run the services normally on your machine:

```bash
npm run dev     # starts both TypeScript services
make go/server  # starts the Go server
```

and exercise it with:

```bash
npm run client
```

## Checkpoint

Try the client as three actors:

```text
alice
bob
casey
```

If you can predict who can read and who can edit before pressing Enter, the
client/server ReBAC pattern is working as a teaching tool.
