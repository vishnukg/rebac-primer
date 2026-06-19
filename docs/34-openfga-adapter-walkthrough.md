# OpenFGA Adapter Walkthrough

Read this with `internal/openfga/openfga.go` open.

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
WriteTuples
DeleteTuples
```

Those methods form `documents.AuthorizationService`, an interface owned by the
consumer. The authz HTTP example has a separate interface that also includes
`ListTuples`.

## Check

`Check` maps the app request into the OpenFGA SDK request:

```go
resp, err := s.client.Check(ctx).Body(openfga.ClientCheckRequest{
    User:     string(req.User),
    Relation: string(req.Relation),
    Object:   string(req.Object),
}).Execute()
```

OpenFGA evaluates `model.fga` plus stored tuples and returns allow/deny.

The adapter validates the check shape before making the network call, matching
the in-process service's behavior.

## WriteTuples

When a document is created, the documents service writes document-level
relationship facts. In OpenFGA mode, `WriteTuples` sends those facts to the
OpenFGA Write API.

That is why a later `can_delete` check can see that Alice owns the document.

The adapter pins an authorization model ID. That avoids silently changing check
semantics when a newer model is deployed. Its read-before-write duplicate check
is intentionally simple and not atomic; production event consumers should use
idempotency and retry policy at the workflow level.

## Read and Pagination

OpenFGA's Read API is paginated. `ListTuples` follows continuation tokens until
all matching pages are collected. Missing this loop would silently return a
partial tuple set and could break duplicate detection or cleanup.

The method supports consumers such as the authz HTTP example. Production
applications should prefer purpose-built OpenFGA query APIs for authorization
questions and avoid treating tuple reads as a general listing/search API.

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
