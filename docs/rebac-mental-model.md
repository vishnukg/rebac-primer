# A ReBAC Mental Model You Can Reuse

This chapter is the compact model to keep in your head while reading the rest
of the repository or designing authorization at work. Return to it whenever the
code or OpenFGA DSL starts to feel abstract.

For a first pass, read through “Walk The Alice Decision Backward” and stop. The
remaining sections are a reference for modeling and debugging later.

## The One Equation

```text
model + relationship tuples + check request = decision
rules   current facts          question        allow, deny, or error
```

Those parts change on different clocks:

- The **model** changes when product policy changes.
- **Tuples** change when users join teams, resources move, or sharing changes.
- A **check** happens for each protected operation.

Do not mix them. In particular, do not store the result of a check as if it
were a durable relationship fact.

## Think In Sets First, Graphs Second

The graph picture is useful, but the most precise mental model is set
membership:

```text
document:roadmapDocument#can_edit
```

means:

```text
the effective set of subjects allowed to edit roadmapDocument
```

A check asks whether one subject is in that set:

```text
Is user:alice a member of document:roadmapDocument#can_edit?
```

The model defines how to construct the set. Tuples provide its current members
and links to other sets. Graph traversal is the engine's way of answering the
set-membership question.

This prevents a common mistake: authorization is not arbitrary graph
reachability. A path counts only when every step is permitted by the
authorization model.

## The Five Things In Every Decision

### 1. Principal

The principal is the already-authenticated identity making the request:

```text
user:alice
```

Authentication must establish this value before ReBAC runs. Use a stable,
internal identifier derived from a validated identity—not an unverified JWT
claim, email address, display name, or caller-controlled header.

### 2. Object

An object is a typed resource or grouping concept:

```text
user:alice
team:platformTeam
workspace:productWorkspace
document:roadmapDocument
```

The type is part of the identifier. It tells the model which relations and
rules are valid.

### 3. Relation

A relation names a set on an object:

```text
team:platformTeam#member
workspace:productWorkspace#editor
document:roadmapDocument#can_edit
```

Some relations represent durable roles or structure, such as `member`,
`owner`, or `workspace`. Others represent application actions, such as
`can_read` or `can_edit`.

### 4. Tuple

A tuple is one stored fact. Read it as:

```text
<subject> is a <relation> of <object>
```

For example:

```text
user:alice member team:platformTeam
```

means “Alice is a member of Platform Team.” OpenFGA writes tuples in
subject–relation–object order. The Go `TupleKey` uses named fields in
object–relation–user order; the meaning is identical.

### 5. Permission check

Application code should ask for the operation it wants to perform:

```text
Check(user:alice, can_edit, document:roadmapDocument)
```

It should not ask “is Alice a workspace editor?” and then reproduce document
inheritance rules in application code. `can_edit` is the stable contract;
the model owns how it is derived.

## Three Ways A Set Gets Members

Most paths in this repository use one of three shapes.

### Direct membership

```text
user:alice member team:platformTeam
```

This directly places Alice in `team:platformTeam#member`.

### Membership through another set

```text
team:platformTeam#member editor workspace:productWorkspace
```

The subject is a userset, not one user. It places every member of Platform Team
in `workspace:productWorkspace#editor`.

The `#` is best read as “the set defined by this relation on this object.”

### Membership derived by the model

```text
define viewer: [user, team#member] or editor
define can_edit: editor
```

These rules say that workspace editors are also viewers and that the effective
document `can_edit` set equals its effective `editor` set. No `can_edit` tuple
is written.

Parent inheritance is another model derivation:

```text
define editor: [user] or editor from workspace or owner
```

It follows the document's `workspace` relation and includes the editor set of
that workspace.

## Walk The Alice Decision Backward

Start with the question, not with all stored data:

```text
Is Alice in document:roadmapDocument#can_edit?
```

The model rewrites the question:

```text
document#can_edit
  = document#editor
  = direct document editors
    OR document owners
    OR editor from document#workspace
```

The stored document-to-workspace tuple selects `productWorkspace`, so continue:

```text
Is Alice in workspace:productWorkspace#editor?
```

That set contains `team:platformTeam#member`, so continue:

```text
Is Alice in team:platformTeam#member?
```

The direct membership tuple proves yes. The proof returns through the chain:

```text
Alice is a team member
  -> therefore a workspace editor
  -> therefore a document editor
  -> therefore allowed to edit the document
```

The evaluator works backward from requested permission to supporting facts. It
does not begin at Alice and wander across every edge in the store.

## Facts, Roles, And Permissions

Use these distinctions when modeling:

| Kind | Examples | Usually written as a tuple? |
|---|---|---:|
| identity | `user:alice` | no; supplied by authentication |
| structural fact | document belongs to workspace | yes |
| membership | Alice belongs to team | yes |
| scoped role | team edits workspace | yes |
| computed permission | Alice can edit document | no |
| request context | current time, device risk | no long-lived tuple |

Roles are relationships scoped to an object. Permissions are the operations
application code enforces. Keeping `editor` separate from `can_edit` allows the
meaning of `can_edit` to evolve without changing every call site.

## Allow, Deny, And Error

The primer uses grant-oriented, default-deny policy:

```text
valid proof exists       -> allow
no valid proof exists    -> deny
engine cannot decide     -> error / indeterminate
```

An outage is not a policy denial, even if the application fails closed and
returns a non-successful response for both. Keep deny and error separate in
logs, metrics, retries, and incident handling.

This model has no stored deny tuples and no deny-overrides-allow rule.
Exclusions can be modeled when a real requirement needs them, but they make
reasoning, migration, and debugging more complex.

## The Three Authorization Gates In The HTTP Path

The running application has three separate gates:

```text
1. access-token validation  -> may this credential be trusted for this API?
2. OAuth scope              -> may this client call this class of endpoint?
3. ReBAC check              -> may this principal act on this exact object?
```

Passing one gate does not imply passing another. A test rejected by the scope
gate does not prove the ReBAC model would deny the same actor.

## How To Debug A Decision

When a check surprises you, use this order:

1. Write the exact subject, permission, and object.
2. Confirm the subject came from validated authentication.
3. Open the model and expand only the requested permission.
4. List the tuples needed by each permitted branch.
5. Confirm those facts exist in the authorization store.
6. Check model ID, tenant boundary, consistency mode, and recent writes.
7. Distinguish a real deny from an evaluation error.

Do not begin by dumping every tuple. Goal-directed reasoning is easier and is
closer to how the evaluator works.

## How To Model A New Product Rule

Start with plain product sentences:

```text
Project maintainers may archive a project.
Organization administrators are project maintainers for projects in that org.
Blocked users must not archive projects.
```

Then classify each noun and statement:

1. Which nouns are typed objects?
2. Which changing facts become tuples?
3. Which reusable implications belong in the model?
4. Which `can_*` permission should application code check?
5. Which allow, near-miss deny, revocation, and cross-tenant cases prove it?
6. Which service owns each tuple and how is it synchronized?

If you cannot name the source of truth for a tuple, the production design is
not finished.

## Mapping The Model To This Repository

| Mental-model part | Go teaching backend | OpenFGA backend |
|---|---|---|
| vocabulary | `internal/rebac` | model types and relation names |
| rules | `internal/authz/model.go` | `deployments/openfga/model.fga` |
| facts | `authz.InMemoryStore` | OpenFGA relationship store |
| decision engine | `authz.GraphEvaluator` | OpenFGA Check API |
| application boundary | `documents.AuthorizationService` | same interface |
| policy specification | shared contract cases | `.fga.yaml` model tests and contract checks |

The in-process evaluator exists to make the proof visible. In production, keep
the vocabulary, model, tuples, checks, enforcement points, and contract tests;
replace the educational evaluator and in-memory store with an operated engine
and durable relationship data path.

## Memory Card

If you retain only seven lines, retain these:

```text
Authenticate first; authorize second.
An object#relation is a set of subjects.
A tuple is one current product fact.
The model says how effective sets are derived.
A check asks whether one principal belongs to one permission set.
Only model-valid paths count; no proof means deny.
Deny and engine failure are different outcomes.
```

Next, read [ReBAC concepts](04-rebac-concepts.md) for the vocabulary,
[the OpenFGA model](05-openfga-model.md) for the set rules, and
[the graph evaluator walkthrough](27-graph-evaluator-walkthrough.md) to watch
the proof execute.

Primary references for the same concepts are the OpenFGA
[concepts](https://openfga.dev/docs/concepts),
[usersets](https://openfga.dev/docs/modeling/building-blocks/usersets), and
[model-design principles](https://openfga.dev/docs/best-practices/modeling-design-principles).
