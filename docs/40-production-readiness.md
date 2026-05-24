# Production readiness

This repo teaches the ReBAC mental model and the correct architectural patterns
for layering authorization in a TypeScript or Go application. That foundation is
solid and directly transferable to a production system.

But the gap between a working tutorial and a production-ready system is real.
This chapter names the gaps so you can close them deliberately.

Production-ready does not mean "more complicated for its own sake." It means:

- the authenticated user is verified by the server, not trusted from input
- authorization data is durable, auditable, and updated with domain changes
- OpenFGA runs as a real service with pinned config, health checks, and backups
- failures are handled intentionally instead of accidentally allowing access
- tests prove the model, the app wiring, and the cross-language examples agree
- the system can be operated, observed, rolled back, and explained under stress

## What this repo does well

Before listing what is missing, it is worth being explicit about what carries
over directly:

- The three-layer separation: HTTP parses -> domain service decides authz is
  required -> authorizer answers allow/deny
- Factory/constructor injection with an `AuthzClient` interface: swapping
  `makeAuthzServiceClient` or `NewGraphAuthorizer` for a real OpenFGA client
  requires one change, in one place
- The tuple model: objects, relations, subject sets, inheritance via `from`
- The OpenFGA DSL: types, type restrictions, computed permissions
- Testing authorization behavior rather than mocks
- Composition roots that keep wiring out of domain code

Those patterns apply unchanged in production.

## The production shape

The tutorial keeps everything small enough to understand in one sitting. A real
deployment has more moving parts, but the request still follows the same path.

```text
Browser / CLI / agent
  |
  | 1. Sends OAuth/OIDC access token
  v
HTTP API
  |
  | 2. Verifies token signature, issuer, audience, expiry
  v
Application service
  |
  | 3. Loads domain data and asks: "may this actor do this?"
  v
Authorizer port
  |
  | 4. Calls OpenFGA check/listObjects/batchCheck
  v
OpenFGA service
  |
  | 5. Reads model + relationship tuples from postgres/mysql
  v
Allow / deny decision
```

The important idea is that authentication and authorization stay separate:

```text
Authn proves identity:     "this request is user:123"
Authz answers permission:  "may user:123 edit document:456?"
ReBAC stores relationships: "user:123 is member of workspace:abc"
```

## Production readiness map

Use this map when you turn the tutorial into a service.

| Area | Tutorial version | Production version |
|------|------------------|--------------------|
| Identity | actor passed in request field | verified OAuth/OIDC token or session |
| Authorization engine | in-memory graph authorizer | OpenFGA service via SDK |
| Domain data | in-memory repository | database with migrations and backups |
| Relationship tuples | seeded fixture | written/deleted from domain events |
| Config | local defaults | environment variables and secret manager |
| Containers | local Docker examples | pinned images, health checks, non-root runtime |
| Observability | test output and local logs | structured logs, metrics, traces, audit logs |
| Failure mode | tutorial errors | timeouts, retries, circuit breakers, fail-closed rules |
| Tests | unit tests | unit, model, integration, contract, migration, load tests |
| Operations | manual local startup | repeatable deploys, rollback, runbooks |

This repo is intentionally a learning system, not a deployable SaaS starter kit.
The value is that the boundaries are correct: each production concern has a
specific place to attach without rewriting the domain model.

## Twelve-factor configuration

Production services should read operational config from the environment, not
hard-code it in source files. The code should validate config at startup and
fail loudly if required values are missing.

Typical config for this repo's production-shaped version:

| Variable | Purpose |
|----------|---------|
| `PORT` | HTTP server port |
| `FGA_API_URL` | OpenFGA API endpoint |
| `FGA_STORE_ID` | OpenFGA store ID |
| `FGA_MODEL_ID` | pinned authorization model ID |
| `OIDC_ISSUER` | expected issuer for access tokens |
| `OIDC_AUDIENCE` | expected audience for this API |
| `OIDC_JWKS_URL` | public keys used to verify JWT signatures |
| `DATABASE_URL` | application database connection string |
| `LOG_LEVEL` | structured logging level |

The application should parse this once at the composition root. Domain services
should receive typed dependencies, not `process.env` or `os.Getenv` directly.

## Gap 1: Tuple lifecycle management

The tutorial seeds a static fixture at startup:

```ts
const repository = makeInMemoryTupleRepository(seedPolicyTuples());
```

In production, tuples are written in response to domain events:

| Event | Tuples to write |
|-------|-----------------|
| User joins a team | `(team:x, member, user:y)` |
| Document created in a workspace | `(document:x, workspace, workspace:y)` |
| User leaves a team | delete `(team:x, member, user:y)` |
| Document moved to a different workspace | delete old `workspace` tuple, write new one |
| User is offboarded | delete all tuples for that user |

OpenFGA exposes write APIs for both tuple writes and tuple deletes. SDKs often
wrap these with helper methods such as `WriteTuples` and `DeleteTuples`. The
application must call them as part of the consistency strategy for the domain
event that caused the relationship change.

The hard problem is consistency: if a document is created but the tuple write
fails, the creator cannot see their own document. You need a strategy for this
— options include outbox pattern, saga, or atomic writes depending on your
database.

A common production pattern is:

```text
1. Start database transaction
2. Write domain change, for example "document created"
3. Write outbox event, for example "document.created"
4. Commit transaction
5. Background worker reads outbox event
6. Worker writes required OpenFGA tuples
7. Worker marks outbox event processed
```

This avoids the worst failure mode: domain data committed but no durable record
of the relationship change that still needs to happen.

For critical flows, make tuple writes idempotent. Retrying the same tuple write
should not corrupt state. Retrying the same tuple delete should be safe when the
tuple is already gone.

Also add reconciliation jobs. A reconciliation job compares domain state against
OpenFGA tuples and repairs drift:

```text
Domain says: document:roadmap belongs to workspace:platform
OpenFGA has: no document:roadmap#workspace tuple
Repair job: write document:roadmap#workspace@workspace:platform
```

Reconciliation is not a substitute for correct writes, but it is how mature
systems recover from partial failures, deploy bugs, and manual data repairs.

## Gap 2: OpenFGA deployment

The tutorial uses graph authorizers, in-process implementations that are
deliberately educational. The production path is to install the SDK for the
language track you are using:

```text
npm install @openfga/sdk
go get github.com/openfga/go-sdk
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

Do not run production on `latest` image tags. Pin image versions so a deployment
is repeatable. OpenFGA's published tags do not carry a leading `v` — use the
plain `X.Y.Z` form you see on the [releases page](https://github.com/openfga/openfga/releases):

```yaml
image: openfga/openfga:X.Y.Z
```

Production OpenFGA also needs the same operational basics as any other service:

- database backups and restore tests
- readiness and liveness probes
- request timeouts
- connection pool limits
- TLS between services, or private network boundaries
- resource requests and limits in Kubernetes
- alerting for high latency, error rate, and datastore failures

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

Treat the model like code:

```text
model.fga lives in git
model tests run in CI
model write happens during deploy
application config points at the approved model ID
rollback restores the previous app version and previous model ID
```

For this learning repo, it is fine that the model is shown in docs and tests.
For production, keep the model file as a first-class deploy artifact.

## Gap 4: Consistency

OpenFGA can be deployed with datastore and replica setups where freshness
matters. This is most visible in two common patterns.

**Write then check**: a user creates a document and immediately tries to read it.
The tuple write may not yet be visible to the read replica serving the check.

OpenFGA exposes a `consistency` parameter:

```ts
await fgaClient.check({
  user: "user:alice",
  relation: "can_read",
  object: "document:roadmapDocument",
  consistency: ConsistencyPreference.HigherConsistency
});
```

`HIGHER_CONSISTENCY` asks OpenFGA to favor fresher reads. It may be slower, but
it is the safer choice for immediately-after-write reads. `MINIMIZE_LATENCY` is
better for the common read path when stale reads are acceptable.

Choose based on the operation, not globally.

## Gap 5: Performance — list-objects and batch-check

The tutorial calls `check` once per operation. At scale, two patterns become
necessary.

**List-objects**: instead of checking each document one by one, ask OpenFGA
which documents a user can access:

```ts
const response = await fgaClient.listObjects({
  user: "user:alice",
  relation: "can_read",
  type: "document"
});
// response.objects -> ["document:roadmapDocument", ...]
```

This is how authorization-aware list endpoints work without N+1 check calls.

**Batch-check**: if your deployed OpenFGA version supports it, check multiple
relations in one request:

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
{ "actorId": "alice" }
```

In production, the actor identity comes from a verified token, not a request
field. The flow is:

```text
Client sends:   Authorization: Bearer <JWT>
Server verifies JWT signature and expiry
Server extracts `sub` claim: "user-uuid-123"
Server maps sub to OpenFGA user ID: "user:user-uuid-123"
Document domain calls authorizer with that ID
```

Never trust a request field for the acting user. The JWT must be verified by
the server before its claims are used for authorization. Key steps:

1. Validate the token signature against your identity provider's JWKS endpoint.
2. Check `exp`, `iss`, and `aud` claims.
3. Map the `sub` claim to your OpenFGA user ID format.

Libraries: `jose` (Node-native), `jsonwebtoken`, or your framework's auth
middleware.

The domain layer should not know about JWTs. Keep the boundary clean:

```text
HTTP middleware verifies token
HTTP handler builds RequestContext { actorId, requestId, traceId }
Domain service receives actorId
Authorizer receives OpenFGA-formatted user string
```

That separation matters because the same domain service may later be called by:

- a browser user with an access token
- a CLI using device authorization flow
- a service account using client credentials
- an agent acting on behalf of a human
- a background worker replaying an outbox event

Each caller authenticates differently. The authorization question remains the
same: "what subject is acting, and what may it do?"

Use stable, non-guessable subject IDs. Do not use email addresses as OpenFGA
user IDs; emails change and can leak personal data into logs. Prefer:

```text
user:7c970d8c-3a23-456c-b8bb-e170f7c20a9f
service-account:billing-worker
agent-session:session-01J...
```

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

Never rename a relation without checking whether any tuples and checks still use
it. A relation rename can turn a previously valid access path into a deny or
error in the current model.

## Gap 9: Error handling and fallback strategy

When the OpenFGA service is unavailable, your application must decide:

```text
fail open   -> allow the request (dangerous for sensitive operations)
fail closed -> deny the request (safer, may impact availability)
```

There is no universally correct answer. The default should be **fail closed**
for write operations and privileged reads. Use circuit breakers and timeouts to
avoid cascading latency from a slow authorization service.

The `AuthzClient` interface in this repo makes this easy to handle in one place:

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

## Gap 10: Persistent domain data

The tutorial repositories are in-memory so the core ideas stay visible. A real
service needs durable storage for documents, workspaces, users, teams, and audit
events.

Production persistence needs:

- schema migrations, not ad-hoc table changes
- transactions around domain state changes
- unique constraints for identifiers and memberships
- optimistic locking or version columns for concurrent edits
- backups and restore drills
- data retention and deletion policy
- indexes for common read paths

The important ReBAC point is that domain data and relationship tuples are
related but not the same thing.

```text
Application DB:
  document id, title, workspace_id, created_by, timestamps

OpenFGA:
  document:doc123#workspace@workspace:platform
  workspace:platform#member@user:alice
```

The application database is the source of truth for business facts. OpenFGA is
the source of truth for authorization relationships. Production systems need a
clear rule for how facts create, update, and delete tuples.

## Gap 11: Tenant isolation

Most real authorization systems are multi-tenant. Tenant isolation should be
designed early because it affects object IDs, tuple writes, and list queries.

Two common strategies:

| Strategy | Shape | Tradeoff |
|----------|-------|----------|
| Tenant in object IDs | `document:tenantA/doc123` | one OpenFGA store, careful ID discipline |
| Store per tenant | separate store IDs | stronger isolation, more operational overhead |

For learning, a single store is easier. For production, choose deliberately.
The dangerous bug is cross-tenant leakage:

```text
Bad:
document:doc123#viewer@user:alice

Better:
document:tenantA/doc123#viewer@user:tenantA/alice
```

If tenant boundaries are strict, validate tenant membership before tuple writes.
Do not allow an API request to write:

```text
document:tenantA/doc123#viewer@user:tenantB/bob
```

unless the product explicitly supports cross-tenant sharing and has tests for
that behavior.

## Gap 12: Observability

When authorization fails in production, the debugging question is rarely "did
the code run?" The question is usually:

```text
Which actor?
Which object?
Which relation?
Which model ID?
Which tuple was missing?
Was OpenFGA slow, unavailable, or returning deny correctly?
```

Add structured logs around authorization decisions. Avoid logging raw tokens or
personal data.

Example log fields:

```json
{
  "event": "authz.check",
  "actor": "user:7c970d8c",
  "relation": "can_edit",
  "object": "document:tenantA/doc123",
  "allowed": false,
  "model_id": "01HV...",
  "request_id": "req_123",
  "duration_ms": 14
}
```

Useful metrics:

- authorization check latency
- authorization allow/deny counts
- OpenFGA error count by status code
- tuple write success/failure count
- outbox lag
- reconciliation repair count
- listObjects result size and latency

Useful traces:

```text
HTTP request span
  -> load document span
  -> OpenFGA check span
  -> domain operation span
  -> tuple write/outbox span
```

Observability is part of correctness. If the team cannot explain a deny result,
the system is not production-ready.

## Gap 13: Security hardening

The tutorial keeps HTTP simple. Production HTTP services need guardrails:

- HTTPS at the edge
- secure cookies if browser sessions are used
- CSRF protection for cookie-authenticated browser writes
- CORS restricted to known origins
- request body size limits
- rate limits for login-adjacent and expensive endpoints
- input validation at the HTTP boundary
- no stack traces in public error responses
- dependency scanning
- least-privilege service credentials
- secret rotation plan

For OpenFGA specifically:

- keep admin APIs off the public internet
- protect store/model write permissions
- separate local, staging, and production stores
- avoid using production tuples in uncontrolled developer environments
- treat tuple exports as sensitive data

Authorization data reveals organization structure. Even if a tuple does not
contain document content, it may reveal who works with whom and which resources
exist.

## Gap 14: Deployment and operations

Production readiness includes the ability to change the system safely.

Required deployment habits:

- build immutable container images
- pin base images and service images
- run as a non-root user where practical
- expose health and readiness endpoints
- handle graceful shutdown
- set CPU and memory limits
- run database migrations explicitly
- deploy OpenFGA model changes deliberately
- keep rollback instructions short and tested

A safe release sequence for an authorization change:

```text
1. Run model tests in CI
2. Run app tests in TS and Go
3. Write new OpenFGA model to staging
4. Replay representative tuples against staging model
5. Deploy app pointing at staging model ID
6. Promote model to production
7. Deploy app with new FGA_MODEL_ID
8. Watch authz deny/error metrics
9. Roll back app config to previous FGA_MODEL_ID if needed
```

Treat `FGA_MODEL_ID` as a release control. It lets you roll forward and back
without guessing which model the app is using.

## Gap 15: Testing strategy

The repo already has unit tests that teach behavior. Production needs a wider
test pyramid.

| Test type | What it proves |
|-----------|----------------|
| Domain unit tests | services ask for authorization before changing state |
| Authorizer adapter tests | TS and Go map app requests into OpenFGA requests correctly |
| Model tests | the `.fga` model allows and denies the intended cases |
| Integration tests | app works against a real OpenFGA container |
| Contract tests | TS and Go examples behave the same for the same scenario |
| Migration tests | model and tuple changes preserve important access paths |
| Load tests | list and check endpoints stay fast enough at realistic size |
| Failure tests | app fails closed when OpenFGA is unavailable |

Test both positive and negative cases. Authorization tests that only prove
"Alice can read the document" are incomplete. You also need:

```text
Alice can read because she is a workspace member.
Bob cannot read because he is not a workspace member.
Alice cannot edit if she is only a viewer.
Deleted memberships stop granting access.
Cross-tenant tuples are rejected or ignored.
```

The negative tests are where most authorization bugs are found.

## Gap 16: Production readiness for agentic systems

Agentic systems make authorization more important, not less. An agent may call
tools, chain actions, summarize private data, or act while the human is away.

A production agentic authorization design should answer:

- Who is the human principal?
- Is the agent acting as the human, as itself, or as a delegated session?
- Which tools can this agent call?
- Which resources can each tool access?
- What is the maximum action scope?
- When does the delegation expire?
- How is every tool call audited?
- Can the human revoke the delegation?

A useful model is:

```text
human user
  -> grants limited delegation
agent session
  -> may call approved tool
tool invocation
  -> performs normal ReBAC check against target resource
```

Example relationship language:

```text
user:alice is owner of workspace:platform
agent-session:s123 is delegate of user:alice
agent-session:s123 has tool_access to tool:create-document
document:roadmap is in workspace:platform
```

The tool call should still ask the same authorization question:

```text
May agent-session:s123 create document in workspace:platform?
```

Depending on risk, the answer may require both:

```text
agent-session:s123 is delegated by user:alice
user:alice can_create_document in workspace:platform
agent-session:s123 has tool_access to tool:create-document
```

Do not give an agent a raw all-access API token just because the human has broad
access. Use scoped delegation, short lifetimes, clear audit trails, and explicit
tool permissions.

## What not to ship from the tutorial

These are acceptable for learning and unacceptable as-is in production:

- actor IDs supplied by request body or query string
- in-memory document repositories
- in-memory tuple stores
- OpenFGA running with the memory datastore
- Docker images using `latest` instead of explicit version tags
- static seed tuples as the only authorization state
- missing JWT/session verification
- missing request size limits
- missing tuple audit logs
- no rollback story for model changes
- no integration test against real OpenFGA

This does not make the tutorial code "bad." It means the tutorial code is
optimized for clarity. Production code must optimize for correctness,
operability, and failure recovery.

## Production-ready checklist

Use this as a final review before calling a ReBAC service production-ready.

- Authn: tokens or sessions are verified at the edge.
- Authn: issuer, audience, expiry, and signature checks are enforced.
- Authn: the app maps identity provider subjects to stable internal subjects.
- Authz: domain services enforce checks before sensitive reads and writes.
- Authz: OpenFGA model is stored in git and tested.
- Authz: application pins `FGA_MODEL_ID`.
- Authz: relationship tuples are written from domain events.
- Authz: tuple writes are idempotent and auditable.
- Authz: tuple drift can be detected and repaired.
- Data: domain state is persisted in a database with migrations.
- Data: tuple updates and domain updates have a consistency strategy.
- Tenancy: object IDs and tuple writes cannot leak across tenants.
- Runtime: config comes from environment or secret manager.
- Runtime: startup validates required config and fails fast.
- Runtime: OpenFGA calls have timeouts and safe fallback behavior.
- Runtime: app fails closed for sensitive operations.
- Runtime: logs, metrics, traces, and audit logs exist.
- Runtime: health checks and graceful shutdown are implemented.
- Security: CORS, CSRF, body limits, TLS, and rate limits are handled where relevant.
- Testing: positive and negative authorization cases are covered.
- Testing: app is tested against a real OpenFGA service before deploy.
- Operations: rollback is possible by restoring the previous app config and model ID.

## What to read next

- [OpenFGA documentation](https://openfga.dev/docs) — the authoritative source
  for deployment, model design, and API reference
- [OpenFGA SDK for Node.js](https://github.com/openfga/js-sdk) — the production
  client used by the TypeScript OpenFGA adapter
- [OpenFGA SDK for Go](https://github.com/openfga/go-sdk) — the production
  client used by the Go adapter
- [OpenFGA sample stores](https://github.com/openfga/sample-stores) — worked
  examples for common access models (Google Drive, GitHub, Slack)
- [FGA Playground](https://play.fga.dev) — test your model and tuples in a
  browser before writing code
- [OAuth 2.0 Security Best Current Practice](https://www.rfc-editor.org/rfc/rfc9700)
  — security guidance for OAuth-based authentication flows
- [OpenID Connect Core](https://openid.net/specs/openid-connect-core-1_0-final.html)
  — identity layer commonly used to authenticate users before authorization

## Summary

| Gap | What to add |
|-----|-------------|
| Tuple lifecycle | write tuples on domain events; handle write failures |
| OpenFGA runtime | run OpenFGA as a service with a postgres backend |
| Store setup | create store and model via API; pin model ID in config |
| Consistency | choose `HIGHER_CONSISTENCY` after writes |
| Performance | use `listObjects` and `batchCheck` for scale |
| Token propagation | verify JWT; map `sub` to OpenFGA user ID |
| Audit logging | append-only log of every tuple write |
| Model versioning | deploy new model versions safely |
| Failure handling | fail closed; use circuit breakers |
| Domain persistence | replace in-memory repos with database-backed adapters |
| Tenant isolation | prevent cross-tenant tuple leakage |
| Observability | logs, metrics, traces, and audit events |
| Security hardening | TLS, input limits, CORS/CSRF, least privilege |
| Deployment | pinned images, health checks, graceful rollback |
| Testing | model, integration, contract, migration, load, and failure tests |
| Agentic systems | scoped delegation, tool permissions, audit trails |

The patterns in this repo — interface-based authorizer, composition roots,
domain-layer enforcement — hold in all of these scenarios. The gaps are
operational and infrastructure concerns layered on top of a design that is
already correct.
