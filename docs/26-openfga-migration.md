# OpenFGA Versus a Custom ReBAC Engine

The repo has two authz backends:

1. an in-process graph evaluator, which exposes the core mechanics
2. an OpenFGA adapter, which delegates evaluation and tuple storage

Both satisfy the `documents.AuthorizationService` interface. Other consumers,
such as the authz HTTP example, declare their own required interface.

This is a backend substitution, not proof that the surrounding demo is
production-ready. Token verification, durable document storage, OpenFGA
authentication, consistency choices, and operational controls remain separate
production concerns.

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

## Mapping

| Go concept | OpenFGA concept |
|---|---|
| `rebac.TupleKey` | relationship tuple |
| `rebac.Object` | object ID, e.g. `document:roadmapDocument` |
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

`openfga/model-test` runs the executable permission matrix in
`deployments/openfga/model.fga.yaml`. It does not require a running server.

`openfga/seed` does four things:

1. creates an OpenFGA store
2. uploads `model.fga`
3. writes demo workspace/team policy tuples
4. writes generated IDs to `deployments/openfga/.ids.env`

The Go server creates the demo document at startup. That writes the document's
runtime tuples through the selected service's `WriteTuples`, so in OpenFGA mode they land
in the OpenFGA tuple store.

## Tuple Split

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
- permission-aware listing and search
- immediate grant/revocation behavior
- policy migrations
- backend outages and latency budgets

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

## Recommended Prototype

Before selecting the engine at work:

1. Model two or three real product workflows, not only a toy document.
2. Write allow and near-miss deny cases before implementing.
3. Implement the application-facing `Check` interface independently of OpenFGA.
4. Run the policy contract against OpenFGA.
5. Prototype relationship writes from the actual source-of-truth services.
6. Measure Check and listing behavior at representative depth and cardinality.
7. Test revocation freshness, OpenFGA outage behavior, and model rollout.
8. Document every capability gap and every custom component still required.

Read [Designing a ReBAC authorization service](07-rebac-authorization-service-design.md)
for the broader architecture and evaluation checklist.
