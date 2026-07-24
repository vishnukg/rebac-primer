# Go ReBAC Implementation

This chapter maps the ReBAC concepts to the Go code.

## Layout

```text
cmd/server/          composition root
internal/rebac/      shared vocabulary
internal/authz/      authorization service and graph evaluator
internal/documents/  document service and authn/repository ports
internal/api/        HTTP adapter
internal/openfga/    OpenFGA-backed authz service
internal/fixtures/   demo subjects, relationships, tokens
```

## Vocabulary

Open `internal/rebac/rebac.go`.

```go
type Resource string
type Relation string
type Permission string
type Subject string
```

`Relationship` is one relationship fact:

```go
type Relationship struct {
    Subject  Subject
    Relation Relation
    Resource Resource
}
```

The fields read as: subject is a relation of resource.

At the OpenFGA boundary, the same relationship is conventionally presented as:

```text
user  relation  object
```

For example, the internal value:

```go
rebac.NewRelationship(
    rebac.Subject(rebac.User("alice")),
    rebac.RelationTeamMember,
    rebac.Team("platformTeam"),
)
```

maps to the OpenFGA tuple:

```text
user:alice  member  team:platformTeam
```

## Authz Service

Open `internal/authz/authz.go` and `internal/authz/service.go`.

`authz.Service` is the concrete in-process implementation:

```go
type Service struct {
    writer    RelationshipWriter
    lister    RelationshipLister
    evaluator Evaluator
}
```

Consumers declare narrow interfaces that `*authz.Service` satisfies implicitly.
The service delegates checks to an `Evaluator` and writes to a
`RelationshipWriter`. Listing uses `RelationshipLister`, while the graph evaluator depends
only on `RelationshipReader`. Checks and relationship writes are validated against
the known model before they reach a backend. This avoids turning caller
mistakes into silent denials or storing facts that can never match.

## Graph Evaluator

Open `internal/authz/evaluator.go`.

For each `Check(subject, permission, resource)`, the evaluator first maps the
permission to the relation required by policy. It then tries:

1. direct relationship
2. subject-set relationship
3. relation expansion from `internal/authz/model.go`
4. document workspace inheritance

`docs/27-graph-evaluator-walkthrough.md` traces those steps line by line.

Computed `can_*` permissions cannot be written as relationships because
`Permission` and `Relation` are separate domain types. For example, `can_edit`
maps to the `editor` relation and is then proved from the relationship graph.

The evaluator is intentionally a small subset, not a replacement OpenFGA
implementation. It demonstrates direct relationships, subject sets, unions,
same-resource relation expansion, and document-to-workspace inheritance. It does
not implement intersections, exclusions, conditions, contextual relationships,
wildcards, consistency controls, or production-scale query planning.

## Documents Service

Open `internal/documents/service.go`.

The document operations are:

```text
Create -> check can_create_document -> atomically create doc -> write document relationships
Read   -> load doc -> check can_read
Update -> load doc -> check can_edit -> save update
```

If relationship creation fails, `Create` performs compensating cleanup. This keeps the
example coherent without pretending two independent stores share a transaction.

The document service depends on two narrow ports from
`internal/documents/documents.go`:

```text
DocumentRepository
AuthorizationService
```

The HTTP adapter declares its own `DocumentService` and `Authenticator`
interfaces because it is the package that consumes those capabilities.

## HTTP Adapter

Open `internal/api/handler.go`.

The handler authenticates the bearer token, enforces the endpoint's coarse OAuth
scope, decodes bounded JSON, calls the documents service, and maps domain errors
to HTTP statuses. ReBAC remains the separate resource-level decision.

## Composition Root

Open `cmd/server/main.go`.

This is the only place that chooses concrete implementations. Default mode wires
the in-process graph evaluator. `AUTHZ_BACKEND=openfga` wires the OpenFGA
adapter instead.

## Run

```bash
make test
make server
```

Trace the evaluator:

```bash
go test -v -run TestTrace ./internal/authz
```

## Read It Actively

After each package, answer one question:

- `rebac`: what values can represent a graph edge?
- `authz`: where are policy facts stored, and where are rules stored?
- `documents`: which business operations require which permissions?
- `api`: which failures are authentication, scope, ReBAC, or malformed input?
- `cmd/server`: which concrete implementations are selected?

If you cannot answer one, return to that package before continuing.
