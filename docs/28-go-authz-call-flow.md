# Go Authz Call Flow

This traces:

```text
GET /documents/roadmapDocument
Authorization: Bearer demo-token-bob
```

## 0. Wiring

`cmd/server/main.go` wires:

```text
token verifier
document repository
authz service
documents service
HTTP handler
```

Default mode:

```text
authz.New(InMemoryStore, GraphEvaluator)
```

OpenFGA mode:

```text
openfga.New(...)
```

Both are passed to `documents.New(...)` through the same
`documents.AuthorizationService` interface.

## 1. HTTP Route

`internal/api/server.go` registers:

```go
mux.HandleFunc("GET /documents/{id}", h.handleGetDocument)
```

The handler extracts the path ID and the `Authorization` header.

## 2. Authentication

`internal/documents/token.go` parses:

```text
Authorization: Bearer demo-token-bob
```

The demo verifier returns:

```text
AuthenticatedUser{
  Subject: "user:bob",
  Scopes:  ["documents:read"],
}
```

This establishes who is asking. It does not decide what Bob can do.

## 3. OAuth Scope Gate

The handler requires `documents:read`. This is the coarse API permission:

```text
Does this token permit calls to the document-read endpoint?
```

If the scope is absent, the handler returns 403 with an
`insufficient_scope` bearer challenge. Passing this gate does not grant access
to any specific document.

## 4. Document Use Case

`handleGetDocument` calls:

```go
h.docs.Read(r.Context(), "roadmapDocument", user.Subject)
```

`documents.Read` loads the document and then requires:

```text
user:bob can_read document:roadmapDocument
```

## 5. Authorization Boundary

`documents.requireAllowed` calls:

```go
s.authzClient.Check(ctx, rebac.CheckRequest{
    User:     actor,
    Relation: rebac.RelationDocumentCanRead,
    Object:   rebac.Document(id),
})
```

If `Allowed` is false, it returns `ForbiddenError`.

## 6. Graph Evaluation

Default mode uses `internal/authz/evaluator.go`.

For Bob reading the roadmap document, the successful path is:

```text
document:roadmapDocument
  --workspace--> workspace:productWorkspace
  --viewer--> user:bob
```

The document model says:

```text
can_read <- viewer
```

So Bob can read.

## 7. Response

Allowed:

```text
HTTP 200 {"document": ...}
```

Denied:

```text
HTTP 403 {"error": "..."}
```

Missing/invalid token:

```text
HTTP 401
```

Missing document:

```text
HTTP 404
```

Unsupported JSON media type and oversized bodies are rejected at the HTTP
boundary with 415 and 413 respectively, before domain logic runs.

## Try It

```bash
make server

curl :4001/documents/roadmapDocument \
  -H "Authorization: Bearer demo-token-bob"

curl -X PATCH :4001/documents/roadmapDocument \
  -H "Authorization: Bearer demo-token-bob" \
  -H "content-type: application/json" \
  -d '{"body":"x"}'
```

Bob's PATCH is rejected at the scope gate because his demo token has only
`documents:read`. The HTTP tests also include a Bob token with
`documents:write` to prove that ReBAC independently denies a viewer who reaches
the object-level check.
