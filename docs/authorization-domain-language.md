# Authorization Domain Language

This repository uses one authorization vocabulary across product services,
domain types, HTTP examples, tests, and documentation:

```text
Check(subject, permission, resource) -> decision
```

The vocabulary is deliberately independent of any authorization engine. An
adapter translates it when an external system uses different field names.

## The Terms

| Term | Meaning | Example |
|---|---|---|
| subject | resource or subject set on the left side of a relationship; in a check, the identity being evaluated | `user:alice`, `team:platformTeam#member` |
| resource | typed entity being protected or related | `document:roadmapDocument` |
| action | business operation currently being attempted | `edit` |
| relation | named association on a resource used as stored or derivable policy evidence | `member`, `editor`, `workspace` |
| relationship | stored fact in `(subject, relation, resource)` form | Alice is a member of Platform Team |
| permission | policy-derived authority that an application may check | `can_edit` |
| decision | result of successfully checking one permission | `allow` or `deny` |

Each term has one job. In particular, relations are policy evidence and
permissions are policy results. Application code checks permissions; it does
not reproduce policy by checking a role-like relation.

An evaluation failure is an error, not a denial decision. Callers may fail
closed, but logs and reliability controls should preserve that distinction.

## Relationships And Checks

A stored relationship has this shape:

```text
Relationship(subject, relation, resource)
```

For example:

```text
Relationship(user:alice, member, team:platformTeam)
Relationship(team:platformTeam#member, editor, workspace:productWorkspace)
```

Read those as:

```text
Alice is a member of Platform Team.
Platform Team members are editors of Product Workspace.
```

A permission check has a different middle term:

```text
Check(subject, permission, resource)
Check(user:alice, can_edit, document:roadmapDocument)
```

The model derives `can_edit` from relationships such as `editor`, `owner`, team
membership, and document-to-workspace structure.

## Action Versus Permission

An action is what the application is doing. A permission is the authority that
the action requires:

```text
action:              edit a document
required permission: can_edit
```

Simple operations often map one-to-one to permissions, but they are not the
same concept. A workflow can require several permission checks, and one
permission can protect more than one application entry point.

## Go Types

The shared domain package makes the distinction explicit:

```go
type Relation string
type Permission string

type Relationship struct {
    Subject  Subject
    Relation Relation
    Resource Resource
}

type CheckRequest struct {
    Subject    Resource
    Permission Permission
    Resource   Resource
}
```

Permission constants are separate from relation constants. For example:

```go
rebac.RelationWorkspaceEditor // relationship evidence
rebac.PermissionDocumentEdit  // permission checked by an application
```

The relationship writer accepts only `Relation`, so a permission such as
`can_edit` cannot accidentally be persisted as a durable fact.

## OpenFGA Translation

OpenFGA uses `user`, `relation`, and `object` for both stored tuples and checks.
Its model also declares computed permissions such as `can_edit` in the
`relations` section. That is OpenFGA's transport and model vocabulary, not the
application's domain vocabulary.

The adapter owns this translation:

| Domain | OpenFGA |
|---|---|
| subject | user |
| permission, for a check | relation |
| relation, for a relationship write | relation |
| resource | object |
| relationship | tuple |

Consequently, this OpenFGA request is correct at the adapter boundary:

```go
openfga.ClientCheckRequest{
    User:     string(req.Subject),
    Relation: string(req.Permission),
    Object:   string(req.Resource),
}
```

No product or domain package should adopt `user/relation/object` merely because
the external adapter requires it.

## Review Rules

Use these checks when adding a feature:

1. Name the attempted business action.
2. Name the permission that protects it, usually `can_*`.
3. Add or reuse relations that describe durable business facts.
4. Derive the permission from those relations in policy.
5. Make application code call `Check(subject, permission, resource)`.
6. Store only relationships, never permission results.
7. Translate to engine-specific vocabulary only inside its adapter.
