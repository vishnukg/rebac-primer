# Guided Go Feature Lab: Delete a Document

This lab turns the reading path into an end-to-end Go change. You will add a
document deletion operation through domain types, interfaces, service logic,
HTTP handling, authorization, and tests.

Do this after chapters 10–13, chapters 20–25, and the request-flow walkthrough.
The repository should pass before you start:

```bash
go test ./...
```

Commit or otherwise save your starting state. The lab intentionally asks you to
edit production code.

## Required Behavior

Add:

```text
DELETE /documents/{id}
```

Rules:

- the bearer token must be valid
- the token must contain `documents:write`
- the actor must have `can_delete` on the document
- a successful deletion returns HTTP 204
- deleting a missing document returns HTTP 404
- insufficient OAuth scope and failed ReBAC checks remain distinct cases
- document relationship tuples must be removed with the document

Before coding, write down which package owns each rule.

## 1. Start With a Failing Service Test

Open `internal/documents/service_test.go`. Add tests for:

1. an owner deleting an existing document
2. a viewer being denied
3. a missing document
4. an authorization backend error
5. tuple cleanup failure

Run only the new tests:

```bash
go test -run TestService_Delete ./internal/documents
```

They should fail to compile because `Delete` does not exist. A compile failure
is a valid first red test.

## 2. Extend the Consumer-Owned Interface

Open `internal/documents/documents.go`.

Determine whether the existing `AuthorizationService` already exposes the
operations deletion needs. If it does, do not add redundant methods.

The document repository already has a delete capability. Verify its semantics in
`internal/documents/store.go` and decide whether deletion should be idempotent at
the repository boundary or report not-found at the service boundary.

This step exercises a central Go design rule: interfaces are shaped by the
consumer's use case.

## 3. Implement the Use Case

Add a method with a shape similar to:

```go
func (s *Service) Delete(
    ctx context.Context,
    id string,
    actor rebac.Object,
) error
```

The method should:

1. load the document
2. check `can_delete`
3. remove document relationship tuples
4. delete the document

There is no transaction shared by the document repository and authorization
backend. Choose and document an ordering and failure policy. Consider:

- what happens if tuple deletion succeeds but document deletion fails?
- what happens if document deletion succeeds but tuple deletion fails?
- can compensating work restore the previous state?
- should cleanup use the canceled request context?

There is no universally correct answer for two independent stores. The expected
result is an explicit policy plus tests proving it.

Run:

```bash
go test ./internal/documents
```

## 4. Add the HTTP Port and Handler

Open `internal/api/handler.go` and `internal/api/server.go`.

Extend the API package's narrow `DocumentService` interface with only the method
the handler requires. Register:

```go
mux.HandleFunc("DELETE /documents/{id}", h.handleDeleteDocument)
```

The handler should follow the existing sequence:

```text
authenticate
require documents:write scope
extract path input
call document service
map domain error
write response
```

Do not duplicate authentication, scope, or error-mapping rules if an existing
helper already owns them.

## 5. Add HTTP Tests

In `internal/api/handler_test.go`, cover:

| Case | Expected status | Service called? |
|---|---:|---|
| missing token | 401 | no |
| missing write scope | 403 | no |
| ReBAC denial | 403 | yes |
| missing document | 404 | yes |
| success | 204 | yes |
| service failure | 500 | yes |

Assert more than status where it matters:

- the required bearer challenge for scope failures
- the document ID and actor passed to the service
- an empty body for HTTP 204

Run:

```bash
go test ./internal/api
```

## 6. Exercise It Through HTTP

Run the server:

```bash
go run ./cmd/server
```

Use a fixture token with write scope. First prove that a non-owner remains
denied, then delete as an owner. Verify that a subsequent GET returns not-found.

If the existing demo tokens do not permit this scenario, add a narrowly named
fixture rather than weakening an unrelated token.

## 7. Complete the Quality Loop

Run:

```bash
gofmt -w .
go test ./...
go vet ./...
go tool staticcheck ./...
go test -race ./...
```

Review the diff and confirm:

- domain logic is not in the HTTP handler
- environment or concrete adapters did not leak into domain packages
- interfaces remain consumer-owned and narrow
- every error path has an intentional HTTP mapping
- authorization has both allow and deny tests
- the operation has an explicit cross-store consistency policy

## Stretch Tasks

Complete these independently:

1. Add a `DELETE` example to the request-flow documentation.
2. Add an OpenFGA contract case proving only owners can delete.
3. Add an audit decorator around document deletion.
4. Add a benchmark only if you have a performance question to answer.
5. Replace a hand-written repeated test with a table only if readability
   improves.

## Completion Checkpoint

You have completed the Go course when you can implement this feature and explain:

- which declarations use values and which use pointers
- where each interface belongs and why
- how errors retain their type across package boundaries
- how context reaches authorization and storage
- why handlers are concurrency-sensitive
- which tests are unit, contract, and HTTP boundary tests
- where the application chooses concrete implementations

At that point, use [Production readiness](40-production-readiness.md) as a review
checklist rather than as introductory material.
