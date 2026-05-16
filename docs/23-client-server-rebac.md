# Client/server ReBAC demo

This repo now includes a small client/server example.

It is intentionally modest:

- Node built-in `http` server
- JSON API
- interactive terminal client
- no Express
- no TUI framework
- ReBAC enforced in the service layer

The goal is to show the pattern before adding more libraries.

## Scene

So far, many examples run inside one process. Real authorization usually sits
behind a server boundary: a client asks for something, the server checks the
graph, and only then does the action happen.

This demo makes that boundary visible.

## Run the server

```bash
npm run server
```

The server listens on:

```text
http://127.0.0.1:4000
```

Health check:

```bash
curl http://127.0.0.1:4000/health
```

## Run the client

In another terminal:

```bash
npm run client
```

The client is a simple interactive terminal UI. It lets you:

- read the seeded `roadmap` document
- update the `roadmap` document
- try different actors

Actors:

```text
alice    -> can edit through team membership
bob      -> can read as workspace viewer
chandra  -> denied by default
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
curl "http://127.0.0.1:4000/documents/roadmap?actorId=bob"
```

Example update:

```bash
curl -X PATCH "http://127.0.0.1:4000/documents/roadmap" \
  -H "content-type: application/json" \
  -d '{"actorId":"alice","body":"Updated from curl"}'
```

Bob can read but cannot update:

```bash
curl -X PATCH "http://127.0.0.1:4000/documents/roadmap" \
  -H "content-type: application/json" \
  -d '{"actorId":"bob","body":"Should fail"}'
```

## Where ReBAC is enforced

The HTTP layer parses requests and maps errors to responses.

The domain service enforces authorization:

```ts
await this.requireAllowed(input.actor, "can_edit", documentObject(input.id), "edit");
```

That is the important boundary.

The client does not decide whether Bob can edit. The server decides. The server
uses the domain service. The domain service uses the authorizer.

```text
client -> HTTP server -> DocumentService -> Authorizer -> relationship graph
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
npm run server
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
chandra
```

If you can predict who can read and who can edit before pressing Enter, the
client/server ReBAC pattern is working as a teaching tool.
