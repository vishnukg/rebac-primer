# ReBAC concepts

Authorization asks:

```text
Can subject S perform action A on object O?
```

ReBAC answers by turning authorization data into a graph.

## Objects

Objects are typed ids:

```text
user:alice
team:platform
workspace:acme
document:roadmap
```

## Relations

Relations are named edges:

```text
team:platform member user:alice
workspace:acme editor team:platform#member
document:roadmap workspace workspace:acme
```

## Tuples

A tuple is one stored fact:

```text
(object, relation, user)
```

In code:

```ts
tuple(workspace("acme"), "editor", subjectSet(team("platform"), "member"))
```

This single tuple means every current and future member of
`team:platform` is an editor of `workspace:acme`.

## Graph traversal

When checking `user:alice can_edit document:roadmap`, the graph path is:

```text
user:alice
  -> member of team:platform
  -> team:platform#member is editor of workspace:acme
  -> document:roadmap belongs to workspace:acme
  -> workspace editor implies document editor
  -> document editor implies can_edit
```

Read `src/authz/graph-authorizer.ts` and run
`test/graph-authorizer.test.ts` to see the traversal as a trace.
