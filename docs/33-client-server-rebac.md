# Client/Server ReBAC Demo

The Go server exposes the document service over HTTP.

## Run

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
