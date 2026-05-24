# TypeScript ReBAC implementation — theory to code

This doc bridges the earlier theory chapters to the TypeScript implementation.
After reading this you should be able to open any file in `typescript/src/` and
know exactly why it exists and how it connects to the concepts you learned.

Prerequisites:
- Docs 01–02 (OAuth / authn vs authz fundamentals)
- Docs 03–04 (graph theory + ReBAC concepts)
- Doc 05 (OpenFGA model)
- Doc 12 (factory pattern, ports and adapters)

---

## The architecture in one diagram

```text
              ┌─────────────────────────────────────────┐
              │              src/core/                   │
              │                                          │
              │  domain/documents/  ◄── business rules   │
              │  ports/authn.ts     ◄── "who are you?"   │
              │  ports/authz.ts     ◄── "what can you do?"│
              └─────────────┬───────────────────────────┘
                            │ depends on (via ports)
              ┌─────────────▼───────────────────────────┐
              │            src/adapters/                 │
              │                                          │
              │  authn/   — token verification           │
              │  authz/   — graph traversal, OpenFGA     │
              │  db/      — in-memory document store     │
              │  http/    — HTTP request/response        │
              │  client/  — HTTP + terminal clients      │
              └─────────────┬───────────────────────────┘
                            │ wired by
              ┌─────────────▼───────────────────────────┐
              │           composition roots              │
              │                                          │
              │  server/compose.ts  — HTTP server        │
              │  cli/compose.ts     — terminal client    │
              │  demo/compose.ts    — local graph demo   │
              └─────────────────────────────────────────┘
```

The core never imports from adapters. Adapters import from core. Both depend only
on the port interfaces declared in core — not on each other's concrete types.

---

## From authn theory to code

Doc 01 introduced OAuth 2.0 bearer tokens. In production:

1. Client sends `Authorization: Bearer <token>` with every request.
2. Server verifies the token against an IdP (introspection or JWT signature).
3. Server learns the caller's identity (`sub` claim) and allowed scopes.

In the TypeScript implementation:

**Port** — `src/core/ports/authn.ts`
```ts
export interface Authenticator {
    verifyAccessToken: (authorizationHeader: string | undefined) => Promise<AuthenticatedUser>;
}
```
This is what the domain needs. The adapter decides how to implement it.

**Adapter** — `src/adapters/authn/makeDemoTokenVerifier.ts`
```ts
const makeDemoTokenVerifier = ({ tokens }: DemoTokenVerifierCfg): Authenticator => {
    const verifyAccessToken = async (header) => {
        const token = extractBearer(header);   // strips "Bearer " prefix
        const claims = tokens[token];          // looks up in static table
        if (!claims) throw AuthenticationError("Invalid token");
        return { subject: user(claims.sub), scopes: claims.scopes };
    };
    return { verifyAccessToken };
};
```

In a real system this adapter would call `jwt.verify()` or an IdP `/introspect`
endpoint. The domain code does not change — it only knows about the
`Authenticator` port.

**Result** — an `AuthenticatedUser`:
```ts
{ subject: "user:alice", scopes: ["documents:read", "documents:write"] }
```

The `subject` field (`"user:alice"`) becomes the `actor` in every authz check.

---

## From authz theory to code

Doc 02 explained that authz answers "what may this caller do?" ReBAC (docs 03–04)
answers that question by traversing a graph of relationship tuples.

### The relationship graph — `src/core/ports/authz.ts`

Docs 03–04 talked about nodes, edges, and typed relationships. The code
represents this as `TupleKey` values stored in a `TupleStore`:

```ts
// A single edge in the graph:
// "user:alice has relation 'editor' on document:roadmapDocument"
export type TupleKey = {
    object:   RebacObject;   // e.g. "document:roadmapDocument"
    relation: Relation;      // e.g. "editor"
    user:     Subject;       // e.g. "user:alice" or "team:platform#member"
};
```

The `Subject` field is where subject sets appear. `"team:platform#member"` means
"everyone who holds the `member` relation on `team:platform`" — exactly the
subject-set concept from doc 04.

The `TupleStore` port exposes just two methods:

```ts
export interface TupleStore {
    has:                  (object, relation, user) => boolean;
    findByObjectRelation: (object, relation)       => TupleKey[];
}
```

`has` checks for a direct edge. `findByObjectRelation` retrieves all edges
leaving a node on a given relation type — used when expanding subject sets or
inherited workspace permissions.

### The permission model — `src/adapters/authz/permissionModel.ts`

Doc 05 defined the OpenFGA model with type definitions and computed relations.
`permissionModel.ts` is the same model expressed as plain data tables:

```ts
// workspace.owner implies workspace.editor implies workspace.viewer
export const WORKSPACE_RULES: ImpliedBy = {
    editor: ["owner"],   // "editor is satisfied by owner"
    viewer: ["editor"],  // "viewer is satisfied by editor"
};

// document.can_edit is satisfied by having editor role
export const DOCUMENT_RULES: ImpliedBy = {
    can_edit:    ["editor"],
    can_delete:  ["owner"],
    can_read:    ["viewer"],
    viewer:      ["editor"],
    editor:      ["owner"],
};
```

Reading the table: `DOCUMENT_RULES.viewer = ["editor"]` means "if you hold
`editor` on a document, you also hold `viewer`." The traversal in
`makeGraphAuthorizer.ts` reads these tables during graph expansion.

### Graph traversal — `src/adapters/authz/makeGraphAuthorizer.ts`

Doc 03 described DFS traversal with cycle detection. The `hasRelation` function
is that traversal:

```
hasRelation(user, object, relation, trace, visited)
│
├─ [cycle guard] skip if (object#relation) already in visited set
│
├─ hasTuple(user, object, relation)?              ← direct edge check
│   ├─ tupleStore.has(object, relation, user)?    ← direct match
│   └─ any subject-set tuple containing user?     ← e.g. team:platform#member
│       └─ subjectSetContains → recursive hasRelation on the team object
│
└─ expand via permission model rules
    ├─ object type = "team"      → expandByRules(TEAM_RULES, ...)
    ├─ object type = "workspace" → expandByRules(WORKSPACE_RULES, ...)
    └─ object type = "document"  → expandDocument(...)
        ├─ expandByRules(DOCUMENT_RULES, ...)       ← role hierarchy
        └─ for each workspace parent tuple          ← workspace inheritance
            └─ hasRelation(user, parent, relation)  ← recurse into workspace
```

The `trace` array accumulates a human-readable log of every step. When you run
the demo (`npm run demo`) you see exactly this trace printed to the terminal.

#### Concrete example

Checking `can_edit` for `user:alice` on `document:roadmapDocument`:

```
Check whether user:alice has can_edit on document:roadmapDocument
  document:roadmapDocument can_edit includes editor     ← DOCUMENT_RULES
  Check whether user:alice has editor on document:roadmapDocument
    document:roadmapDocument editor includes owner
    Check whether user:alice has owner on document:roadmapDocument
      No direct tuple found
      document:roadmapDocument owner can inherit owner from workspace:productWorkspace
      Check whether user:alice has owner on workspace:productWorkspace
        No direct tuple found
        workspace:productWorkspace owner includes ... (alice is not owner)
      Result: denied this branch
    (expand via subject sets)
    Found subject-set tuple (document:roadmapDocument, editor, team:platform#member)
    Resolve subject set: does team:platform#member contain user:alice?
    Check whether user:alice has member on team:platform
      Found direct tuple (team:platform, member, user:alice)
    Result: allowed
```

Alice reaches `can_edit` through the subject-set path: she is a member of
`team:platform`, and `team:platform#member` holds `editor` on the document.

---

## From request to response — `src/adapters/http/makeHttpHandler.ts`

The HTTP handler is where authn and authz meet. Every protected route follows
the same two-step pattern:

```ts
// 1. Authn — establish caller identity from the bearer token
const authed = await authenticator.verifyAccessToken(request.authorization);

// 2. Domain — the use case enforces authz via the Authorizer port
const doc = await documents.read({ id: documentId, actor: authed.subject });
```

Step 1 can throw `AuthenticationError` (missing or invalid token → 401).
Step 2 can throw `ForbiddenError` (relationship check failed → 403) or
`DocumentNotFoundError` (document missing → 404).

The `toErrorResponse` function maps tagged errors to HTTP status codes using
type guards — no `instanceof`, no class hierarchy:

```ts
const toErrorResponse = (error: unknown): HttpResponse => {
    if (isAuthenticationError(error))   return json(401, { error: error.message });
    if (isForbiddenError(error))        return json(403, { error: error.message });
    if (isDocumentNotFoundError(error)) return json(404, { error: error.message });
    return json(400, { error: "Unknown error" });
};
```

---

## The demo fixtures — `src/demo/fixtures.ts`

The relationship graph used by all demos and tests is seeded from `fixtures.ts`.
It represents a realistic but minimal workspace scenario:

```text
team:platform
  admin: user:alice
  member: user:alice      (implied by admin via TEAM_RULES)

workspace:productWorkspace
  owner: user:alice
  viewer: user:bob

document:roadmapDocument
  workspace: workspace:productWorkspace   ← parent link for inheritance
  editor: team:platform#member            ← subject set — platform members are editors
```

Why this is enough to demonstrate ReBAC:

| Actor       | Can read? | Can edit? | Reason                                        |
|-------------|-----------|-----------|-----------------------------------------------|
| `user:alice`| ✓         | ✓         | platform team member → editor via subject set |
| `user:bob`  | ✓         | ✗         | workspace viewer → can_read; not editor       |
| `user:casey`| ✗         | ✗         | no tuples at all → denied                    |

---

## Putting it all together — composition root

`src/server/compose.ts` is where every piece is wired together:

```ts
const tupleStore    = makeInMemoryTupleStore({ seed: seedRelationshipTuples() });
const authorizer    = makeGraphAuthorizer({ tupleStore });
const authenticator = makeDemoTokenVerifier({ tokens: demoTokens });
const repository    = makeInMemoryDocumentRepository();
const documents     = makeDocuments({ repository, authorizer });
const handler       = makeHttpHandler({ authenticator, documents });
const server        = makeHttpServer({ handler });
```

Each line introduces one adapter. Nothing in `core/` imports from `adapters/`.
Swapping from the in-memory graph to the real OpenFGA adapter is a one-line
change here — nothing else changes.

---

## File map — theory chapter to source file

| Theory chapter      | Core concept          | Source file(s)                              |
|---------------------|-----------------------|---------------------------------------------|
| Doc 01 (OAuth)      | Bearer token authn    | `ports/authn.ts`, `authn/makeDemoTokenVerifier.ts` |
| Doc 02 (authz)      | Actor + permission    | `ports/authn.ts` (AuthenticatedUser), `domain/documents/` |
| Doc 03 (graph)      | DFS with cycle guard  | `authz/makeGraphAuthorizer.ts`              |
| Doc 04 (ReBAC)      | Tuples, subject sets  | `ports/authz.ts`, `authz/makeInMemoryTupleStore.ts` |
| Doc 05 (OpenFGA)    | Permission schema     | `authz/permissionModel.ts`, `authz/model.ts` |
| Doc 12 (factories)  | Ports and adapters    | `core/ports/`, `adapters/`, composition roots |

---

## Checkpoint

Trace a read request for `user:bob` on `document:roadmapDocument` from HTTP
entry to the authz decision. What three functions are involved? What error would
be thrown if Bob tried to edit instead of read?

Good answer:

1. `makeHttpHandler` calls `authenticator.verifyAccessToken` → gets `user:bob`.
2. `documents.read` → `makeReadDocument` calls `authorizer.check` with
   `{ user: "user:bob", relation: "can_read", object: "document:roadmapDocument" }`.
3. `makeGraphAuthorizer.check` traverses the graph: bob has `viewer` on the
   workspace, workspace inheritance gives `viewer` on the document, `viewer`
   satisfies `can_read`. Result: allowed.

For edit: `authorizer.check` with `can_edit`. Bob has only `viewer`, not
`editor`, and is not in `team:platform`. Graph returns `allowed: false`.
`makeUpdateDocument` throws `ForbiddenError`. `makeHttpHandler` maps it to 403.
