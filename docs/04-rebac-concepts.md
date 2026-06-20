# ReBAC Concepts

Relationship-based access control answers:

```text
does user U have relation R on object O?
```

In this repo:

```text
does user:alice have can_edit on document:roadmapDocument?
```

This chapter gives names to the pieces you already saw in the graph chapter.
It is deliberately compact: learn the vocabulary, then use it immediately.

## Objects

Objects are typed IDs:

```text
user:alice
team:platformTeam
workspace:productWorkspace
document:roadmapDocument
```

Go models them in `internal/rebac/rebac.go`:

```go
type Object string
```

## Relations

Relations name edges or permissions:

```text
member
editor
viewer
workspace
can_read
can_edit
```

## Tuples

A tuple is one relationship fact:

```text
(object, relation, user)
```

Examples:

```text
(team:platformTeam, member, user:alice)
(workspace:productWorkspace, editor, team:platformTeam#member)
(document:roadmapDocument, workspace, workspace:productWorkspace)
```

A tuple is a stored fact, not the complete effective policy. The model can
derive implied relationships from several tuples. Alice has an implied
`can_edit` relationship to the roadmap document even though no `can_edit` tuple
is stored.

## Subject Sets

`team:platformTeam#member` means:

```text
everyone who has member on team:platformTeam
```

One tuple can grant access to a whole team:

```text
(workspace:productWorkspace, editor, team:platformTeam#member)
```

## Checks

A check asks whether a subject belongs to the effective set for a permission:

```go
rebac.CheckRequest{
    User:     rebac.User("alice"),
    Relation: rebac.RelationDocumentCanEdit,
    Object:   rebac.Document("roadmapDocument"),
}
```

The evaluator tries to prove that request by following only the tuples and
model rules admitted by `can_edit`. An arbitrary graph connection is not
enough.

In OpenFGA API terminology, the subject field is named `user`, but it can
represent a human, workload, another object, userset, or typed wildcard when the
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

Then edit `internal/fixtures/fixtures.go`, change one tuple, and predict which
checks change before rerunning the test.

## Checkpoint

Explain the difference between these two values:

```text
user:alice
team:platformTeam#member
```

The first is one subject. The second is a set of subjects defined by a relation
on another object.

Next: [OpenFGA model](05-openfga-model.md) shows how the schema decides which
tuple paths count for a permission.
