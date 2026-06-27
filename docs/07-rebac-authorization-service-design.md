# Designing a ReBAC Authorization Service

This chapter moves from the mechanics of tuples and graph traversal to the
engineering decisions required for a real authorization service.

Read it after the OpenFGA model chapter. The goal is not to teach one vendor or
one paper. It is to build a reusable mental model for implementing ReBAC at
work.

```text
product requirements
        ↓
authorization model
        ↓
relationship data
        ↓
decision API
        ↓
enforcement in every protected operation
```

OpenFGA and Google's Zanzibar paper appear as concrete references because they
make many of these ideas precise. They are examples of ReBAC systems, not the
definition of ReBAC.

## How to Read This Chapter

For a first pass, focus on:

1. What ReBAC Is
2. Start With Product Sentences
3. Core Data Concepts
4. Authorization Service Boundaries
5. Data Ownership and Synchronization
6. OpenFGA or a Custom ReBAC Engine?
7. Implementation Checklist

Return later for consistency, caching, multi-tenancy, policy migration,
auditability, and performance. Those sections become critical during a real
prototype and production design review.

## What ReBAC Is

Relationship-based access control grants or denies an operation based on
relationships among subjects and resources.

```text
Alice is a member of Platform Team.
Platform Team is an editor of Product Workspace.
Roadmap Document belongs to Product Workspace.
```

Those relationships imply:

```text
Alice can edit Roadmap Document.
```

The core decision shape remains:

```text
Can subject S perform permission P on resource R?
```

For this repository:

```text
Check(user:alice, can_edit, document:roadmapDocument)
```

ReBAC is broader than any particular tuple syntax or graph engine. Different
systems may express policy as:

- relationship tuples plus userset/set-algebra rules
- graph path expressions
- object-oriented relationship paths
- a hybrid of relationships, roles, and attributes

The common idea is that the relationship structure is part of the authorization
decision.

## ReBAC Does Not Replace Every Other Model

A practical system often combines models:

```text
authentication  → establishes the principal
OAuth scope     → permits a client to call an API category
ReBAC           → checks object-specific relationships
ABAC/context    → adds time, network, risk, or resource attributes
```

RBAC can also exist inside ReBAC:

```text
user:alice member team:platform
team:platform#member editor workspace:product
```

`editor` is role-like, but it is scoped to one workspace and reached through a
relationship. The problem ReBAC solves is not “roles are always bad.” It solves
the mismatch between global roles and object-specific collaboration, ownership,
hierarchies, sharing, and delegation.

## Start With Product Sentences

Do not begin with graph syntax. Begin with authorization requirements that
product and security stakeholders can review:

```text
Workspace owners can manage a workspace.
Workspace editors can create documents in that workspace.
Team members inherit the access granted to their team.
Document owners can delete the document.
Document viewers can read but cannot edit.
```

Turn each requirement into examples:

| Subject | Operation | Resource | Expected | Reason |
|---|---|---|---:|---|
| Alice | edit | roadmap | allow | team membership grants workspace editor |
| Bob | read | roadmap | allow | direct workspace viewer |
| Bob | edit | roadmap | deny | viewer is not editor |
| Casey | read | roadmap | deny | no granting relationship |

This permission matrix is the policy contract. Model syntax and code are
implementations of it.

## From Requirements to Policy

After writing product sentences, classify each sentence before touching the DSL:

| Product sentence | Classification | Where it goes |
|---|---|---|
| Alice is in platformTeam | durable relationship fact | tuple |
| platformTeam members edit productWorkspace | durable relationship fact using a subject set | tuple |
| roadmapDocument lives in productWorkspace | durable structural fact | tuple |
| workspace editors can edit workspace documents | reusable derivation rule | model |
| document owners can delete documents | application permission rule | model |
| user has `documents:write` OAuth scope | request/token attribute | checked outside ReBAC |

This classification prevents two common mistakes:

- storing derived permissions as tuples, which creates duplicated authorization
  state and hard revocation
- putting changing product facts into the model, which makes ordinary workflow
  changes require policy deployment

The model should be boring and reusable. The tuples should carry the changing
product state.

Ask these questions before modeling:

- What are the protected resource types?
- Which operations require authorization?
- Which relationships exist in the product already?
- Which component owns each relationship?
- Which relationships are durable, and which exist only for one request?
- How quickly must grants and revocations take effect?
- Do tenants share resources or identities?
- Which decisions require an explanation or audit trail?

Model the actual domain rather than inventing a generic meta-model too early:

```text
Prefer: organization, workspace, repository, document
Avoid:  entity, resource, generic_role for everything
```

Specific types make the policy easier to review, support type-specific
permissions and listing, and usually produce shallower evaluation paths. Use
recursive generic hierarchies only when the product genuinely permits arbitrary
nesting.

## Why Policy Has Shape

A good ReBAC policy usually has three layers:

```text
base relations       owner, editor, viewer, member
structural relations workspace, parent, organization
computed permissions can_read, can_edit, can_delete
```

Base relations are usually product concepts people understand. Structural
relations connect objects so access can inherit. Computed permissions are the
operations application code checks.

This repo's policy follows that shape:

| Layer | Example | Why it exists |
|---|---|---|
| base relation | `workspace#editor` | a workspace-scoped role-like fact |
| subject set | `team#member` | grant access to a dynamic group |
| structural relation | `document#workspace` | connect child document to parent workspace |
| hierarchy rule | `viewer includes editor` | avoid duplicating weaker-role tuples |
| inheritance rule | `editor from workspace` | reuse workspace access for documents |
| computed permission | `can_edit: editor` | let application code check an action |

When extending the model, place a new concept in the right layer. For example,
`folder` would probably be a new object type plus a structural relation;
`can_share` would probably be a computed permission; "Alice is a reviewer"
would probably be a tuple.

## The Graph Has Semantics

It is useful to draw ReBAC as a graph:

```text
user:alice
  └─member of─► team:platform

team:platform#member
  └─editor of─► workspace:product

workspace:product
  └─workspace of─► document:roadmap
```

But authorization is not arbitrary reachability. A random graph path must not
grant access.

The requested permission defines which relationships may be traversed:

```text
can_edit = editor
editor   = direct editor OR workspace editor OR owner
```

Then the decision engine evaluates whether Alice belongs to the effective set:

```text
document:roadmap#can_edit
```

This distinction is fundamental:

```text
Incorrect: any path between Alice and the document grants access
Correct: only a path admitted by the permission's policy grants access
```

## Core Data Concepts

Names vary between systems, but a robust design usually needs these concepts.

### Subject

The actor whose authority is being checked:

```text
user:alice
service:billing-worker
agent:document-assistant
```

Do not assume every subject is a human user.

### Resource or object

The protected entity:

```text
workspace:product
document:roadmap
invoice:INV-001
```

Use immutable, opaque identifiers. Avoid putting email addresses, names, or
other personal data in relationship keys.

### Relation

A named relationship:

```text
member
parent
owner
editor
viewer
```

Relations describe facts or sets of subjects associated with a resource.

### Permission

An application action expressed as a policy result:

```text
can_read
can_edit
can_delete
can_share
```

Keeping permissions separate from structural relations gives the model room to
evolve:

```text
can_edit = editor
editor includes owner
```

Callers ask for `can_edit`; they do not need to know how that permission is
currently derived.

### Relationship tuple

A stored relationship fact:

```text
subject + relation + object
```

For example:

```text
user:alice  member  team:platform
```

For this course, the canonical external representation is OpenFGA's:

```text
subject + relation + object
```

This repository's Go `TupleKey` lists fields as `Object`, `Relation`, `User`.
That internal struct layout does not reverse or change the relationship. Read
field names and convert explicitly at adapter boundaries.

### Direct and implied relationships

A direct relationship is backed by stored data:

```text
user:alice member team:platform
```

An implied relationship is derived by policy:

```text
Alice can_edit roadmap
```

There is no need to store every implied permission. Materializing all derived
access causes duplication, write amplification, and difficult revocation.

### Subject sets or usersets

A subject set represents everyone related to an object by a relation:

```text
team:platform#member
```

It allows one relationship to grant access to a dynamic group:

```text
team:platform#member editor workspace:product
```

Adding or removing a team member updates inherited access without rewriting
every workspace relationship.

## Common Policy Building Blocks

Most ReBAC models combine a small number of ideas.

### Direct grant

```text
user:alice viewer document:roadmap
```

Use for explicit sharing or ownership.

### Role hierarchy

```text
viewer includes editor
editor includes owner
```

As sets of subjects:

```text
owner ⊆ editor ⊆ viewer
```

### Group inheritance

```text
team:platform#member editor workspace:product
```

### Parent-child inheritance

```text
workspace:product workspace document:roadmap
document editor includes editor from workspace
```

This is useful for folders, organizations, projects, repositories, workspaces,
and documents.

### Union, intersection, and exclusion

```text
can_read  = viewer OR editor
can_merge = writer AND approved_reviewer
can_read  = viewer BUT NOT blocked
```

Union is common. Intersection and exclusion can express important rules, but
they increase reasoning and testing complexity. Use them because the product
rule needs them, not because the engine supports them.

### Context and attributes

Some decisions need information that is not a durable relationship:

```text
current time
selected tenant
network zone
device trust
document classification
```

Possible approaches include:

- evaluate an ABAC rule before or after ReBAC
- pass request-scoped contextual relationships
- use conditional relationships
- model a durable fact when it truly is a durable relationship

Do not encode every runtime attribute as long-lived graph data.

## Authorization Service Boundaries

A production authorization system usually separates:

```text
Policy Enforcement Point (PEP)
  application/API code that blocks or permits the operation

Policy Decision Point (PDP)
  authorization service that evaluates the request

Policy Information
  relationships, model versions, and any trusted context used by the decision
```

In this repository:

```text
HTTP handler         → request boundary
documents.Service    → decides when authorization is required
authz/OpenFGA        → evaluates the authorization question
```

The domain service is the important enforcement point because it knows the
business operation. A UI hiding a button is not enforcement.

## Decision API Design

Start with a small API:

```text
Check(subject, permission, resource) -> allow or deny
```

A production response often needs more than a boolean:

```json
{
  "decision": "allow",
  "policyVersion": "01J...",
  "decisionId": "01J...",
  "evaluatedAt": "2026-06-20T02:00:00Z"
}
```

Useful operations may include:

| Operation | Purpose |
|---|---|
| Check | One subject, permission, and resource |
| BatchCheck | Several independent checks |
| ListResources | Resources a subject may access |
| ListSubjects | Subjects with access to a resource |
| Explain/Expand | Debug policy structure or a decision |
| WriteRelationships | Add or remove relationship facts |
| WatchChanges | Feed relationship updates to consumers |

Do not assume listing is merely “run Check for every database row.” That can
create latency, load, pagination, and information-leak problems. Design
permission-aware search explicitly.

## Data Ownership and Synchronization

The authorization service should not invent business relationships. Product
domains own facts such as:

```text
team membership
document ownership
workspace hierarchy
project assignment
```

Decide the source of truth for every relation:

| Relation | Source of truth | Writer |
|---|---|---|
| team member | team service | membership workflow |
| document workspace | document service | document creation/move |
| document owner | document service | ownership workflow |

Common synchronization patterns:

- synchronous dual write with compensation
- transactional outbox plus idempotent consumers
- change-data capture
- request-scoped contextual data
- authorization service as the source of truth for explicitly delegated grants

There is no universal answer. The critical requirements are:

- one documented owner per relationship
- idempotent writes and deletes
- retry and reconciliation strategy
- observability for drift
- explicit behavior while data is missing or delayed

This repository uses compensation during document creation for teaching.
A production system commonly uses an outbox or durable event workflow.

## Consistency and Revocation

Authorization correctness includes time:

```text
When was this relationship changed?
Which version did the decision evaluate?
How long may stale access remain?
```

Grants and revocations may have different risk:

- a delayed grant causes temporary inconvenience
- a delayed revocation may expose sensitive data

Document consistency requirements by operation:

| Operation | Example requirement |
|---|---|
| ordinary read | small bounded staleness may be acceptable |
| change ownership | subsequent checks must observe the change |
| remove employee | revocation should take effect immediately or within a defined SLO |
| destructive action | use the freshest supported decision mode |

The Zanzibar paper is valuable here because it explains the “new enemy”
problem and uses consistency tokens called zookies to coordinate content and
ACL versions.

Current OpenFGA uses query consistency preferences:

```text
MINIMIZE_LATENCY
HIGHER_CONSISTENCY
```

That is not the same API or guarantee as Zanzibar zookies. If you use OpenFGA,
follow current OpenFGA semantics rather than assuming every Zanzibar paper
feature exists.

## Caching

Authorization caches can improve latency and throughput, but every cache creates
a revocation window.

Possible cache keys include:

```text
subject + permission + resource + policy version + relationship version/context
```

Questions to answer:

- Are allow and deny results cached?
- What invalidates the entry?
- What is the maximum stale-access window?
- Does the cache key include tenant and policy version?
- Can sensitive operations bypass the cache?
- How are partial outages handled?

Never add an authorization cache without a written invalidation and staleness
model.

## Multi-Tenancy

Tenant isolation must be structural, not a naming convention that callers can
forget.

Approaches include:

- tenant as a parent object in the graph
- tenant-qualified resource IDs
- isolated stores or databases for stronger boundaries
- a required tenant relationship on every permission path

Test cross-tenant near misses:

```text
same user, same resource ID, different tenant
team from tenant A related to resource in tenant B
administrator of one tenant accessing another tenant
```

Do not rely only on globally unique IDs to enforce tenant isolation.

## Policy Lifecycle

Treat the authorization model like application code:

```text
version
review
test
deploy
observe
rollback or migrate
```

Model changes may require data migration. Renaming `editor` to `writer`, for
example, can require both application changes and relationship rewrites.

A safe rollout can include:

1. write a permission matrix
2. add tests for old and new behavior
3. deploy the new policy version without activating it
4. run shadow decisions against both versions
5. investigate every unexpected difference
6. migrate relationship data if needed
7. progressively switch callers

OpenFGA models are immutable and identified by model IDs, which supports this
style of rollout. Other ReBAC systems need an equivalent policy-version
strategy.

## Testing Strategy

Authorization tests should be treated as security specifications.

Cover:

- direct allow
- inherited allow
- group/subject-set allow
- unrelated-user denial
- near-miss denial
- cross-tenant denial
- revoked relationship
- missing relationship data
- invalid relationship shape
- policy migration parity
- backend timeout or failure
- stale versus fresh consistency behavior
- listing APIs, not only Check

Test the reason for a decision. A request denied by a scope gate does not prove
that the ReBAC model would deny it.

This repository has:

- Go evaluator tests
- shared backend contract tests
- HTTP enforcement tests
- an OpenFGA `.fga.yaml` model contract in
  `deployments/openfga/model.fga.yaml`

Run the OpenFGA model contract with:

```bash
make openfga/model-test
```

## Failure Semantics

Fail closed:

```text
authorization timeout ≠ allow
unknown permission     ≠ allow
missing policy version ≠ use an arbitrary version
```

Distinguish:

- deny: the policy evaluated successfully and did not grant access
- indeterminate/error: the service could not make a trustworthy decision

Your application may map both to a non-successful response, but metrics, retry
policy, and incident response need the distinction.

Define:

- timeout budget
- retry policy
- circuit-breaker behavior
- whether reads and writes differ during outages
- how callers handle malformed requests
- whether not-found and forbidden responses are intentionally indistinguishable

## Auditability and Explanations

Record enough information to investigate decisions without logging sensitive
graph data indiscriminately:

```text
decision ID
subject
permission
resource
allow/deny/error
policy version
consistency mode
latency
caller/service identity
```

An explanation trace is useful for tests and debugging, but exposing full graph
paths to end users can leak organization structure. Separate internal
explanations from public error messages.

## Performance and Model Complexity

Cost depends on:

- relationship depth
- branch breadth
- nested groups
- intersections and exclusions
- hot resources
- listing result size
- datastore and cache behavior

Set limits for depth, breadth, result size, and request concurrency. Measure
real permission shapes rather than benchmarking only direct tuples.

Model clarity is a performance and security feature. Prefer a short,
explainable path:

```text
user → team membership → workspace permission → document inheritance
```

over a deeply indirect policy that no reviewer can reason about.

## OpenFGA or a Custom ReBAC Engine?

This is the central implementation decision for this course.

The options are not:

```text
OpenFGA  versus  understanding ReBAC
```

You need the ReBAC model and service architecture either way. The actual choice
is:

```text
adopt OpenFGA's tuple/userset engine
versus
own the policy engine, storage, APIs, and operations yourself
```

### What OpenFGA gives you

OpenFGA provides:

- an authorization-model language and validation
- relationship-tuple storage APIs
- Check, BatchCheck, ListObjects, ListUsers, Read, Expand, and change APIs
- usersets, parent inheritance, union, intersection, exclusion, and conditions
- immutable authorization-model versions
- configurable consistency preferences and caching
- supported SDKs, server configuration, metrics, tracing, and datastore support
- model testing through `.fga.yaml` files and the CLI

This removes a large amount of engine and operational work.

### What OpenFGA does not decide for you

Your team still owns:

- product authorization requirements
- resource, relation, and permission vocabulary
- where enforcement occurs
- authentication of callers and workloads
- source-of-truth ownership for relationships
- eventing, retries, idempotency, and reconciliation
- consistency requirements per operation
- tenant isolation design
- policy migration and rollout procedures
- application-facing error semantics and audit policy

OpenFGA evaluates the model you give it. It cannot tell whether that model
matches the business's intended security policy.

### What a custom engine requires

Building the teaching evaluator is useful for learning. Building a production
authorization system also requires:

- durable and indexed relationship storage
- policy validation and versioning
- consistency semantics
- cache invalidation
- bounded graph evaluation
- multi-region behavior if required
- authentication and authorization of the authz service itself
- migrations, audit, metrics, tracing, and operational tooling

It also requires ongoing ownership of semantic correctness. Features that begin
as “just traverse a graph” become substantially harder when you add nested
groups, intersection, exclusion, listing, cycles, consistency, hot objects,
multi-tenancy, migrations, and bounded evaluation.

### Comparison

| Concern | OpenFGA | Custom implementation |
|---|---|---|
| Policy language | Provided DSL/JSON model | Design and maintain your own |
| Tuple validation | Enforced against model | Implement and test it |
| Check evaluation | Provided | Implement graph/set evaluation |
| ListObjects/ListUsers | Provided with operational limits | Design algorithms and indexes |
| Conditions/context | Supported | Design semantics and evaluator |
| Policy versions | Immutable model IDs | Build versioning and rollout |
| Storage | Supported datastores | Own schema, indexes, migration, scale |
| Consistency | Documented OpenFGA modes | Define and implement guarantees |
| Observability | Server metrics/tracing available | Instrument every layer |
| Extensibility | Constrained to OpenFGA semantics | Full control |
| Dependency risk | External service and project dependency | Internal platform ownership |
| Initial effort | Lower engine effort | High |
| Long-term effort | Operate/integrate OpenFGA | Operate and evolve the entire engine |

### When OpenFGA is a strong fit

- Your policy maps naturally to typed relationships and set operations.
- You need groups, hierarchy, ownership, sharing, or delegation.
- You want to avoid owning graph evaluation and relationship indexing.
- OpenFGA's APIs and consistency model meet your requirements.
- Running or consuming an external authorization service is acceptable.
- Your team can invest in model design, integration, and operations.

### When a custom or different engine may be justified

- The policy language is fundamentally different from tuple/userset ReBAC.
- Most decisions are complex attribute or risk expressions rather than
  relationships.
- You require guarantees or query shapes OpenFGA cannot provide.
- A very narrow, static policy can be implemented safely without becoming a
  general authorization platform.
- Regulatory, deployment, or dependency constraints rule OpenFGA out.
- You already have a mature internal authorization platform with the needed
  semantics and operational support.

“We can write DFS” is not sufficient justification for building. Compare the
full lifecycle, not only the Check algorithm.

### Recommended evaluation approach

Use this repository as a controlled comparison:

1. Treat the Go evaluator as an executable specification of the basic policy.
2. Express the same policy in `deployments/openfga/model.fga`.
3. Run the same permission matrix against both.
4. Add one realistic work requirement at a time: nested groups, tenant
   boundaries, listing, revocation, model migration, and outage behavior.
5. Record where the custom implementation needs new engine features and where
   OpenFGA requires integration or operational work.
6. Decide from measured fit, risk, and ownership cost.

For most teams whose requirements fit the model, OpenFGA is the safer starting
point than creating a new general-purpose ReBAC engine. Keep your application
behind a narrow authorization interface so the decision remains reversible.

## Where Zanzibar and OpenFGA Fit

Zanzibar is a published design for a globally distributed relationship-based
authorization system. It is especially useful for understanding:

- relation tuples and usersets
- object-independent policy rewrites
- nested group evaluation
- consistency under replication and caching
- large-scale operational design

OpenFGA is a Zanzibar-inspired open-source implementation with its own current
DSL, APIs, features, and guarantees. It is useful for learning and implementing
tuple/userset-style ReBAC without recreating the storage and evaluation engine.

This repository's in-process evaluator implements a deliberately small subset:

| Capability | Teaching evaluator | OpenFGA |
|---|---:|---:|
| Direct relationships | yes | yes |
| Subject sets/groups | yes | yes |
| Computed permissions | yes | yes |
| Parent inheritance | document → workspace only | model-defined |
| Union | yes | yes |
| Intersection/exclusion | no | yes |
| Conditions/contextual tuples | no | yes |
| Policy version selection | static Go tables | immutable model IDs |
| Consistency selection | current in-memory state | query preferences |
| Zanzibar zookies | no | no |

## Implementation Checklist

Before implementing ReBAC at work, produce:

1. a glossary of subjects, resources, relations, and permissions
2. product authorization sentences
3. an allow/deny permission matrix
4. a diagram of common relationship paths
5. source-of-truth ownership for every relation
6. the Check and listing API contracts
7. consistency and revocation SLOs
8. multi-tenant isolation rules
9. failure and outage behavior
10. policy versioning and migration strategy
11. audit and observability requirements
12. contract tests independent of the chosen engine

If one of these is missing, the difficult part has probably been deferred rather
than solved.

## Checkpoint

You are ready to design a ReBAC service when you can explain:

- why ReBAC is broader than Zanzibar-style tuple systems
- why valid policy paths differ from arbitrary graph paths
- who owns each relationship and how it reaches the authorization store
- how quickly revocation must become visible
- how Check differs from permission-aware listing
- how policy versions are tested and rolled out
- what the application does when the authorization service cannot decide

## Sources and further study

General authorization and ReBAC:

- [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)
- [Relationship-Based Access Control for OpenMRS](https://arxiv.org/abs/1503.06154)
- [ReBAC policy mining from ACL and object data](https://arxiv.org/abs/1708.04749)

Zanzibar-style systems:

- [Zanzibar: Google's Consistent, Global Authorization System](https://www.usenix.org/conference/atc19/presentation/pang)
- [OpenFGA concepts](https://openfga.dev/docs/concepts)
- [OpenFGA configuration language](https://openfga.dev/docs/configuration-language)
- [OpenFGA authorization-model design principles](https://openfga.dev/docs/best-practices/modeling-design-principles)
- [OpenFGA source-of-truth guidance](https://openfga.dev/docs/best-practices/source-of-truth)
- [OpenFGA relationship queries](https://openfga.dev/docs/interacting/relationship-queries)
- [OpenFGA contextual tuples](https://openfga.dev/docs/interacting/contextual-tuples)
- [OpenFGA query consistency](https://openfga.dev/docs/interacting/consistency)
- [OpenFGA model testing](https://openfga.dev/docs/modeling/testing)

The OpenFGA documentation was reviewed on June 20, 2026; several cited pages
were updated on June 19, 2026.
