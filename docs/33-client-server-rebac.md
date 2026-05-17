# Client/server ReBAC demo

This repo includes small client/server examples.

The TypeScript version is intentionally modest:

- Node built-in `http` server
- JSON API
- interactive terminal client
- no Express
- no TUI framework
- ReBAC enforced in the service layer

The Go version exposes the same HTTP shape with the standard library. The goal
is to show the pattern before adding more libraries.

## Scene

So far, many examples run inside one process. Real authorization usually sits
behind a server boundary: a client asks for something, the server checks the
graph, and only then does the action happen.

This demo makes that boundary visible.

## Run the server

TypeScript:

```bash
make ts-server
```

Go:

```bash
make go-server
```

The servers listen on:

```text
TypeScript: http://127.0.0.1:4000
Go:         http://127.0.0.1:4001
```

Health check:

```bash
curl http://127.0.0.1:4000/health
curl http://127.0.0.1:4001/health
```

## Run the client

In another terminal:

```bash
make ts-client
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

The domain service enforces authorization.

TypeScript:

```ts
await this.requireAllowed(input.actor, "can_edit", documentObject(input.id), "edit");
```

Go:

```go
err := s.requireAllowed(ctx, input.Actor, authz.RelationDocumentCanEdit, authz.Document(input.ID), "edit")
```

That is the important boundary.

The client does not decide whether Bob can edit. The server
decides. The server uses the domain service. The domain service uses the
authorizer.

```text
client -> HTTP server -> DocumentService -> Authorizer -> relationship graph
```

## Composition roots in this demo

The executable files stay intentionally thin:

```text
src/server.ts     -> createServerApp(), then listen()
src/client/tui.ts -> createClientApp(), then run()
go/cmd/server/main.go -> app.New(), then ListenAndServe()
```

The object graphs are assembled in the composition roots:

```text
createServerApp
  -> createServices
    -> InMemoryTupleStore
    -> GraphAuthorizer
    -> InMemoryDocumentRepository
    -> DocumentService
  -> createHttpServer

createClientApp
  -> HttpDocumentsClient
  -> Node readline terminal
  -> TerminalClient

go app.New
  -> InMemoryTupleStore
  -> GraphAuthorizer
  -> InMemoryDocumentRepository
  -> DocumentService
  -> httpserver.NewServer
```

That split matters because ReBAC code is easier to reason about when business
rules do not create their own infrastructure. The document service asks an
`Authorizer` interface for a decision; the composition root decides that the
teaching implementation is `GraphAuthorizer`.

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

You can still run the server normally on your machine:

```bash
make ts-server
make go-server
```

and exercise it with:

```bash
make ts-client
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
