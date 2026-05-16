# Production readiness

This repo teaches the ReBAC mental model and the correct architectural patterns
for layering authorization in a TypeScript application. That foundation is solid
and directly transferable to a production system.

But the gap between a working tutorial and a production-ready system is real.
This chapter names the gaps so you can close them deliberately.

## What this repo does well

Before listing what is missing, it is worth being explicit about what carries
over directly:

- The three-layer separation: HTTP parses → domain service decides authz is
  required → authorizer answers allow/deny
- Constructor injection with an `Authorizer` interface — swapping
  `GraphAuthorizer` for a real OpenFGA client requires one change, in one place
- The tuple model: objects, relations, subject sets, inheritance via `from`
- The OpenFGA DSL: types, type restrictions, computed permissions
- Testing authorization behavior rather than mocks
- Composition roots that keep wiring out of domain code

Those patterns apply unchanged in production.

## Gap 1: Tuple lifecycle management

The tutorial seeds a static fixture at startup:

```ts
const tupleStore = new InMemoryTupleStore(seedRelationshipTuples());
```

In production, tuples are written in response to domain events:

| Event | Tuples to write |
|-------|-----------------|
| User joins a team | `(team:x, member, user:y)` |
| Document created in a workspace | `(document:x, workspace, workspace:y)` |
| User leaves a team | delete `(team:x, member, user:y)` |
| Document moved to a different workspace | delete old `workspace` tuple, write new one |
| User is offboarded | delete all tuples for that user |

The OpenFGA API has `Write` (for both writes and deletes) and `WriteTuples` /
`DeleteTuples` methods. The application must call them transactionally with the
domain event that caused them.

The hard problem is consistency: if a document is created but the tuple write
fails, the creator cannot see their own document. You need a strategy for this
— options include outbox pattern, saga, or atomic writes depending on your
database.

## Gap 2: OpenFGA deployment

The tutorial uses `GraphAuthorizer`, an in-process implementation that is
deliberately educational. The production path is:

```text
npm install @openfga/sdk
```

Then run OpenFGA as a service. OpenFGA provides:

- Docker image: `openfga/openfga`
- Kubernetes Helm chart
- Managed cloud offering

OpenFGA needs a backend store. Options:

```text
memory     -> development and tests only
postgres   -> recommended for production
mysql      -> supported
```

A minimal Docker Compose for local development with a real OpenFGA server:

```yaml
services:
  openfga:
    image: openfga/openfga:latest
    command: run
    environment:
      OPENFGA_DATASTORE_ENGINE: postgres
      OPENFGA_DATASTORE_URI: postgres://openfga:openfga@postgres/openfga
    ports:
      - "8080:8080"
    depends_on:
      - postgres

  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: openfga
      POSTGRES_PASSWORD: openfga
      POSTGRES_DB: openfga
```

Run `openfga migrate` before starting the server to initialize the schema.

## Gap 3: Store and model initialization

Every OpenFGA deployment needs:

1. A **store** — a namespace for your models and tuples
2. A **model** — the DSL schema written to that store
3. A **store ID** and **model ID** — used in every API call

In production, you manage these through the OpenFGA API or CLI:

```bash
fga store create --name "my-app"
fga model write --store-id $STORE_ID --file model.fga
```

Your application reads the store ID and model ID from environment variables, not
from code.

```ts
const fgaClient = new OpenFgaClient({
  apiUrl: process.env.FGA_API_URL,
  storeId: process.env.FGA_STORE_ID,
  authorizationModelId: process.env.FGA_MODEL_ID
});
```

The `authorizationModelId` pin is important. Without it, every check uses the
latest model, which means a model deployment can change authorization behavior
for in-flight requests. Pinning the model ID makes deployments explicit.

## Gap 4: Consistency

OpenFGA is eventually consistent when using replicated databases. This matters
for two common patterns.

**Write then check**: a user creates a document and immediately tries to read it.
The tuple write may not yet be visible to the read replica serving the check.

OpenFGA exposes a `consistency` parameter:

```ts
await fgaClient.check({
  user: "user:workspaceEditor",
  relation: "can_read",
  object: "document:roadmapDocument",
  consistency: ConsistencyPreference.HigherConsistency
});
```

`HIGHER_CONSISTENCY` reads from the primary. It is slower but correct for
immediately-after-write reads. `MINIMIZE_LATENCY` uses replicas and is better
for the common read path.

Choose based on the operation, not globally.

## Gap 5: Performance — list-objects and batch-check

The tutorial calls `check` once per operation. At scale, two patterns become
necessary.

**List-objects**: instead of checking each document one by one, ask OpenFGA
which documents a user can access:

```ts
const response = await fgaClient.listObjects({
  user: "user:workspaceEditor",
  relation: "can_read",
  type: "document"
});
// response.objects -> ["document:roadmapDocument", ...]
```

This is how authorization-aware list endpoints work without N+1 check calls.

**Batch-check** (OpenFGA v1.5+): check multiple relations in one request:

```ts
const response = await fgaClient.batchCheck({
  checks: [
    { user: "user:x", relation: "can_read",  object: "document:y" },
    { user: "user:x", relation: "can_edit",  object: "document:y" },
    { user: "user:x", relation: "can_delete", object: "document:y" }
  ]
});
```

Use this when a single UI action needs multiple permission answers at once.

## Gap 6: Token propagation

The tutorial identifies actors by a string passed in a request field:

```json
{ "actorId": "workspaceEditor" }
```

In production, the actor identity comes from a verified token, not a request
field. The flow is:

```text
Client sends:   Authorization: Bearer <JWT>
Server verifies JWT signature and expiry
Server extracts `sub` claim: "user-uuid-123"
Server maps sub to OpenFGA user ID: "user:user-uuid-123"
Domain service calls authorizer with that ID
```

Never trust a request field for the acting user. The JWT must be verified by
the server before its claims are used for authorization. Key steps:

1. Validate the token signature against your identity provider's JWKS endpoint.
2. Check `exp`, `iss`, and `aud` claims.
3. Map the `sub` claim to your OpenFGA user ID format.

Libraries: `jose` (Node-native), `jsonwebtoken`, or your framework's auth
middleware.

## Gap 7: Audit logging

Every tuple write should be logged with:

```text
who changed it      (the acting user or service account)
what changed        (the tuple: object, relation, user)
when                (timestamp)
why                 (the operation that caused it, e.g. "user joined team")
```

OpenFGA does not store this history for you. Your application must write it.

A minimal audit log entry:

```ts
type TupleAuditEntry = {
  actorId: string;
  operation: "write" | "delete";
  tuple: TupleKey;
  cause: string;
  timestamp: Date;
};
```

Store these in an append-only table or a dedicated audit log system. Compliance
requirements (SOC 2, GDPR, HIPAA) almost always ask for this.

## Gap 8: Model versioning

The OpenFGA model is versioned by default — every `WriteAuthorizationModel` call
creates a new model version. But your deployment process needs to handle:

- Rolling out a new model without breaking in-flight requests
- Migrating tuples when relation names change
- Testing the new model against a snapshot of real tuples before deploying

The safest pattern:

1. Deploy the new model version alongside the old one.
2. Pin new app instances to the new model ID.
3. Drain old app instances.
4. Optionally clean up the old model version.

Never rename a relation without checking whether any tuples use it. OpenFGA
will silently return `denied` for checks against a relation that no longer exists
in the current model.

## Gap 9: Error handling and fallback strategy

When the OpenFGA service is unavailable, your application must decide:

```text
fail open   -> allow the request (dangerous for sensitive operations)
fail closed -> deny the request (safer, may impact availability)
```

There is no universally correct answer. The default should be **fail closed**
for write operations and privileged reads. Use circuit breakers and timeouts to
avoid cascading latency from a slow authorization service.

The `Authorizer` interface in this repo makes this easy to handle in one place:

```ts
async check(request: CheckRequest): Promise<CheckResult> {
  try {
    return await this.fgaClient.check(request);
  } catch (error) {
    if (isTransientError(error)) {
      // log, record metric, return denied
      return { allowed: false };
    }
    throw error;
  }
}
```

## What to read next

- [OpenFGA documentation](https://openfga.dev/docs) — the authoritative source
  for deployment, model design, and API reference
- [OpenFGA SDK for Node.js](https://github.com/openfga/js-sdk) — the production
  client that replaces `GraphAuthorizer`
- [OpenFGA sample stores](https://github.com/openfga/sample-stores) — worked
  examples for common access models (Google Drive, GitHub, Slack)
- [FGA Playground](https://play.fga.dev) — test your model and tuples in a
  browser before writing code

## Summary

| Gap | What to add |
|-----|-------------|
| Tuple lifecycle | write tuples on domain events; handle write failures |
| Deployment | run OpenFGA as a service with a postgres backend |
| Store setup | create store and model via API; pin model ID in config |
| Consistency | choose `HIGHER_CONSISTENCY` after writes |
| Performance | use `listObjects` and `batchCheck` for scale |
| Token propagation | verify JWT; map `sub` to OpenFGA user ID |
| Audit logging | append-only log of every tuple write |
| Model versioning | deploy new model versions safely |
| Failure handling | fail closed; use circuit breakers |

The patterns in this repo — interface-based authorizer, composition roots,
domain-layer enforcement — hold in all of these scenarios. The gaps are
operational and infrastructure concerns layered on top of a design that is
already correct.
