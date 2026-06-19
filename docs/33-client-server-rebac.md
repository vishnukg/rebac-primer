# HTTP Boundaries: Product API and Authz API

This repository contains two different HTTP examples:

1. `cmd/server` exposes the document product API and uses authorization
   internally.
2. `examples/authzhttp` exposes the authorization service itself, demonstrating
   the seam used when authz becomes a separate service.

Do not confuse them. The product API is runnable; the authz HTTP package is a
tested teaching adapter and has no standalone command.

## Run the Product API

```bash
make server
```

Server:

```text
http://127.0.0.1:4001
```

## Endpoints

```text
GET   /health
GET   /whoami
POST  /documents
GET   /documents/{id}
PATCH /documents/{id}
```

## Try It

Bob can read:

```bash
curl "http://127.0.0.1:4001/documents/roadmapDocument" \
  -H "Authorization: Bearer demo-token-bob"
```

Bob cannot edit:

```bash
curl -X PATCH "http://127.0.0.1:4001/documents/roadmapDocument" \
  -H "Authorization: Bearer demo-token-bob" \
  -H "content-type: application/json" \
  -d '{"body":"no"}'
```

Alice can edit:

```bash
curl -X PATCH "http://127.0.0.1:4001/documents/roadmapDocument" \
  -H "Authorization: Bearer demo-token-alice" \
  -H "content-type: application/json" \
  -d '{"body":"updated"}'
```

Who am I?

```bash
curl "http://127.0.0.1:4001/whoami" \
  -H "Authorization: Bearer demo-token-alice"
```

## Flow

```text
client -> internal/api -> internal/documents -> internal/authz -> tuple graph
```

With `AUTHZ_BACKEND=openfga`, the last hop becomes:

```text
internal/documents -> internal/openfga -> OpenFGA server
```

## Inspect the Authz-Service Seam

`examples/authzhttp` defines:

```text
POST   /check
POST   /tuples
DELETE /tuples
GET    /tuples
```

Run its integration tests:

```bash
go test -v ./examples/authzhttp
```

Those tests exercise HTTP decoding, tuple validation, writes, revocation, and
permission checks over the real in-process authorization service.

In production, exposing tuple mutation is a privileged administrative API. It
requires strong service authentication, authorization, audit logging, request
limits, and careful ownership rules. The example intentionally focuses only on
the client/server shape.
