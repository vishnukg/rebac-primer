# Production Readiness

This repo is a primer, not a production service. The OpenFGA adapter is the
production direction, but several demo components are intentionally simple.
Follow the staged migration in
[From the In-Process Evaluator to Production OpenFGA](26-openfga-migration.md)
before using this chapter as the final gate.

## What Production Means Here

There are three separate deliverables:

```text
correct policy       the model grants exactly the intended access
correct data         OpenFGA receives complete, fresh relationship facts
reliable operation   checks remain secure and observable under load and failure
```

Passing model tests proves only the first for the cases tested. An SDK adapter
does not prove relationship completeness or operational readiness.

## Replace For Production

| Area | Primer | Production |
|---|---|---|
| Authn | static demo bearer tokens | OIDC login plus access-token validation for the documented token format |
| OAuth scopes | demo scopes enforced by handlers | define an API scope policy and validate issuer, audience, lifetime, signature, and scopes before ReBAC |
| Document storage | in-memory repository | durable database |
| Authz backend | graph evaluator by default | pinned OpenFGA service with durable datastore |
| Policy deployment | local seed script | migration/deployment pipeline |
| Observability | basic logs | structured logs, metrics, tracing, alerts |
| Secrets/config | local env vars | secret manager and validated config |
| Relationship delivery | fixtures and synchronous writes | domain ownership, durable events, retries, and reconciliation |
| Consistency | current process state | per-workflow freshness and cache policy |
| Failure handling | local errors | fail-closed policy, timeouts, alerting, and recovery |

## OpenFGA

For production:

1. run a pinned OpenFGA release with a dedicated PostgreSQL or MySQL datastore
2. version `deployments/openfga/model.fga`
3. deploy model changes through a controlled pipeline
4. write relationship tuples from domain events
5. test models and expected allow/deny behavior before deployment
6. pin the intended immutable authorization model ID on queries and writes
7. choose and document the required consistency behavior per operation
8. authenticate and authorize access to OpenFGA itself
9. page tuple reads; do not treat `Read` as effective-access enumeration
10. bound and monitor Check, ListObjects, and ListUsers resolution
11. disable the playground and enable HTTP or gRPC TLS
12. tune database pools, cache policy, concurrency, depth, breadth, and result limits from measured load
13. test database migration, backup, restore, server upgrade, and rollback

The Compose file pins OpenFGA for reproducible learning. Upgrade deliberately,
read migration notes, and avoid `latest` in deployed environments.

OpenFGA authentication protects access to the API; it does not automatically
decide which authenticated client may administer each store or tuple type. The
built-in fine-grained access-control feature is currently documented as
experimental, so design the production control plane using supported controls
and current OpenFGA guidance.

## Security Notes

Authorization should fail closed. If OpenFGA is unavailable, sensitive operations
should deny or return a server error rather than allow.

The tutorial currently distinguishes not-found from forbidden. In higher
security systems, consider returning the same response for both to avoid leaking
which document IDs exist.

Relationship tuples are sensitive data because they reveal organization
structure. Treat tuple reads and logs accordingly.

Use opaque identifiers in tuples. Do not store email addresses, names, or other
personal and regulated data in relationship keys.

Document creation spans a document store and an authorization store. The primer
uses compensating cleanup. Production systems normally use an outbox/domain
event and idempotent consumers so failed OpenFGA tuple writes are retried
reliably.

Treat a successful policy denial and an indeterminate engine failure as
different outcomes. Both may block the operation, but only the latter should
drive availability alerts and retry/circuit-breaker behavior.

## Cutover Gates

Do not make OpenFGA authoritative until:

- the same action contract passes for the old and new paths
- real relationship data has been backfilled and reconciled
- shadow traffic has no unexplained decision differences
- read-after-write and revocation freshness meet their SLOs
- Check and listing latency are measured at realistic graph depth and breadth
- authentication, TLS, datastore, capacity, and failure tests pass
- a canary, rollback, and model-migration procedure has been rehearsed

After cutover, retire the previous decision path deliberately. Leaving two
silent authorities in place creates an ambiguous failure policy.

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
go tool govulncheck ./...
```

The last command uses the official Go vulnerability tool pinned by `go.mod`.

Test the OpenFGA model itself:

```bash
make openfga/model-test
```

## Current References

- [OAuth 2.0 Security Best Current Practice (RFC 9700)](https://www.rfc-editor.org/rfc/rfc9700)
- [OpenFGA: testing authorization models](https://openfga.dev/docs/modeling/testing)
- [OpenFGA: relationship query APIs](https://openfga.dev/docs/interacting/relationship-queries)
- [OpenFGA: query consistency](https://openfga.dev/docs/interacting/consistency)
- [OpenFGA: running in production](https://openfga.dev/docs/best-practices/running-in-production)
- [OpenFGA: adoption and shadowing patterns](https://openfga.dev/docs/best-practices/adoption-patterns)
- [OpenFGA: immutable models](https://openfga.dev/docs/getting-started/immutable-models)
- [OpenFGA: model migrations](https://openfga.dev/docs/modeling/migrating)
- [Zanzibar paper](https://www.usenix.org/conference/atc19/presentation/pang)
- [Go vulnerability management](https://go.dev/doc/security/vuln/)
