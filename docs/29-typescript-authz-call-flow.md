# TypeScript: how an authorization check flows through the code

This chapter traces a single HTTP request end to end and shows **every function
call** it makes on the way to an allow/deny decision. Read it with the files
open. By the end you should be able to point at any layer and say what it does
and what it calls next.

It complements:

- `docs/18-typescript-rebac-implementation.md` — the theory-to-code walkthrough.
- `docs/19-factory-function-pattern.md` — why everything is a `make*` factory.

This is the glue: how a request reaches the evaluator and how the answer gets
back to the client. Its Go twin is `docs/28-go-authz-call-flow.md` — read them
side by side to see the one big difference, called out below.

---

## The request we are tracing

```text
GET /documents/roadmapDocument
Authorization: Bearer demo-token-bob
```

Bob is a viewer of the workspace the roadmap lives in, so this returns **200**
with the document. The same request as `PATCH` (an edit) returns **403**,
because Bob is not an editor. We trace the read in full, then show where the 403
branches off.

---

## The shape of the system: two services, not one

This is the big difference from Go. In TypeScript the documents service and the
authz service are **separate processes** that talk over HTTP:

```text
  client (cli / curl)
        │  HTTP :4000
        ▼
 ┌──────────────────────── documents-service (port 4000) ─────────────────────┐
 │  makeDocumentsHttpServer  →  makeDocumentsHttpHandler  →  makeReadDocument  │
 │   (node http)                 (authn + routing)            (domain)         │
 │                                                               │ check()      │
 │                                                         AuthzClient port     │
 │                                                               │              │
 │                                              makeAuthzServiceClient (adapter)│
 └───────────────────────────────────────────────────────────────┬───────────┘
                                                                   │ HTTP :4100
                                                                   │ POST /check
                                                                   ▼
 ┌──────────────────────── authz-service (port 4100) ─────────────────────────┐
 │  makeAuthzHttpServer  →  makeAuthzHttpHandler  →  composeAuthzDomain           │
 │   (node http)             (routing)               (check)                  │
 │                                                       │ evaluate()           │
 │                                              makeGraphEvaluator → repository │
 └────────────────────────────────────────────────────────────────────────────┘
```

The `AuthzClient` port is the seam. The documents domain calls
`authzClient.check(...)`; the concrete implementation behind it is
`makeAuthzServiceClient`, which does `fetch(POST http://127.0.0.1:4100/check)`.

> **In tests** the same port is satisfied by `makeInProcessAuthzClient`
> (`test/fixtures.ts`), which runs the real graph evaluator in-process and skips
> HTTP entirely. The domain code is byte-for-byte identical — only the adapter
> wired in changes. That is the whole point of the port.

---

## The call flow, step by step

### 0. Wiring — `src/documents-service/compose.ts`

The composition root picks the concrete adapters:

```ts
const authzClient   = makeAuthzServiceClient({ baseUrl: authzUrl }); // HTTP to :4100
const authenticator = makeDemoTokenVerifier({ tokens });
const repository    = makeInMemoryDocumentRepository();
const documents     = composeDocuments({ repository, authzClient }); // domain gets the port
const handler = makeDocumentsHttpHandler({ authenticator, documents });
const server  = makeDocumentsHttpServer({ handler });
```

`composeDocuments` receives `authzClient` as a plain object satisfying the
`AuthzClient` interface. It never learns whether that object talks HTTP or runs
in-process.

### 1. Node HTTP server — `adapters/http/makeDocumentsHttpServer.ts`

`createServer` turns the raw Node request into a plain `HttpRequest` object
(method, path, query, authorization header, parsed JSON body) and calls the
handler. Decoupling the transport from the handler is what lets the same handler
run under `httptest`-style unit tests with no socket.

### 2. Authn + routing — `adapters/http/makeDocumentsHttpHandler.ts`

```ts
const docId = matchDocumentPath(request.path);   // "roadmapDocument"
if (docId && request.method === "GET") {
    const authed = await authenticator.verifyAccessToken(request.authorization); // step 3
    return json(200, { document: await documents.read({ id: docId, actor: authed.subject }) }); // step 4
}
```

The whole body is wrapped in `try/catch`; thrown domain errors are turned into
status codes by `toErrorResponse` (see the 403 section).

### 3. Authn — `adapters/authn/makeDemoTokenVerifier.ts`

`verifyAccessToken` strips `Bearer `, looks the token up, and returns
`{ subject: "user:bob", scopes: [...] }`. **This answers *who is asking*. The
*what may they do* question is the authz check in step 5.** A missing/invalid
token throws `AuthenticationError` → 401.

### 4. Use case — `core/domain/makeReadDocument.ts`

```ts
const read = async ({ id, actor }) => {
    const doc = await repository.findById(id);
    if (!doc) throw DocumentNotFoundError(id);          // → 404, checked first

    const { allowed } = await authzClient.check({       // ── step 5: the boundary
        user: actor, relation: "can_read", object: document(id),
    });
    if (!allowed) throw ForbiddenError(`${actor} cannot read ${id}`);
    return doc;
};
```

Existence is checked **before** authorization, so a missing document is 404, not 403.

### 5. The authorization boundary — `adapters/authz/makeAuthzServiceClient.ts`

This is where TypeScript leaves the process. `check` POSTs to the authz service:

```ts
const check = async (request) => {
    const result = await post("/check", {              // fetch → http://127.0.0.1:4100/check
        user: request.user, relation: request.relation, object: request.object,
    });
    if (!isCheckResult(result)) throw new Error("unexpected /check response");
    return result;                                     // { allowed, trace }
};
```

The request crosses the network. The next four steps happen **in the other
service**.

### 6. Authz HTTP in — `authz-service/adapters/http/makeAuthzHttpHandler.ts`

```ts
if (request.method === "POST" && request.path === "/check") {
    const result = await authz.check({ user, relation, object });  // step 7
    return json(200, { allowed: result.allowed, trace: result.trace });
}
```

### 7. Authz core — `authz-service/core/domain/composeAuthzDomain.ts`

```ts
const check = (request) => evaluator.evaluate(request);   // delegate to the Evaluator port
```

The core does no graph work itself; it delegates to whichever `Evaluator` was
wired (the graph evaluator today; the OpenFGA SDK tomorrow).

### 8. Graph traversal — `authz-service/adapters/graph/makeGraphEvaluator.ts`

`evaluate` runs the recursive `hasRelation` search:
`can_read → viewer → (workspace inheritance) → workspace:productWorkspace viewer
→ Bob is a direct viewer → allowed`, returning `{ allowed: true, trace: [...] }`.
The recursion is the same algorithm `docs/18` and (in Go terms)
`docs/27` walk through. It reads tuples from the `TupleRepository`; it never writes.

### 9. The answer propagates back — across the network and up the stack

```text
makeGraphEvaluator.evaluate → { allowed: true }      (authz-service)
  composeAuthzDomain.check returns it
    authz handler → json(200, { allowed, trace })
      ── HTTP response :4100 ──►
        makeAuthzServiceClient.check parses { allowed:true }   (documents-service)
          makeReadDocument: allowed → returns the document
            documents handler → json(200, { document })
              ── HTTP response :4000 ──► client
```

---

## Where the 403 comes from (the same flow, denied)

For `PATCH` as Bob, step 4 is `makeUpdateDocument` and step 5 checks `can_edit`.
Bob is a viewer, so the authz service returns `{ allowed: false }`, and:

```ts
if (!allowed) throw ForbiddenError(`${actor} cannot edit ${id}`);
```

The thrown error unwinds to the handler's `catch`, where `toErrorResponse`
chooses the status code:

```ts
// adapters/http/makeDocumentsHttpHandler.ts
const toErrorResponse = (error) => {
    if (isAuthenticationError(error))   return json(401, { error: error.message });
    if (isForbiddenError(error))        return json(403, { error: error.message });
    if (isDocumentNotFoundError(error)) return json(404, { error: error.message });
    return json(400, { error: error instanceof Error ? error.message : "Unknown error" });
};
```

So the **domain decides allow/deny** (by throwing typed errors) and the **HTTP
adapter decides the status code** (by matching them). The error type guards
(`isForbiddenError`, etc.) are how TypeScript recovers the type after it has been
thrown as a generic `unknown` in `catch`.

---

## Layer summary

| Layer | File | Responsibility | Calls next |
|---|---|---|---|
| HTTP in | `documents-service/adapters/http/makeDocumentsHttpServer.ts` | Node socket → `HttpRequest` | the handler |
| Routing + errors | `.../makeDocumentsHttpHandler.ts` | route, map thrown errors→status | authn, then domain |
| Authn | `.../adapters/authn/makeDemoTokenVerifier.ts` | token → identity | (returns) |
| Use case | `core/domain/makeReadDocument.ts` | exists? allowed? | `authzClient.check` |
| **Boundary** | `adapters/authz/makeAuthzServiceClient.ts` | **`fetch` POST /check** | the authz service |
| Authz HTTP in | `authz-service/adapters/http/makeAuthzHttpHandler.ts` | route `/check` | `authz.check` |
| Authz core | `core/domain/composeAuthzDomain.ts` | delegate to evaluator | `evaluator.evaluate` |
| Evaluator | `adapters/graph/makeGraphEvaluator.ts` | traverse the tuple graph | repository reads |

---

## The one idea to take away

`makeReadDocument` calls `authzClient.check(...)` and does not care what is on
the other side. In production that is an HTTP call to a separate authz service;
in tests it is an in-process function call (`makeInProcessAuthzClient`). Same
domain code, two completely different transports — chosen by the composition
root, not the domain.

This is the **same boundary** the Go implementation has
(`docs/28-go-authz-call-flow.md`), but Go wires it as an in-process method call
by default while TypeScript wires it as two HTTP services. The port (`AuthzClient`
in both languages) is what makes the transport a swappable detail rather than a
rewrite.

---

## Try it

1. Start both services (two terminals):
   `npm run authz` (port 4100), then `npm run documents` (port 4000).
2. Read as Bob (allowed): `curl :4000/documents/roadmapDocument -H "Authorization: Bearer demo-token-bob"` → 200.
3. Edit as Bob (denied): `curl -X PATCH :4000/documents/roadmapDocument -H "Authorization: Bearer demo-token-bob" -H "content-type: application/json" -d '{"body":"x"}'` → 403.
4. Watch the boundary: hit the authz service directly —
   `curl -X POST :4100/check -H "content-type: application/json" -d '{"user":"user:bob","relation":"can_edit","object":"document:roadmapDocument"}'`.
   The `trace` array in the response is exactly what step 8 built.

## Checkpoint

1. Which function call in the documents service crosses the network, and which
   interface lets the domain stay unaware of that?
2. In tests no authz service is running, yet the authz checks still work. How?
3. A viewer's edit is denied. Which file *threw* the denial, and which file
   turned it into `403`?

Good answers:
1. `makeAuthzServiceClient.check` (a `fetch` to `:4100/check`). The domain
   depends only on the `AuthzClient` interface, so it cannot tell an HTTP client
   from an in-process one.
2. The composition in the test swaps `makeAuthzServiceClient` for
   `makeInProcessAuthzClient`, which runs the real `makeGraphEvaluator` in the
   same process against a shared repository — no socket, same logic.
3. `core/domain/makeUpdateDocument.ts` threw `ForbiddenError`; 
   `adapters/http/makeDocumentsHttpHandler.ts` (`toErrorResponse`) mapped it to 403.
