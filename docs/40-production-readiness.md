# Production Readiness

This repo is a primer, not a production service. The OpenFGA adapter is the
production direction, but several demo components are intentionally simple.

## Replace For Production

| Area | Primer | Production |
|---|---|---|
| Authn | static demo bearer tokens | OIDC login plus access-token validation for the documented token format |
| OAuth scopes | demo scopes enforced by handlers | define an API scope policy and validate issuer, audience, lifetime, signature, and scopes before ReBAC |
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
5. test models and expected allow/deny behavior before deployment
6. choose and document the required consistency behavior per operation
7. authenticate and authorize access to OpenFGA itself
8. page tuple reads; do not treat `Read` as a bulk export API

The Compose file pins OpenFGA for reproducible learning. Upgrade deliberately,
read migration notes, and avoid `latest` in deployed environments.

## Security Notes

Authorization should fail closed. If OpenFGA is unavailable, sensitive operations
should deny or return a server error rather than allow.

The tutorial currently distinguishes not-found from forbidden. In higher
security systems, consider returning the same response for both to avoid leaking
which document IDs exist.

Relationship tuples are sensitive data because they reveal organization
structure. Treat tuple reads and logs accordingly.

Document creation spans a document store and an authorization store. The primer
uses compensating cleanup. Production systems normally use an outbox/domain
event and idempotent consumers so failed tuple writes are retried reliably.

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
go tool staticcheck ./...
go fix -diff ./...
go test -race ./...
```

Also run `govulncheck ./...` in CI using the official Go vulnerability tool.

## Current References

- [OAuth 2.0 Security Best Current Practice (RFC 9700)](https://www.rfc-editor.org/rfc/rfc9700)
- [OpenFGA: testing authorization models](https://openfga.dev/docs/modeling/testing)
- [OpenFGA: running in production](https://openfga.dev/docs/best-practices/running-in-production)
- [Go vulnerability management](https://go.dev/doc/security/vuln/)
