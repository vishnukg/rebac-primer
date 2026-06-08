# OpenFGA adapter: how a check flows when AUTHZ_BACKEND=openfga

Docs 28 and 29 trace a request through the **from-scratch graph evaluator**.
This chapter traces the same request through the **OpenFGA adapter** — the
alternative backend you select with `AUTHZ_BACKEND=openfga`. Read it after doc 26
(the concept mapping + the flag) and alongside the adapter source:

- Go: `go/internal/openfga/openfga.go`
- TS: `typescript/src/authz-service/adapters/openfga/makeOpenFgaAuthzService.ts`
- Model: `deployments/openfga/model.fga` · Seed: `deployments/openfga/seed.sh`

## The one thing that changes

The whole point of the ports design: **only the authz backend changes.** The
HTTP handlers, the documents domain, authn, and the tests are untouched. The
adapter implements the same driving port the graph version does:

```text
authz.Service (Go) / AuthzService (TS):  Check · WriteTuples · DeleteTuples · ListTuples
```

| | Graph backend (default) | OpenFGA backend (`AUTHZ_BACKEND=openfga`) |
|---|---|---|
| Who answers `Check` | `authz.GraphEvaluator` traverses tuples in memory | OpenFGA server's Check API |
| Where tuples live | in-memory `TupleRepository` | the OpenFGA store |
| Where the model lives | Go maps / TS tables (`permissionmodel.*`) | DSL uploaded to the store (`model.fga`) |
| `Check` result trace | full step-by-step trace | one synthetic line (OpenFGA returns only allow/deny) |

> Why the adapter implements the **`Service`** port, not the inner `Evaluator`
> port: `Evaluator` only covers checks, and the in-memory `TupleRepository.Write`
> is synchronous with no `ctx`/error — a poor fit for a network backend.
> `authz.Service` has `ctx` + error on every method, so checks *and* tuple writes
> both go to OpenFGA and stay consistent. (Doc 26 explains this in full.)

## Setup (one time)

```bash
make openfga/up     # OpenFGA on :8080 (ephemeral memory datastore)
make openfga/seed   # create store, write model.fga, seed the policy tuples
                    # → writes deployments/openfga/.ids.env (store + model IDs)
```

`seed.sh` writes the workspace/team **policy** tuples (the ones
`fixtures.SeedRelationshipTuples` / `seedPolicyTuples` hold for the in-memory
backend). The **document** tuples are written at runtime by the documents service
on create — they just land in OpenFGA now.

## The call flow — "can Bob edit the roadmap?"

`PATCH /documents/roadmapDocument` as Bob, with the OpenFGA backend selected.

```text
client ─► documents service ─► documents domain (Update)
                                     │ authzClient.Check(user:bob, can_edit, document:roadmapDocument)
                                     ▼
                          OpenFGA adapter (authz backend)
                                     │ s.client.Check(ctx).Body({user, relation, object})
                                     ▼  HTTP/gRPC
                          OpenFGA server  ── evaluates model.fga + stored tuples
                                     │ { allowed: false }
                                     ▼
                          adapter → CheckResult{ Allowed: false }
                                     │
              documents domain: !allowed → ForbiddenError ─► HTTP 403
```

Step by step:

1. The documents domain calls `authzClient.Check(...)` — exactly as in the graph
   build. In Go that's an in-process call to the OpenFGA adapter; in TS the
   documents service calls the authz service over HTTP, and *the authz service's*
   backend is the OpenFGA adapter.
2. The adapter's `Check` maps the repo's `CheckRequest` to the SDK's check call:

   ```go
   // Go — openfga.go
   resp, err := s.client.Check(ctx).Body(openfga.ClientCheckRequest{
       User: string(req.User), Relation: string(req.Relation), Object: string(req.Object),
   }).Execute()
   return rebac.CheckResult{Allowed: resp.GetAllowed(), Trace: [...]}, nil
   ```
   ```ts
   // TS — makeOpenFgaAuthzService.ts
   const { allowed } = await client.check({ user, relation, object });
   return { allowed: allowed === true, trace: [...] };
   ```
3. OpenFGA evaluates the relationship graph **server-side** against the uploaded
   model and stored tuples — the same traversal `evaluator.go` does in process —
   and returns `allowed: false` (Bob is a viewer, not an editor).
4. Back in the documents domain, `!allowed` becomes a `ForbiddenError`, which the
   HTTP adapter maps to **403** — identical to the graph backend.

The mirror case (`document.create` writing tuples) goes the other way:
`authzClient.WriteTuples` → the adapter's `WriteTuples` → OpenFGA `Write` API, so
the new document's `workspace`/`owner` tuples are persisted in the store and
visible to the next check.

## Method mapping

| Port method | Go SDK call | TS SDK call |
|---|---|---|
| `Check` | `client.Check(ctx).Body(ClientCheckRequest{...})` | `client.check({user, relation, object})` |
| `WriteTuples` | `client.Write(ctx).Body(ClientWriteRequest{Writes})` | `client.writeTuples([...])` |
| `DeleteTuples` | `client.Write(ctx).Body(ClientWriteRequest{Deletes})` | `client.deleteTuples([...])` |
| `ListTuples` | `client.Read(ctx).Body(ClientReadRequest{...})` | `client.read({...})` |

## Run it and compare

```bash
make go/server-openfga      # or: make ts/server-openfga

# Bob can read (viewer) → 200
curl localhost:4001/documents/roadmapDocument -H "Authorization: Bearer demo-token-bob"
# Bob cannot edit → 403
curl -X PATCH localhost:4001/documents/roadmapDocument \
  -H "Authorization: Bearer demo-token-bob" -H "content-type: application/json" -d '{"body":"no"}'
```

The responses match the graph backend exactly — that's the proof the swap is
transparent. The one visible difference is the `trace` field: graph mode returns
the full traversal; OpenFGA mode returns a single synthetic line, because the
Check API reports only allow/deny.

## Checkpoint

1. Which port does the OpenFGA adapter implement, and why that one rather than
   `Evaluator`?
2. After the swap, where does `document.create`'s `owner` tuple end up, and how
   does a later `can_delete` check see it?
3. Why is the `trace` shorter in OpenFGA mode?

Good answers:
1. `authz.Service` / `AuthzService` — it has `ctx` + error on every method, so
   both checks and tuple writes can go to the network backend. `Evaluator` only
   covers checks and the in-memory `Write` has no `ctx`/error.
2. `WriteTuples` sends it to the OpenFGA store via the Write API; a later
   `can_delete` check is answered by OpenFGA against that stored tuple.
3. OpenFGA's Check API returns only `allowed`; it does not stream the per-step
   traversal the in-process evaluator builds, so the adapter emits one line.
