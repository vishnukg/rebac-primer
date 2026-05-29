# TypeScript ReBAC implementation — structure, communication, and how authz works

Prerequisites:
- Docs 01–02 (authn vs authz fundamentals)
- Docs 03–04 (graph theory + ReBAC concepts)
- Doc 12 (factory functions, ports and adapters)

---

## Theory → code map

The first two theory docs draw a clean line between two questions.
Here is exactly where each question is answered in this codebase:

| Theory concept (Doc 01–02) | Where it lives in code |
|---|---|
| **Authentication** — "who are you?" | `documents-service/core/ports/authenticator.ts` (port) |
| Bearer token extraction + verification | `documents-service/adapters/authn/makeDemoTokenVerifier.ts` |
| **Authorization** — "what can you do?" | `authz-service/core/domain/makeAuthzDomain.ts` |
| Relationship tuple store | `authz-service/adapters/db/makeInMemoryTupleRepository.ts` |
| Graph traversal (ReBAC evaluation) | `authz-service/adapters/graph/makeGraphEvaluator.ts` |
| Permission model (who implies what) | `authz-service/adapters/graph/permissionModel.ts` |
| Shared vocabulary (subject/object/relation) | `shared/rebac.ts` |

The handoff from Doc 01 in one line of code — authn result feeds authz input:

```ts
// makeDemoTokenVerifier answers: "who are you?"
const { subject } = await authenticator.verifyAccessToken(authorizationHeader);

// makeGraphEvaluator answers: "what can you do?"
const { allowed } = await authzClient.check({ user: subject, relation: "can_edit", object });
```

---

## Authentication in this project (Doc 01)

Doc 01 describes the OAuth 2.0 bearer token flow:
1. Client sends `Authorization: Bearer <token>` with every request
2. Server verifies the token (JWT signature check, or token introspection against an IdP)
3. Server learns `{ sub, scopes }` — who the caller is and what they are allowed to do

**This project implements that exact flow**, with one simplification: instead of
a real JWT verifier, `makeDemoTokenVerifier` looks up the token in a static map.
The interface is identical to what you would use with a real IdP — only the
adapter changes.

```ts
// Port — what the domain needs (documents-service/core/ports/authenticator.ts)
export interface Authenticator {
    verifyAccessToken: (header: string | undefined) => Promise<AuthenticatedUser>;
}

export type AuthenticatedUser = {
    subject: RebacObject<"user">;  // "user:alice"
    scopes:  string[];             // ["documents:read", "documents:write"]
};
```

```ts
// Demo adapter (documents-service/adapters/authn/makeDemoTokenVerifier.ts)
// In production: swap this for a JWT verifier that calls your IdP.
const makeDemoTokenVerifier = ({ tokens }): Authenticator => ({
    verifyAccessToken: async header => {
        const token  = extractBearer(header);   // strips "Bearer " prefix
        const claims = tokens[token];            // static lookup
        if (!claims) throw AuthenticationError("Invalid token");
        return { subject: user(claims.sub), scopes: claims.scopes };
    },
});
```

**Where authn is used:**
- Every protected endpoint in `makeDocumentsHttpHandler` calls `authenticator.verifyAccessToken()` first
- `/whoami` returns the verified identity — the simplest demonstration of authn
- A 401 is returned for a missing or unrecognised token before any authz check runs
- The `subject` from authn becomes the `actor` that is passed into every domain operation

**What the scopes are for:**
Scopes (`documents:read`, `documents:write`) are how the token declares what the
client application is allowed to ask for. This is the OAuth 2.0 concept from
Doc 01 — the IdP can restrict a token to certain operations. In this demo they
are stored but not enforced; in production a handler would check
`scopes.includes("documents:write")` before allowing mutations.

---

## Authorization in this project (Doc 02–04)

Doc 02 introduces the decision shape: `subject + action + object → allow or deny`.
Doc 03–04 explain ReBAC: instead of flat roles, you store relationships as a graph
and traverse it.

**The check request maps directly to that shape:**

```ts
// shared/rebac.ts
export type CheckRequest = {
    user:     RebacObject<"user">;  // subject  e.g. "user:alice"
    relation: Relation;             // action   e.g. "can_edit"
    object:   RebacObject;          // object   e.g. "document:roadmapDocument"
};
```

**The relationship graph is the tuples:**

```
(team:platformTeam,          member,    user:alice)
(workspace:productWorkspace, editor,    team:platformTeam#member)
(workspace:productWorkspace, viewer,    user:bob)
(document:roadmapDocument,   workspace, workspace:productWorkspace)
(document:roadmapDocument,   owner,     user:alice)
```

Each tuple is one edge in the graph. The `team:platformTeam#member` entry is a
**subject set** (Doc 04 concept): "everyone who holds `member` on `team:platformTeam`".
It lets you grant permissions to a whole group without listing each user individually.

---

## The big picture

The project is two independent HTTP services that talk to each other:

```
┌─────────────────────────────┐       ┌─────────────────────────────┐
│      AuthZ Service          │       │    Documents Service         │
│      port 4100              │◄──────│    port 4000                 │
│                             │  HTTP │                              │
│  Stores relationships.      │       │  1. Authn: verify token      │
│  Evaluates "can X do Y?"    │       │  2. Authz: call authz svc    │
│  on the graph.              │       │  3. Execute business logic   │
└─────────────────────────────┘       └─────────────────────────────┘
          ▲                                        ▲
          │                                        │
    npm run authz                         npm run documents
```

They share one vocabulary: `src/shared/rebac.ts`.
That file defines `RebacObject`, `TupleKey`, `CheckRequest`, and all the helper
constructors (`user()`, `workspace()`, `tuple()`, …). Think of it as the SDK
both services would publish in production.

---

## Project structure

```
src/
├── shared/
│   └── rebac.ts                 ← shared types and constructors
│
├── authz-service/
│   ├── index.ts                 ← entrypoint: npm run authz
│   ├── compose.ts               ← wires all adapters together
│   ├── core/
│   │   ├── domain/
│   │   │   ├── types.ts         ← AuthzService interface + errors
│   │   │   └── makeAuthzDomain.ts ← check / writeTuples / deleteTuples / listTuples
│   │   └── ports/
│   │       └── tupleRepository.ts ← what the domain needs from storage
│   └── adapters/
│       ├── db/
│       │   └── makeInMemoryTupleRepository.ts
│       ├── graph/
│       │   ├── permissionModel.ts   ← rule tables (pure data)
│       │   └── makeGraphEvaluator.ts ← graph traversal
│       └── http/
│           ├── makeAuthzHttpHandler.ts  ← routes POST /check, POST /tuples, etc.
│           └── makeAuthzHttpServer.ts   ← Node HTTP server
│
├── documents-service/
│   ├── index.ts                 ← entrypoint: npm run documents
│   ├── compose.ts               ← wires all adapters together
│   ├── core/
│   │   ├── domain/
│   │   │   ├── types.ts         ← Documents interface + errors
│   │   │   ├── makeDocuments.ts ← assembles create/read/update
│   │   │   ├── makeCreateDocument.ts
│   │   │   ├── makeReadDocument.ts
│   │   │   └── makeUpdateDocument.ts
│   │   └── ports/
│   │       ├── authenticator.ts     ← "who are you?" (authn port)
│   │       ├── authzClient.ts       ← "what can you do?" (authz port)
│   │       └── documentRepository.ts
│   └── adapters/
│       ├── authn/
│       │   └── makeDemoTokenVerifier.ts  ← bearer token → identity
│       ├── authz/
│       │   └── makeAuthzServiceClient.ts ← HTTP client → authz service
│       ├── db/
│       │   └── makeInMemoryDocumentRepository.ts
│       ├── http/
│       │   ├── makeDocumentsHttpHandler.ts
│       │   └── makeDocumentsHttpServer.ts
│       └── client/
│           ├── makeHttpDocumentsClient.ts ← HTTP client for the CLI
│           └── makeTerminalClient.ts      ← interactive terminal loop
│
└── cli/
    ├── index.ts     ← entrypoint: npm run client
    └── compose.ts   ← wires readline + makeTerminalClient
```

### The rule: core never imports adapters

Every `core/` file only imports from `shared/` and other `core/` files.
Every `adapters/` file imports from `core/` to satisfy port interfaces.
`compose.ts` is the only file allowed to import from both — it is the wiring.

---

## Ports and adapters — why this shape

A **port** is an interface the domain declares it needs. It describes *what*,
not *how*. The domain does not know or care what is on the other side.

An **adapter** is the concrete implementation that satisfies a port. You can
swap adapters without touching domain code.

The documents service has two driven ports — one for each theory question:

```
core/ports/authenticator.ts         ← port: "I need something that can verify a token"
adapters/authn/makeDemoTokenVerifier.ts  ← adapter: static lookup (swap for JWT in prod)

core/ports/authzClient.ts           ← port: "I need something that can check()"
adapters/authz/makeAuthzServiceClient.ts ← adapter: calls the real HTTP service
```

In tests, a stub is passed instead of the real adapter — this is exactly
what `test/fixtures.ts` provides via `makeInProcessAuthzClient`:

```ts
// Satisfies AuthzClient port, uses real graph evaluator, no HTTP calls.
export const makeInProcessAuthzClient = (seed: TupleKey[] = []): AuthzClient => {
    const repository = makeInMemoryTupleRepository({ seed });
    const evaluator  = makeGraphEvaluator({ repository });
    return {
        check:       req => evaluator.evaluate(req),
        writeTuples: async tpls => { for (const t of tpls) repository.write(t); },
    };
};
```

---

## How the two services communicate

```
Documents Service (port 4000)            AuthZ Service (port 4100)
        │                                        │
        │  POST /check                           │
        │  { user: "user:alice",                 │
        │    relation: "editor",       ─────────►│
        │    object: "workspace:X" }             │
        │                                        │  evaluate graph
        │◄─────────────────────────────────────── │
        │  { allowed: true, trace: [...] }        │
        │                                        │
        │  POST /tuples                          │
        │  { tuples: [                           │
        │    { object: "document:abc",           │
        │      relation: "workspace",  ─────────►│
        │      user: "workspace:X" },            │
        │    { object: "document:abc",           │
        │      relation: "owner",                │
        │      user: "user:alice" }              │
        │  ]}                                    │
        │◄─────────────────────────────────────── │
        │  { written: 2 }                         │
```

The adapter that makes these calls is `makeAuthzServiceClient.ts`. It
implements the `AuthzClient` port using `fetch`. The documents domain never
sees HTTP — it just calls `authzClient.check()` and `authzClient.writeTuples()`.

---

## Full request walkthrough — create a document

Here is every step that happens when Alice sends `POST /documents`:

```
1.  HTTP request arrives at the Node server
        │
2.  makeDocumentsHttpServer parses method, path, headers, body
        │
3.  AUTHN: makeDocumentsHttpHandler reads the Authorization header
    → authenticator.verifyAccessToken("Bearer demo-token-alice")
    → returns { subject: "user:alice", scopes: ["documents:read", "documents:write"] }
    (401 if header is missing or token is unknown)
        │
4.  handler calls documents.create({
        id:        "roadmapDocument",
        title:     "Roadmap",
        body:      "Initial roadmap",
        workspace: "workspace:productWorkspace",
        actor:     "user:alice",          ← subject from step 3
    })
        │
5.  makeCreateDocument runs:

    a. AUTHZ CHECK: authzClient.check({
           user:     "user:alice",
           relation: "editor",
           object:   "workspace:productWorkspace",
       })
       → POST /check to AuthZ service
       → graph evaluator traverses tuples, returns { allowed: true }
       (403 ForbiddenError if allowed: false)

    b. repository.save(doc)
       → document stored in memory

    c. GRAPH UPDATE: authzClient.writeTuples([
           { object: "document:roadmapDocument", relation: "workspace", user: "workspace:productWorkspace" },
           { object: "document:roadmapDocument", relation: "owner",     user: "user:alice" },
       ])
       → POST /tuples to AuthZ service
       → both tuples now stored in the graph, enabling future can_read/can_edit checks
        │
6.  handler returns { statusCode: 201, body: { document: { id, title, body, … } } }
```

Step 3 is **authentication** (Doc 01). Steps 5a and 5c are **authorization** (Doc 02–04).
The document domain code never mixes them — authn happens in the HTTP adapter before
the domain is called, and authz happens inside the domain via the `authzClient` port.

---

## How authz checks work — the graph evaluator

The AuthZ service stores **tuples**: simple three-part assertions.

```
(workspace:productWorkspace, editor, team:platformTeam#member)
(team:platformTeam,           member, user:alice)
(document:roadmapDocument,   workspace, workspace:productWorkspace)
(document:roadmapDocument,   owner,     user:alice)
```

Each tuple reads: "`user` has `relation` on `object`".

When a check arrives — "does `user:alice` have `can_edit` on `document:roadmapDocument`?"
— the graph evaluator traverses the tuples depth-first:

```
Check: user:alice  can_edit  document:roadmapDocument
  │
  ├─ No direct tuple for (document:roadmapDocument, can_edit, user:alice)
  │
  ├─ DOCUMENT_RULES says: can_edit is satisfied by editor
  │   └─ Check: user:alice  editor  document:roadmapDocument
  │       ├─ No direct tuple for editor
  │       ├─ DOCUMENT_RULES says: editor is satisfied by owner
  │       │   └─ Check: user:alice  owner  document:roadmapDocument
  │       │       └─ Found direct tuple!  ✓  ALLOWED
  │       └─ (short-circuit: already found)
  └─ Result: allowed
```

If Alice were not the owner, it would continue to workspace inheritance:

```
  │       ├─ No owner tuple found
  │       └─ document inherits from workspace:
  │           lookup (document:roadmapDocument, workspace, ?)
  │           → finds tuple pointing to workspace:productWorkspace
  │           └─ Check: user:alice  editor  workspace:productWorkspace
  │               ├─ No direct editor tuple
  │               ├─ WORKSPACE_RULES says: editor is satisfied by owner
  │               ├─ No owner tuple
  │               └─ found: (workspace:productWorkspace, editor, team:platformTeam#member)
  │                   → this is a subject set, so resolve it:
  │                   └─ Check: user:alice  member  team:platformTeam
  │                       └─ Found direct tuple (team:platformTeam, member, user:alice)  ✓
```

The trace array in the `CheckResult` records every step so you can read exactly
why a check was allowed or denied.

### The permission model (rule tables)

The rules live in `permissionModel.ts` as plain objects — no logic, just data:

```ts
// "viewer is satisfied by editor; editor is satisfied by owner"
export const WORKSPACE_RULES: ImpliedBy = {
    editor: ["owner"],
    viewer: ["editor"],
};

// "can_edit is satisfied by editor; can_read is satisfied by viewer"
export const DOCUMENT_RULES: ImpliedBy = {
    can_read:    ["viewer"],
    can_comment: ["viewer"],
    can_edit:    ["editor"],
    can_delete:  ["owner"],
    viewer:      ["editor"],
    editor:      ["owner"],
};
```

The evaluator reads these tables during traversal. If you want to add a new
permission (e.g. `can_share`) you only need to add one line here — no logic changes.

### Cycle detection

The visited `Set<VisitKey>` in the evaluator prevents infinite loops if
relationship tuples ever form a cycle. Before traversing any `(object#relation)`
pair it checks whether it has already visited that pair in the current call stack.

---

## Code review — is this the most efficient way?

### What is genuinely well done

**Pure functions throughout.** Every factory function takes its dependencies as
arguments and returns an object. There is no hidden state, no singletons, no
module-level side effects. This makes every function testable in isolation.

**Ports and adapters cleanly separated.** The documents domain has zero knowledge
of HTTP or how the authz service is implemented. You could swap the HTTP authz
adapter for an in-process one (like the Go version does) by changing one line
in `compose.ts`.

**Permission model as data, not logic.** `permissionModel.ts` contains no
`if`/`switch`. The evaluator reads the tables generically. Adding a new
relation requires editing only the data file.

**Typed relationship strings.** Using template literal types (`RebacObject<"user">`,
`RebacObject<"workspace">`) means the TypeScript compiler catches
`"alice"` (missing prefix) or `"usr:alice"` (wrong prefix) at compile time.

### Trade-offs to be aware of

**One HTTP round-trip per authz check.**
Every document operation makes at least one `POST /check` call to the authz
service over the network. For a low-traffic demo this is fine. At scale you
would add:
- Response caching with a short TTL (check results are usually stable for seconds)
- Batch check endpoint (`POST /check/batch`) to resolve multiple permissions in one call

**`findByObjectRelation` does a linear scan.**
The in-memory store uses `[...store.values()].filter(...)`. For the demo with
a handful of tuples this is fast enough. A real store would use a secondary
index keyed on `(object, relation)` for O(log n) or O(1) lookup.

**No transaction between `repository.save` and `authzClient.writeTuples`.**
In `makeCreateDocument`, the document is saved first, then tuples are written.
If the `writeTuples` call fails (network error, authz service down), the document
exists in the repo but has no ownership record in the graph — future permission
checks for that document will fail. In production you would handle this with:
- An **outbox pattern**: write tuples as part of the same DB transaction as the document, publish them asynchronously
- Or retry logic around `writeTuples`

**The graph evaluator has no memoisation within a single `check` call.**
The `visited` set prevents infinite loops but does not cache positive results.
If the same `(object, relation)` is reachable via two different paths, both
paths are traversed. For the current graph depth this is imperceptible. Deep
graphs with many shared ancestors would benefit from a memo table.

**`writeTuples` writes one tuple at a time in a loop.**
```ts
const writeTuples = async (tuples: TupleKey[]): Promise<void> => {
    for (const t of tuples) repository.write(t);
};
```
For the in-memory store this is synchronous and instant. A real DB adapter
would want a batch insert here instead of N individual writes.

### What would change in production but is correct for learning

| This implementation | Production equivalent |
|---|---|
| `makeDemoTokenVerifier` (static map) | JWT verification against an IdP |
| `makeInMemoryTupleRepository` | Postgres with indexed tuples table |
| `makeInMemoryDocumentRepository` | Postgres / DynamoDB |
| `makeAuthzServiceClient` (HTTP) | Same, plus circuit breaker + retry |
| `makeGraphEvaluator` (in-process) | Could stay in-process or swap for OpenFGA |

None of these swaps require changing any `core/` code. That is the whole point
of ports and adapters.
