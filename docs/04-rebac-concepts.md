# ReBAC Concepts

Relationship-based access control answers:

```text
does user U have relation R on object O?
```

In this repo:

```text
does user:alice have can_edit on document:roadmapDocument?
```

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

A check asks whether a path exists through the graph:

```go
rebac.CheckRequest{
    User:     rebac.User("alice"),
    Relation: rebac.RelationDocumentCanEdit,
    Object:   rebac.Document("roadmapDocument"),
}
```

The evaluator tries to prove that request by following tuples and model rules.

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
