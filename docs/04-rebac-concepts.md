# ReBAC Concepts

Relationship-based access control answers:

```text
does subject S have permission P on resource R?
```

In this repo:

```text
does user:alice have can_edit on document:roadmapDocument?
```

This chapter gives names to the pieces you already saw in the graph chapter.
It is deliberately compact: learn the vocabulary, then use it immediately.
For a single reusable picture tying these terms together, keep
[the ReBAC mental model](rebac-mental-model.md) beside this chapter.

## The Core Thought Process

ReBAC modeling starts with a product fact, not with code:

```text
Alice is a member of the platform team.
Platform team members are editors of the product workspace.
The roadmap document lives in the product workspace.
Editors can edit documents in that workspace.
```

The first three sentences are facts that can change at runtime. They become
relationships. The last sentence is a rule about how facts imply permission. It belongs
in the model.

That split is the main design move:

```text
relationship -> a durable product fact
model        -> a reusable rule for deriving access from facts
check        -> one authorization question at request time
```

When you model another domain, ask this first:

- Is this a business relationship that can be created or removed? Store a relationship.
- Is this an action the application wants to allow or deny? Define a permission.
- Is this a rule that should apply to many resources? Put it in the model.
- Is this true only for one request, such as time or device state? Treat it as
  context, not as a long-lived relationship.

## Resources

Resources are typed IDs:

```text
user:alice
team:platformTeam
workspace:productWorkspace
document:roadmapDocument
```

Go models them in `internal/rebac/rebac.go`:

```go
type Resource string
```

## Relations

Relations name durable or structural connections:

```text
member
editor
viewer
workspace
```

## Permissions

Permissions name policy-derived authority:

```text
can_read
can_edit
can_delete
```

They are checked but never stored as relationship facts.

## Relationships

A relationship is one durable fact:

```text
subject + relation + resource
```

Domain examples:

```text
user:alice                  member     team:platformTeam
team:platformTeam#member    editor     workspace:productWorkspace
workspace:productWorkspace  workspace  document:roadmapDocument
```

Read them as:

```text
Alice is a member of Platform Team.
Platform Team members are editors of Product Workspace.
Product Workspace is the workspace of Roadmap Document.
```

The Go type uses the same field names:

```go
type Relationship struct {
    Subject  Subject
    Relation Relation
    Resource Resource
}
```

Therefore the second relationship becomes:

```go
rebac.NewRelationship(
    rebac.SubjectSet(
        rebac.Team("platformTeam"),
        rebac.RelationTeamMember,
    ),
    rebac.RelationWorkspaceEditor,
    rebac.Workspace("productWorkspace"),
)
```

OpenFGA translates the same values at its adapter boundary:

```text
domain:  subject, relation, resource
OpenFGA: user,    relation, object
```

A relationship is a stored fact, not the complete effective policy. The model
derives permissions from several relationships. Alice has the `can_edit`
permission on the roadmap document even though no `can_edit` relationship is
stored.

## Why Relationships

Relationships work well for ReBAC because they are small, independent facts. One fact
can be added, removed, replicated, audited, or replayed without rewriting the
authorization model.

| Product change | Relationship change | Model change |
|---|---|---|
| Alice joins a team | write `user:alice member team:platformTeam` | none |
| Bob loses workspace access | delete `user:bob viewer workspace:productWorkspace` | none |
| a document moves workspace | replace its `workspace` relationship | none |
| editors gain a new permission | none | update the model rule |

This is why the repo does not store `can_edit` or `can_read` relationships. Those are
derived permissions. Storing derived permissions would duplicate the model's
work and make revocation harder: removing Alice from the team would also require
finding and deleting every materialized permission she inherited from that team.

Good relationship candidates usually answer one of these questions:

- Who belongs to this group?
- Which group has a relation to this resource?
- Who directly owns or shares this resource?
- Which parent resource does this resource inherit from?

Poor relationship candidates are usually computed outcomes:

- `user:alice can_edit document:roadmapDocument`
- `user:bob can_read document:roadmapDocument`

Those are answers to checks, not source-of-truth facts.

## Subject Sets

`team:platformTeam#member` means:

```text
everyone who has member on team:platformTeam
```

One relationship can grant access to a whole team:

```text
team:platformTeam#member  editor  workspace:productWorkspace
```

## Checks

A check asks whether a subject belongs to the effective set for a permission:

```go
rebac.CheckRequest{
    Subject:    rebac.User("alice"),
    Permission: rebac.PermissionDocumentEdit,
    Resource:   rebac.Document("roadmapDocument"),
}
```

The evaluator tries to prove that request by following only the relationships and
model rules admitted by `can_edit`. An arbitrary graph connection is not
enough.

This model is grant-oriented and defaults to deny:

```text
proof of an allowed path exists  -> allow
no allowed path can be proved    -> deny
```

There are no stored deny relationships in this primer and no “deny overrides allow”
rule. OpenFGA can model exclusions, but explicit-deny semantics add policy and
debugging complexity and should be introduced only for a real requirement.
Removing the last relationship that supports a path revokes the derived access, subject
to the consistency behavior of the authorization backend.

In OpenFGA API terminology, the subject field is named `user`, but it can
represent a human, workload, another resource, userset, or typed wildcard when the
model permits it.

## The Demo Story

The fixtures say:

```text
Alice is a member of platformTeam.
platformTeam members are editors of productWorkspace.
roadmapDocument lives in productWorkspace.
```

Therefore Alice can edit the roadmap document.

Bob is a viewer of the workspace, so Bob can read but not edit.

Casey has no path through the graph, so Casey is denied.

## Try It

```bash
go test -v -run TestTrace ./internal/authz
```

Then edit `internal/fixtures/fixtures.go`, change one relationship, and predict which
checks change before rerunning the test.

## Checkpoint

Explain the difference between these two values:

```text
user:alice
team:platformTeam#member
```

The first is one subject. The second is a set of subjects defined by a relation
on another resource.

Next: [OpenFGA model](05-openfga-model.md) shows how the schema decides which
relationship paths count for a permission.
