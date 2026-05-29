# Authorization fundamentals

Authorization is the part of the system that decides whether an authenticated
subject can perform an action on an object.

```text
subject + action + object -> allow or deny
```

Before ReBAC makes sense, you need to understand the common authorization models
and where they break.

## Scene

The product starts simple:

```text
Admins can do everything.
Editors can edit.
Viewers can read.
```

Then customers ask:

```text
Can Bob edit only documents in workspace:productWorkspace?
Can Alice edit documents because she is in team:platformTeam?
Can Casey read one shared document but nothing else?
```

Now global roles are not enough.

## The decision shape

Every authorization check needs the same three pieces:

```text
subject + action + object -> allow or deny
```

For this repo:

```text
subject = user:alice
action  = can_edit
object  = document:roadmapDocument
```

The check becomes:

```text
Can user:alice edit document:roadmapDocument?
```

That sentence sounds strange at first because `can_edit` is written as a
relation. OpenFGA and this repo use relation names for both relationships and
computed permissions:

```text
relationship: user is member of team
permission:   user can_edit document
```

Both are graph questions.

## A Request Timeline

Authorization is one step in a larger request:

```text
HTTP request
  |
  v
authenticate
  "this request is from user:alice"
  |
  v
load/parse target
  "the target is document:roadmapDocument"
  |
  v
authorize
  "can user:alice edit document:roadmapDocument?"
  |
  +-- denied  -> return 403
  |
  +-- allowed -> run business action
```

Two failures are easy to confuse:

```text
401 Unauthorized: the app does not know who you are
403 Forbidden:    the app knows who you are, but you cannot do this
```

The names are historical and slightly confusing. For learning:

```text
401 -> authentication problem
403 -> authorization problem
```

## A Tiny Permission Matrix

Before thinking about graphs, write the desired outcomes:

| Actor | ReBAC subject | Read roadmap? | Edit roadmap? | Why |
|-------|---------------|---------------|---------------|-----|
| Alice | `user:alice` | yes | yes | team membership grants workspace editor |
| Bob | `user:bob` | yes | no | viewer grants read, not edit |
| Casey | `user:casey` | no | no | no relationship path |

This table is a specification. The code and model should make the table true.

If a model cannot explain one row in plain English, slow down before adding more
relations.

## Authentication vs authorization

```text
Authentication
  Who are you?

Authorization
  What can you do?
```

Architecture:

```text
┌──────────────┐
│ Request      │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Authenticate │ "this is user:alice"
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Authorize    │ "can user:alice edit document:roadmapDocument?"
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Execute      │ update the document
└──────────────┘
```

Do not skip authorization just because authentication succeeded.

## Common Beginner Mistakes

Mistake 1: treating login as permission.

```text
Bad:  user is logged in, so allow document update
Good: user is logged in, then check can_edit on this document
```

Mistake 2: checking only broad roles.

```text
Bad:  user has role editor, so can edit every document
Good: user can edit this document if a relationship path grants can_edit
```

Mistake 3: putting authorization only in the client.

```text
Bad:  hide the edit button and trust the browser
Good: hide the edit button for UX, but enforce permission on the server
```

Mistake 4: making the HTTP handler own the policy.

```text
Bad:  every route knows graph rules
Good: document domain asks Authorizer for an allow/deny decision
```

## DAC, MAC, RBAC, ABAC, ReBAC

You will see these acronyms:

```text
DAC   discretionary access control
MAC   mandatory access control
RBAC  role-based access control
ABAC  attribute-based access control
ReBAC relationship-based access control
```

For most application developers, RBAC, ABAC, and ReBAC are the models you will
compare most often.

## RBAC: role-based access control

RBAC grants permissions through roles.

```text
user:alice -> role:editor
role:editor -> permission:edit_document
```

Diagram:

```text
user:alice
    │
    ▼
role:editor
    │
    ▼
edit_document
```

RBAC is good when permissions are broad and stable:

```text
billing_admin can manage billing
support_agent can view support tickets
```

RBAC struggles when permissions are object-specific:

```text
The workspace editor can edit this document but not that document.
```

## Role explosion

To make RBAC object-specific, teams often create more roles:

```text
workspace_acme_editor
workspace_acme_viewer
workspace_beta_editor
workspace_beta_viewer
document_roadmap_editor
document_roadmap_viewer
```

This is role explosion.

Diagram:

```text
users x workspaces x actions = many roles
```

The model becomes hard to understand and hard to maintain.

## ABAC: attribute-based access control

ABAC makes decisions from attributes.

Example:

```text
allow if user.department == document.department
allow if request.ip is trusted
allow if document.classification <= user.clearance
```

Diagram:

```text
┌─────────────┐   attributes    ┌──────────────┐
│ user        │ ───────────────► │ policy engine│
│ document    │ ───────────────► │              │
│ request     │ ───────────────► │              │
└─────────────┘                  └──────┬───────┘
                                        ▼
                                   allow/deny
```

ABAC is powerful for contextual policies, but it can hide product relationships
inside policy expressions.

## ReBAC: relationship-based access control

ReBAC decides from relationships.

```text
team:platformTeam member user:alice
workspace:productWorkspace editor team:platformTeam#member
document:roadmapDocument workspace workspace:productWorkspace
```

Each line is one tuple, written as `object relation user`. The middle line uses
the subject set `team:platformTeam#member` so editor access flows from team
membership rather than being attached to the team object itself. Subject sets
are introduced in detail in `04-rebac-concepts.md`; for now read the line as
"members of the platform team are editors of the product workspace."

Diagram:

```text
user:alice -> team:platformTeam -> workspace:productWorkspace -> document:roadmapDocument
```

ReBAC is strong when your product is naturally relational:

- organizations
- workspaces
- teams
- folders
- documents
- shared resources
- parent-child ownership

That is why ReBAC maps well to collaborative apps.

## Comparing the models

| Model | Best at | Weak spot |
|-------|---------|-----------|
| RBAC | broad job permissions | object-specific sharing |
| ABAC | contextual policy decisions | policies can become opaque |
| ReBAC | object-specific relationships | model design requires care |

Most serious systems use a combination.

Example:

```text
OAuth/OIDC authenticates the user.
RBAC may grant broad admin capability.
ReBAC grants object-specific access.
ABAC may add context checks like tenant or risk.
```

## How ReBAC solves the document problem

Product rule:

```text
Members of team:platformTeam can edit documents in workspace:productWorkspace.
```

RBAC version:

```text
create workspace_acme_editor role
assign every platformTeam member to role
remember to update role when team changes
```

ReBAC version:

```text
team:platformTeam member user:alice
workspace:productWorkspace editor team:platformTeam#member
document:roadmapDocument workspace workspace:productWorkspace
```

Now team membership is the source of truth.

## Authorization architecture in this repo

```text
┌──────────────┐
│ HTTP handler │ parses request
└──────┬───────┘
       │ actor + action + object
       ▼
┌──────────────┐
│ Document     │ enforces business rule
│ Service      │
└──────┬───────┘
       │ Check(user, relation, object)
       ▼
┌──────────────┐
│ Authorizer   │ graph authorizer or OpenFGA adapter
└──────┬───────┘
       │ relationship graph
       ▼
┌──────────────┐
│ Tuple Store  │ facts: object relation user
└──────────────┘
```

The important separation:

```text
HTTP parses.
Domain decides when authz is required.
Authorizer answers allow/deny.
Tuple store holds facts.
```

## Future Agentic Systems

Agentic systems make authorization more important because a model or agent may
take many actions on behalf of a user:

```text
read documents
summarize private content
send messages
create tickets
modify data
call tools
trigger workflows
```

This section is deliberately conceptual. Agentic authorization is still an
emerging architecture area, but the fundamentals here are stable: authenticate
the principal, identify who the action is on behalf of, and authorize each
server-side operation before it executes.

The core questions do not change:

```text
Who is acting?
What action is being attempted?
Which object is the action targeting?
Is the action allowed right now?
```

But agentic systems add one more question:

```text
On whose behalf is the agent acting?
```

A useful mental model:

```text
human user
  authenticates with OAuth/OIDC
  |
  v
app session
  identifies user:alice
  |
  v
agent run
  acts on behalf of user:alice
  |
  v
tool call
  asks Authorizer before touching data
```

Diagram:

```text
┌──────────────┐
│ Human User   │
└──────┬───────┘
       │ login
       ▼
┌──────────────┐
│ Authn layer  │ validates identity
└──────┬───────┘
       │ user:alice
       ▼
┌──────────────┐
│ Agent runtime│ plans tool calls
└──────┬───────┘
       │ wants to read/edit object
       ▼
┌──────────────┐
│ Authz layer  │ Check(user, relation, object)
└──────┬───────┘
       │ allow/deny
       ▼
┌──────────────┐
│ Tool/API     │ executes only if allowed
└──────────────┘
```

Two identities may matter:

| Identity | Example | Why it matters |
|----------|---------|----------------|
| User identity | `user:alice` | Whose data and permissions are being used |
| Agent identity | `agent:docAssistant` | Which agent/tooling is allowed to operate |

For many apps, the agent should not get broad new powers. It should inherit a
limited subset of the user's permissions:

```text
agent can read only documents the user can read
agent can edit only documents the user can edit
agent can call only tools the app permits for this workflow
```

That gives you two checks:

```text
1. Can the user perform this action on this object?
2. Is this agent/tool allowed to perform this kind of action?
```

Example:

```text
User asks: "Update the roadmap with the new launch date."

Authn:
  request belongs to user:alice

Agent planning:
  agent wants to call update_document on document:roadmapDocument

Authz:
  Check(user:alice, can_edit, document:roadmapDocument)
  Check(agent:docAssistant, can_use, tool:update_document)

Only if both are allowed:
  execute the update
```

ReBAC can model these relationships too:

```text
team:platformTeam member user:alice
workspace:productWorkspace editor team:platformTeam#member
document:roadmapDocument workspace workspace:productWorkspace
tool:update_document can_use agent:docAssistant
```

The important production habit is the same as normal web apps:

```text
Do not trust the plan.
Authorize each tool call before it executes.
```

## Fail closed

Authorization should generally fail closed.

```text
if unsure -> deny
```

Do not silently allow access because a check failed.

Bad:

```text
OpenFGA timeout -> allow request
```

Better:

```text
OpenFGA timeout -> return error or deny according to explicit policy
```

## Tests you need

Good authorization tests include:

- allowed direct relationship
- allowed inherited relationship
- allowed subject-set relationship
- denied unrelated user
- near-miss denial
- service rejects denied action
- HTTP maps forbidden to 403

This repo has tests for those patterns.

## Exercise

Write the authorization question for these actions:

```text
Alice reads roadmap document.
Bob edits roadmap document.
Casey creates a document in workspace:productWorkspace.
```

Format:

```text
Can <subject> <action> <object>?
Check(<user>, <relation>, <object>)
```

Example:

```text
Can user:alice edit document:roadmapDocument?
Check(user:alice, can_edit, document:roadmapDocument)
```

## Checkpoint

Why does ReBAC fit collaborative documents better than global RBAC?

Good answer: collaborative documents need object-specific permissions that
follow relationships between users, teams, workspaces, and documents. ReBAC
models those relationships directly.
