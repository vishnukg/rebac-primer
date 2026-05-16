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
Can the workspace viewer edit only documents in workspace:productWorkspace?
Can the workspace editor edit documents because she is in team:platformTeam?
Can the outside collaborator read one shared document but nothing else?
```

Now global roles are not enough.

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
│ Authenticate │ "this is user:workspaceEditor"
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Authorize    │ "can user:workspaceEditor edit document:roadmapDocument?"
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Execute      │ update the document
└──────────────┘
```

Do not skip authorization just because authentication succeeded.

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
user:workspaceEditor -> role:editor
role:editor -> permission:edit_document
```

Diagram:

```text
user:workspaceEditor
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
user:workspaceEditor member team:platformTeam
team:platformTeam editor workspace:productWorkspace
document:roadmapDocument workspace workspace:productWorkspace
```

Diagram:

```text
user:workspaceEditor -> team:platformTeam -> workspace:productWorkspace -> document:roadmapDocument
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
team:platformTeam member user:workspaceEditor
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
│ Authorizer   │ GraphAuthorizer or OpenFgaAuthorizer
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
The workspace editor reads roadmap document.
The workspace viewer edits roadmap document.
The outside collaborator creates a document in workspace:productWorkspace.
```

Format:

```text
Can <subject> <action> <object>?
Check(<user>, <relation>, <object>)
```

Example:

```text
Can user:workspaceEditor edit document:roadmapDocument?
Check(user:workspaceEditor, can_edit, document:roadmapDocument)
```

## Checkpoint

Why does ReBAC fit collaborative documents better than global RBAC?

Good answer: collaborative documents need object-specific permissions that
follow relationships between users, teams, workspaces, and documents. ReBAC
models those relationships directly.
