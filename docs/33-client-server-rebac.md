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
make go-server
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
npm run cli
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

```text
GET /health
POST /documents
GET /documents/:id?actorId=alice
PATCH /documents/:id
```

Example read:

```bash
curl "http://127.0.0.1:4000/documents/roadmapDocument?actorId=bob"
curl "http://127.0.0.1:4001/documents/roadmapDocument?actorId=bob"
```

Example update:

```bash
curl -X PATCH "http://127.0.0.1:4000/documents/roadmapDocument" \
  -H "content-type: application/json" \
  -d '{"actorId":"alice","body":"Updated from curl"}'

curl -X PATCH "http://127.0.0.1:4001/documents/roadmapDocument" \
  -H "content-type: application/json" \
  -d '{"actorId":"alice","body":"Updated from curl"}'
```

Bob can read but cannot update:

```bash
curl -X PATCH "http://127.0.0.1:4000/documents/roadmapDocument" \
  -H "content-type: application/json" \
  -d '{"actorId":"bob","body":"Should fail"}'

curl -X PATCH "http://127.0.0.1:4001/documents/roadmapDocument" \
  -H "content-type: application/json" \
  -d '{"actorId":"bob","body":"Should fail"}'
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
err := s.requireAllowed(ctx, input.Actor, authz.RelationDocumentCanEdit, authz.Document(input.ID), "edit")
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
go/cmd/server/main.go          -> app.New(), then ListenAndServe()
```

The object graphs are assembled in the composition roots:

```text
composeAuthzService (authz-service/compose.ts)
  -> makeInMemoryTupleRepository (seeded with policy tuples)
  -> makeGraphEvaluator
  -> makeAuthzDomain
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

go app.New
  -> NewInMemoryTupleStore
  -> NewGraphAuthorizer
  -> NewInMemoryDocumentRepository
  -> NewDocumentService
  -> httpserver.NewServer
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
make go-server  # starts the Go server
```

and exercise it with:

```bash
npm run cli
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
