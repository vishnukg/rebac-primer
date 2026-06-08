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
ListTuples
```

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

## WriteTuples

When a document is created, the documents service writes document-level
relationship facts. In OpenFGA mode, `WriteTuples` sends those facts to the
OpenFGA Write API.

That is why a later `can_delete` check can see that Alice owns the document.

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
