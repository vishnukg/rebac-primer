# Production Readiness

This repo is a primer, not a production service. The OpenFGA adapter is the
production direction, but several demo components are intentionally simple.

## Replace For Production

| Area | Primer | Production |
|---|---|---|
| Authn | static demo bearer tokens | JWT/OIDC verification with issuer, audience, expiry, JWKS |
| OAuth scopes | carried on the token but never enforced | reject requests whose token lacks the scope for the endpoint, then run the object-level ReBAC check |
| Document storage | in-memory repository | durable database |
| Authz backend | graph evaluator by default | OpenFGA service with durable datastore |
| Policy deployment | local seed script | migration/deployment pipeline |
| Observability | basic logs | structured logs, metrics, tracing, alerts |
| Secrets/config | local env vars | secret manager and validated config |

## OpenFGA

For production:

1. run OpenFGA with PostgreSQL or MySQL
2. version `deployments/openfga/model.fga`
3. deploy model changes through a controlled pipeline
4. write relationship tuples from domain events
5. keep contract tests for expected allow/deny behavior

## Security Notes

Authorization should fail closed. If OpenFGA is unavailable, sensitive operations
should deny or return a server error rather than allow.

The tutorial currently distinguishes not-found from forbidden. In higher
security systems, consider returning the same response for both to avoid leaking
which document IDs exist.

Relationship tuples are sensitive data because they reveal organization
structure. Treat tuple reads and logs accordingly.

## Test Strategy

Keep these test layers:

```text
unit tests         -> pure parsing, stores, service behavior
contract tests     -> canonical allow/deny matrix
adapter tests      -> OpenFGA request/response mapping
integration tests  -> HTTP request through authn, documents, authz
race tests         -> in-memory concurrency safety
```

Run before shipping:

```bash
go test ./...
go vet ./...
go test -race ./...
```
