# From Scratch to OpenFGA

The repo has two authz backends:

1. in-process graph evaluator, for learning
2. OpenFGA adapter, showing the external authorization-service direction

Both satisfy the same app-facing `authz.Service` shape.

This is a backend substitution, not proof that the surrounding demo is
production-ready. Token verification, durable document storage, OpenFGA
authentication, consistency choices, and operational controls remain separate
production concerns.

## Mapping

| Go concept | OpenFGA concept |
|---|---|
| `rebac.TupleKey` | relationship tuple |
| `rebac.Object` | object ID, e.g. `document:roadmapDocument` |
| `rebac.Subject` with `#` | subject set, e.g. `team:platformTeam#member` |
| `internal/authz/model.go` | authorization model DSL |
| `authz.InMemoryStore` | OpenFGA tuple store |
| `authz.GraphEvaluator` | OpenFGA check engine |
| `openfga.Service` | SDK-backed `authz.Service` |

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

## Bootstrapping

```bash
make openfga/up
make openfga/seed
make server-openfga
```

`openfga/seed` does four things:

1. creates an OpenFGA store
2. uploads `model.fga`
3. writes demo workspace/team policy tuples
4. writes generated IDs to `deployments/openfga/.ids.env`

The Go server creates the demo document at startup. That writes the document's
runtime tuples through `authzService.WriteTuples`, so in OpenFGA mode they land
in the OpenFGA tuple store.

## Tuple Split

Bootstrap tuples:

```text
(team:platformTeam, member, user:alice)
(workspace:productWorkspace, editor, team:platformTeam#member)
(workspace:productWorkspace, viewer, user:bob)
```

Runtime document tuples:

```text
(document:roadmapDocument, workspace, workspace:productWorkspace)
(document:roadmapDocument, owner, user:alice)
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
