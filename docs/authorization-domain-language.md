# Authorization Domain Language

This repository uses one authorization vocabulary across product services,
domain types, HTTP examples, tests, and documentation:

```text
Check(subject, action, resource) -> decision
```

The vocabulary is deliberately independent of any authorization engine. An
adapter translates it when an external system uses different field names.

## The Terms

| Term | Meaning | Example |
|---|---|---|
| subject | resource or subject set on the left side of a relationship; in a check, the identity being evaluated | `user:alice`, `team:platformTeam#member` |
| resource | typed entity being protected or related | `document:roadmapDocument` |
| action | policy-derived operation a subject may perform, checked by applications | `can_edit` |
| relation | named association on a resource used as stored or derivable policy evidence | `member`, `editor`, `workspace` |
| relationship | stored fact in `(subject, relation, resource)` form | Alice is a member of Platform Team |
| decision | result of successfully checking one action | `allow` or `deny` |

Each term has one job. In particular, relations are policy evidence and
actions are policy results. Application code checks actions; it does not
reproduce policy by checking a role-like relation.

Much of the authorization literature calls this checked term a *permission*
(SpiceDB even has a `permission` keyword). This repository says *action*
because it names what the subject is trying to do; the concept is the same.

An evaluation failure is an error, not a denial decision. Callers may fail
closed, but logs and reliability controls should preserve that distinction.

Resource ids may not contain `#` or whitespace (`rebac.ValidateID`), because
resources and subject sets share one string space and `#` separates a subject
set from its relation. OpenFGA applies the same rule to object ids.

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

An action check has a different middle term:

```text
Check(subject, action, resource)
Check(user:alice, can_edit, document:roadmapDocument)
```

The model derives `can_edit` from relationships such as `editor`, `owner`, team
membership, and document-to-workspace structure.

## Action Versus Relation

An action is what the application is trying to do on behalf of a subject. A
relation is a durable fact that serves as evidence for allowing it:

```text
action:              can_edit  (checked, never stored)
supporting relation: editor    (stored, or derived from owner)
```

Simple operations often map one-to-one to actions, but a workflow can require
several action checks, and one action can protect more than one application
entry point.

## Go Types

The shared domain package makes the distinction explicit:

```go
type Relation string
type Action string

type Relationship struct {
    Subject  Subject
    Relation Relation
    Resource Resource
}

type CheckRequest struct {
    Subject  Resource
    Action   Action
    Resource Resource
}
```

Action constants are separate from relation constants. For example:

```go
rebac.RelationWorkspaceEditor // relationship evidence
rebac.ActionDocumentEdit      // action checked by an application
```

The relationship writer accepts only `Relation`, so an action such as
`can_edit` cannot accidentally be persisted as a durable fact.

## OpenFGA Translation

OpenFGA uses `user`, `relation`, and `object` for both stored tuples and checks.
Its model also declares computed relations such as `can_edit` in the
`relations` section. That is OpenFGA's transport and model vocabulary, not the
application's domain vocabulary.

The adapter owns this translation:

| Domain | OpenFGA |
|---|---|
| subject | user |
| action, for a check | relation |
| relation, for a relationship write | relation |
| resource | object |
| relationship | tuple |

Consequently, this OpenFGA request is correct at the adapter boundary:

```go
openfga.ClientCheckRequest{
    User:     string(req.Subject),
    Relation: string(req.Action),
    Object:   string(req.Resource),
}
```

No product or domain package should adopt `user/relation/object` merely because
the external adapter requires it.

## Review Rules

Use these checks when adding a feature:

1. Name the attempted business operation.
2. Name the action that protects it, usually `can_*`.
3. Add or reuse relations that describe durable business facts.
4. Derive the action from those relations in policy.
5. Make application code call `Check(subject, action, resource)`.
6. Store only relationships, never check decisions.
7. Translate to engine-specific vocabulary only inside its adapter.
