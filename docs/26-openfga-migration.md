# From the In-Process Evaluator to Production OpenFGA

The repo has two authz backends:

1. an in-process graph evaluator, which exposes the core mechanics
2. an OpenFGA adapter, which delegates evaluation and tuple storage

Both satisfy the `documents.AuthorizationService` interface. Other consumers,
such as the authz HTTP example, declare their own required interface.

This is a backend substitution, not proof that the surrounding demo is
production-ready. Token verification, durable document storage, OpenFGA
authentication, consistency choices, and operational controls remain separate
production concerns.

The migration principle is:

```text
keep the application contract and policy meaning
replace the educational evaluator and volatile relationship store
prove parity before moving decision authority
```

Use the two implementations to evaluate the production choice, not to imply
they have equal capabilities. The custom evaluator implements only the policy
features needed by this tutorial. OpenFGA is a broader authorization system.

## The Decision

The custom path means your team owns:

```text
policy language and validation
graph/set evaluation
relationship indexes and storage
listing and search algorithms
consistency and caching semantics
model migrations
limits, observability, and operations
```

The OpenFGA path means your team adopts those engine mechanics but still owns:

```text
the product policy
relationship sources of truth
application enforcement points
OpenFGA deployment and authentication
event delivery and reconciliation
consistency choices
model testing and rollout
```

For requirements that map naturally to OpenFGA's typed relationships and
userset rules, OpenFGA is usually the lower-risk starting point. A custom engine
needs a requirement that OpenFGA cannot meet—not merely the observation that a
basic recursive evaluator is small.

## What Carries Forward And What Does Not

The teaching backend is not throwaway work. These parts should carry into a
production integration:

```text
typed resource IDs plus relation and action vocabulary
product-facing can_* action names
model rules and relationship semantics
application enforcement points
the narrow AuthorizationService boundary
allow, deny, and error behavior
the canonical action contract
```

These parts are deliberately replaced:

```text
GraphEvaluator              -> OpenFGA query engine
InMemoryStore               -> durable OpenFGA datastore
static Go policy tables     -> immutable deployed model IDs
fixture seeding             -> domain-owned relationship pipelines
human-readable DFS trace    -> metrics, decision IDs, and protected diagnostics
single-process consistency  -> documented OpenFGA consistency choices
```

Do not turn `GraphEvaluator` into a production engine one feature at a time.
Adding durable indexes, distributed consistency, cache invalidation, policy
migrations, intersections, exclusions, listing algorithms, tenancy controls,
and operations would mean building an authorization platform. Keep extending it
only when a small teaching example needs to expose a concept.

## Mapping

| Go concept | OpenFGA concept |
|---|---|
| `rebac.Relationship` | relationship tuple |
| `rebac.Resource` | object ID, e.g. `document:roadmapDocument` |
| `rebac.Action` | computed relation used in an OpenFGA Check |
| `rebac.Subject` with `#` | subject set, e.g. `team:platformTeam#member` |
| `internal/authz/model.go` | authorization model DSL |
| `authz.InMemoryStore` | OpenFGA tuple store |
| `authz.GraphEvaluator` | OpenFGA check engine |
| `openfga.Service` | concrete SDK-backed authorization service |

## Model

OpenFGA policy lives in:

```text
deployments/openfga/model.fga
```

The Go mirror lives in:

```text
internal/authz/model.go
```

The contract tests keep those meanings aligned.

OpenFGA models are immutable. Writing a model creates a new model ID. The
adapter pins a configured model ID so a new upload cannot silently alter
production decisions.

## Bootstrapping

The model test runs through a pinned OpenFGA CLI container. The local seed
script runs on your host and additionally requires:

```bash
fga version
jq --version
```

Install the OpenFGA CLI using the
[official instructions](https://openfga.dev/docs/getting-started/cli) and
install `jq` with your platform's package manager before running the seed
script. The CLI is not needed on the host for `make openfga/model-test`.

```bash
make openfga/up
make openfga/model-test
make openfga/seed
make server-openfga
```

`openfga/model-test` runs the executable action matrix in
`deployments/openfga/model.fga.yaml`. It does not require a running server.

`openfga/seed` does four things:

1. creates an OpenFGA store
2. uploads `model.fga`
3. writes demo workspace/team policy tuples
4. writes generated IDs to `deployments/openfga/.ids.env`

The Go server creates the demo document at startup. That writes the document's
runtime tuples through the selected service's `WriteRelationships`, so in OpenFGA mode
they land in the OpenFGA tuple store.

## OpenFGA Tuple Split

Bootstrap tuples:

```text
user:alice                  member  team:platformTeam
team:platformTeam#member    editor  workspace:productWorkspace
user:bob                    viewer  workspace:productWorkspace
```

Runtime document tuples:

```text
workspace:productWorkspace  workspace  document:roadmapDocument
user:alice                  owner      document:roadmapDocument
```

The local OpenFGA container uses an in-memory datastore, so restart means reseed.

## Prove Parity

Run the in-process contract normally:

```bash
go test ./internal/authz
```

After seeding a fresh OpenFGA store, source the generated IDs and run:

```bash
set -a
. deployments/openfga/.ids.env
set +a
go test -run TestContract_OpenFGA ./internal/openfga
```

Both backends should satisfy the same allow/deny matrix. That behavioral
contract—not similar-looking code—is the meaningful migration guarantee.

Parity for this small matrix does not prove feature parity. Evaluate work
requirements such as:

- tenant isolation
- deeply nested groups
- intersections or exclusions
- contextual or conditional access
- action-aware listing and search
- immediate grant/revocation behavior
- policy migrations
- backend outages and latency budgets

The contract protects policy meaning; it does not certify the deployment,
relationship pipeline, identity mapping, or operational controls.

## Consistency

The in-memory evaluator reads its current process state under a lock. OpenFGA
queries support consistency preferences:

```text
MINIMIZE_LATENCY    may use configured caches
HIGHER_CONSISTENCY  skips the query cache and reads the database
```

The adapter does not yet expose this option. A production integration must
choose a consistency preference by operation, especially immediately after
relationship grants and revocations.

Do not assume OpenFGA implements Zanzibar zookies. Current OpenFGA
documentation describes a zookie-like consistency token as future work.

## The Production Migration Ladder

Treat these as gates. Do not skip from “the local adapter works” to “OpenFGA is
the production decision maker.”

### Stage 0: Freeze the decision contract

Write down:

- stable subject, resource, relation, and action names
- which business operation checks each `can_*` action
- representative direct and inherited allows
- near-miss, unrelated-user, revocation, and cross-tenant denies
- the distinction between policy denial and engine failure

The repository's `internal/authz/contract` package is the small example. At
work, the matrix should come from product and security requirements rather than
from the implementation.

Exit gate: reviewers can predict every contract result from product sentences.

### Stage 1: Model real workflows

Model two or three real workflows, including a difficult one. Include tenant
boundaries, nested groups, resource inheritance, custom sharing, or contextual
rules if the product needs them. Decide which relations may appear in stored
relationships and which are purely derived, and which `can_*` names are domain
actions represented as computed relations in OpenFGA. Only concrete
relationships are written as facts.

Run OpenFGA model tests for allows and denies. Check the required listing and
search flows too; a fast `Check` does not prove `ListObjects` or `ListUsers`
will meet product requirements.

Exit gate: the model expresses real policy without application code recreating
its derivation rules.

### Stage 2: Stabilize the application boundary

Application use cases should depend on a small capability such as:

```text
Check(subject, action, resource)
WriteRelationships(...)
DeleteRelationships(...)
```

The documents service already does this. A larger organization may place a
domain-specific authorization service in front of OpenFGA, or let services use
an SDK through a shared library. Either choice should keep OpenFGA transport
types out of business rules and keep action names stable.

Exit gate: changing the concrete backend does not change domain or handler
logic.

### Stage 3: Design relationship ownership and delivery

For every writable relation, record:

| Question | Example answer |
|---|---|
| source of truth | team service owns `team#member` |
| creation event | membership accepted |
| deletion event | membership removed |
| delivery | transactional outbox and idempotent consumer |
| reconciliation | compare source records with OpenFGA tuples |
| freshness objective | revocation visible within the agreed SLO |

Avoid an untracked synchronous dual write as the final design. Use durable
events, idempotent tuple operations, retries, a dead-letter/recovery process,
and periodic reconciliation. Define how initial backfill and object deletion
work.

Exit gate: every tuple has one authoritative owner and a recoverable path into
OpenFGA.

### Stage 4: Build the production OpenFGA environment

The local in-memory Compose service is not this environment. Production needs:

- a pinned, supported OpenFGA version
- a migrated and dedicated PostgreSQL or MySQL datastore
- authenticated clients and HTTP or gRPC TLS
- the playground disabled
- secret rotation and least-privilege network access
- structured logs, metrics, sampled tracing, dashboards, and alerts
- database pool, query concurrency, depth, breadth, and result limits
- capacity, backup, restore, upgrade, and rollback procedures

Use the current official
[production guide](https://openfga.dev/docs/best-practices/running-in-production)
for exact server settings; they change more frequently than this course.

Exit gate: load, failure, restore, and security tests pass in a production-like
environment.

### Stage 5: Backfill and validate relationship data

Upload the new
[immutable model](https://openfga.dev/docs/getting-started/immutable-models)
without making it authoritative. Pin its model ID. Backfill relationships from
domain sources, then reconcile counts and sampled facts. Exercise contract
checks against representative staged data.

If a model change renames or reshapes relations, use an explicit
expand–migrate–contract sequence: temporarily accept old and new shapes,
migrate tuples and callers, then remove the old shape. Do not overwrite a model
mentally—OpenFGA models are immutable and tuple migrations may need to coexist
with old and new readers.

Exit gate: relationship completeness and model validity are measured, not
assumed.

### Stage 6: Shadow decisions

Keep the existing authorization path authoritative. Send a sampled copy of
real checks to OpenFGA asynchronously and compare:

```text
subject, action, resource
old decision and error
OpenFGA decision and error
old and new policy/model identity
latency and consistency mode
```

Never let a shadow timeout delay or alter the live response. Protect or hash
sensitive identifiers in comparison telemetry. Investigate every unexplained
allow/deny mismatch; an unexpected allow is especially important.

In this repository the teaching evaluator can shadow only the subset supported
by both backends. At work, shadow the actual existing authorization system, not
a newly invented toy evaluator.

Exit gate: the agreed mismatch, error, and latency thresholds hold for a
representative period and every remaining difference is intentional.

This follows OpenFGA's current
[shadow-adoption guidance](https://openfga.dev/docs/best-practices/adoption-patterns).

### Stage 7: Canary and cut over decision authority

Move a small, observable cohort or low-risk operation to OpenFGA. Keep a fast
rollback to the previous authority. Increase traffic gradually while watching
decision errors, latency, mismatch sampling, tuple lag, and business-denial
rates.

Do not “fail open to the old engine” indefinitely without a written policy: two
authoritative engines can disagree after policy or data changes. A rollback is
a controlled operational change, not a hidden per-request authorization bypass.

Exit gate: OpenFGA is the documented authority, the fallback policy is explicit,
and the old decision path can be retired safely.

### Stage 8: Operate and evolve

For every production change:

1. write or update the action contract
2. create a new immutable model
3. test old and new behavior
4. migrate tuples or callers when required
5. shadow the new model ID
6. canary and progressively activate it
7. retain a tested rollback plan

Choose `MINIMIZE_LATENCY` or `HIGHER_CONSISTENCY` per workflow. In particular,
test read-after-write grants and security-sensitive revocations with the cache
configuration actually used in production.

Exit gate: authorization has owners, SLOs, on-call diagnostics, reconciliation,
and a repeatable model-release process.

## Definition Of Done

The production engine is not done merely because `Check` returns the expected
boolean. Before cutover, be able to answer yes to all of these:

- Are identity mapping and tenant boundaries validated before every check?
- Does application code ask stable actions rather than duplicate policy?
- Does every relationship have an authoritative source and reconciliation?
- Are model IDs pinned and migrated through a controlled pipeline?
- Are allow, deny, error, revocation, and cross-tenant cases tested?
- Are listing/search behavior and operational limits tested at realistic scale?
- Are consistency and cache staleness chosen per sensitive workflow?
- Does an OpenFGA outage fail closed without being misreported as a normal deny?
- Are OpenFGA authentication, TLS, secrets, network policy, and datastore ready?
- Can the team observe, roll back, restore, upgrade, and investigate decisions?

If any answer is no, the migration has a named remaining task rather than a
hidden assumption.

## Recommended Work Prototype

Before selecting the engine at work:

1. model real workflows and write their action contract
2. prototype source-of-truth relationship delivery and reconciliation
3. measure Check and listing at representative depth and cardinality
4. test revocation freshness, outages, and model rollout
5. document every capability gap and custom component still required
6. use shadow mode before transferring decision authority

Read [Designing a ReBAC authorization service](07-rebac-authorization-service-design.md)
for the broader architecture and evaluation checklist, then use
[Production readiness](40-production-readiness.md) as the final operational
gate.
