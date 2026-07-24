# OpenFGA Adapter Walkthrough

Read this with `internal/openfga/openfga.go` open.

This chapter explains the code seam, not the whole production migration. The
adapter proves that application use cases can keep a stable interface while the
backend changes. Backfill, relationship ownership, shadow decisions, canary
cutover, and operations are separate stages in
[doc 26](26-openfga-migration.md).

## What Changes

Default graph backend:

```text
documents -> authz.Service -> GraphEvaluator -> InMemoryStore
```

OpenFGA backend:

```text
documents -> openfga.Service -> OpenFGA server
```

The documents service still calls the same methods:

```text
Check
WriteRelationships
DeleteRelationships
```

Those methods form `documents.AuthorizationService`, an interface owned by the
consumer. The authz HTTP example has a separate interface that also includes
`ListRelationships`.

## Check

`Check` maps the app request into the OpenFGA SDK request:

```go
resp, err := s.client.Check(ctx).Body(openfga.ClientCheckRequest{
    User:     string(req.Subject),
    Relation: string(req.Permission),
    Object:   string(req.Resource),
}).Execute()
```

This is the explicit domain-to-OpenFGA translation:
`subject/permission/resource` becomes `user/relation/object`. OpenFGA evaluates
`model.fga` plus stored tuples and returns allow/deny.

The adapter validates the check shape before making the network call, matching
the in-process service's behavior.

## WriteRelationships

When a document is created, the documents service writes document-level
relationship facts. In OpenFGA mode, `WriteRelationships` sends those facts to the
OpenFGA Write API.

That is why a later `can_delete` check can see that Alice owns the document.

The adapter pins an authorization model ID. That avoids silently changing check
semantics when a newer model is deployed. It also asks OpenFGA to ignore an
already-existing write, making the adapter's idempotent write contract atomic
without a read-before-write race. Deletes similarly ignore missing tuples.
Production event consumers still need retry and idempotency policy for the
larger business workflow.

## Read and Pagination

OpenFGA's Read API is paginated. `ListRelationships` follows continuation tokens until
all matching pages are collected. Missing this loop would silently return a
partial tuple set to administrative consumers.

One domain filter shape is rejected up front: relation without resource. The
OpenFGA Read API requires at least an object type alongside a relation, so the adapter
returns a clear error instead of forwarding a request the server would refuse.
The in-memory store does support relation-only filtering — a small reminder
that two backends satisfying the same interface can still differ at the edges.

The method supports consumers such as the authz HTTP example. Production
applications should prefer purpose-built OpenFGA query APIs for authorization
questions and avoid treating tuple reads as a general listing/search API.

`Read` returns stored tuples. It does not enumerate implied access produced by
the authorization model. OpenFGA separates effective-access queries:

```text
Check        one subject, permission (relation field), and resource (object field)
ListObjects  objects of a type related to one subject
ListUsers    subjects of a selected type related to one object
Expand       userset expression tree for one relation and object
```

This adapter intentionally exposes only permission checks and relationship
administration. Adding
listing requires product-specific pagination, result limits, latency budgets,
and search integration.

## OpenFGA Features Outside This Adapter

OpenFGA also supports:

- BatchCheck
- contextual tuples
- conditional relationships
- query consistency preferences
- intersections and exclusions in the model
- ListObjects, ListUsers, and Expand

The consumer-owned interface does not expose them because the current document
use cases do not require them. When evaluating OpenFGA for work, test the
features your real workflows need instead of judging it only through this
narrow adapter.

## Run

```bash
make openfga/up
make openfga/seed
make server-openfga
```

Bob can read:

```bash
curl localhost:4001/documents/roadmapDocument \
  -H "Authorization: Bearer demo-token-bob"
```

Bob cannot edit:

```bash
curl -X PATCH localhost:4001/documents/roadmapDocument \
  -H "Authorization: Bearer demo-token-bob" \
  -H "content-type: application/json" \
  -d '{"body":"no"}'
```

In the demo, Bob's token also lacks `documents:write`, so this request is denied
by the OAuth scope gate before ReBAC. The authorization contract tests separately
prove that Bob is not a document editor.
