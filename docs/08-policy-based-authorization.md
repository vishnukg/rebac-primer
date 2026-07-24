# Policy-Based Authorization

Every serious authorization system is "policy based" in one sense: somewhere
there is a policy, and decisions should follow it. This chapter is about the
stricter meaning:

```text
Policy-based authorization
  The rules that grant or deny access are a declarative artifact,
  separate from application code, evaluated by an engine.
```

You have already been using this idea without the name. The OpenFGA model in
`deployments/openfga/model.fga` is a policy. The rule tables in
`internal/authz/model.go` are a policy. The graph evaluator in
`internal/authz/evaluator.go` is a policy engine. This chapter makes that
explicit, shows how general-purpose policy engines (OPA, Cedar) work, and
explains where ReBAC fits in the policy-engine landscape.

Read this after doc 07. By the end you should be able to:

- explain the code/policy/data split and why each piece exists
- name the standard architecture roles (PEP, PDP, PIP, PAP) and point at them
  in this repo
- describe how a policy engine evaluates a decision
- write the same product rule in a hardcoded style, OPA/Rego, Cedar, and the
  OpenFGA DSL, and explain the trade-offs
- decide which layer a new rule belongs in

This is a later comparison chapter, not a prerequisite for understanding the
repository. If you are still learning tuples and usersets, skip it until you can
explain the Alice decision without looking at the code.

## The Problem: Authorization Written as Code

Start with the version every codebase grows first:

```go
func (h *Handler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
    user := currentUser(r)
    doc := loadDocument(r)

    ws := loadWorkspace(doc.WorkspaceID)
    allowed := false
    if doc.OwnerID == user.ID {
        allowed = true
    }
    for _, teamID := range ws.EditorTeamIDs {
        if isTeamMember(user.ID, teamID) {
            allowed = true
        }
    }
    if !allowed {
        http.Error(w, "forbidden", http.StatusForbidden)
        return
    }
    // ... update the document
}
```

This works, and for a small app it may be fine. It degrades in predictable
ways:

```text
Scattered      every handler re-implements a slice of the rules
Inconsistent   two handlers disagree about what "editor" means
Unanswerable   "who can edit this document?" requires reading all the code
Unreviewable   security cannot audit rules spread across handlers
Coupled        every product rule change is an application deploy
Untestable     rules can only be tested through full request setups
```

The fix is not "move the if-statements into one function," although that
helps. The fix is to recognize that three different things are tangled
together and to separate them.

## The Core Split: Code, Policy, Data

Every authorization decision is the same computation:

```text
decision = evaluate(policy, data, request)
```

| Piece | What it is | Example here | Changes when |
|---|---|---|---|
| request | the question | Can `user:alice` `can_edit` `document:roadmapDocument`? | every request |
| data | facts about the world | `user:alice member team:platformTeam` | users act (join a team, share a doc) |
| policy | rules deriving decisions from facts | `can_edit: editor`, `editor: ... or editor from workspace` | the product's rules change |
| code (engine) | evaluates policy against data | graph evaluator, OpenFGA | the engine grows a capability |

The rate-of-change column is the whole argument. In the hardcoded handler,
all four pieces change at the same cadence: an application deploy. After the
split:

```text
A user joins a team          -> write one tuple            (data)
"Commenters can react"       -> edit the model, redeploy   (policy)
"Support intersection rules" -> engine work                (code)
```

Each kind of change gets the review, testing, and rollout process it
deserves. Policy changes get security review and shadow testing (doc 07,
Policy Lifecycle). Data changes are ordinary product writes. Engine changes
are rare.

A useful test when you cannot decide where something goes:

```text
Does it change per customer action?        -> data
Does it change per product decision?       -> policy
Does it change the kinds of rules you can
express at all?                            -> code
```

Doc 07's "From Requirements to Policy" table applies exactly this test to
this repo's product sentences.

## The Standard Vocabulary: PEP, PDP, PIP, PAP

These four roles come from the XACML standard, and vendors use them
everywhere. Doc 07 introduced the first three briefly; here is the full set:

```text
┌───────────────────────────────────────────────────────────┐
│  PAP  Policy Administration Point                         │
│       where policy is authored, versioned, and deployed   │
└───────────────────────┬───────────────────────────────────┘
                        │ publishes policy
                        ▼
┌──────────────┐  ask   ┌──────────────┐  fetch  ┌──────────┐
│  PEP         │ ─────► │  PDP         │ ──────► │  PIP     │
│  Policy      │        │  Policy      │         │  Policy  │
│  Enforcement │ ◄───── │  Decision    │ ◄────── │  Info    │
│  Point       │ allow/ │  Point       │  facts  │  Point   │
└──────────────┘  deny  └──────────────┘         └──────────┘
 blocks or permits       evaluates policy         supplies facts
 the operation           against facts            the PDP needs
```

In this repository:

| Role | This repo (custom evaluator) | This repo (OpenFGA backend) |
|---|---|---|
| PEP | `documents.Service` calling `Authorizer.Check`; HTTP layer maps deny to 403 | same |
| PDP | `authz.GraphEvaluator` | the OpenFGA server's Check API |
| PIP | `authz.InMemoryStore` through relationship ports | OpenFGA's tuple storage |
| PAP | rule tables in `internal/authz/model.go`, changed via code review | `model.fga` plus `deployments/openfga/seed.sh` and model IDs |

Two practical notes that matter more than the acronyms:

1. The PEP must be the domain service, not the UI and not only the HTTP
   handler. `documents.Service` is the enforcement point because it knows
   which business operation is being attempted (doc 02, Mistake 4).
2. The PDP/PIP boundary is where engine families genuinely differ. Keep it
   in mind for the rest of the chapter: **who brings the facts to the
   decision?**

## How a Policy Engine Evaluates

Whatever the language, a policy engine's life has two phases.

At policy load time:

```text
parse      policy text -> internal form (AST / compiled rules)
validate   types, unknown relations, schema conformance
version    the compiled policy gets an identity
```

At decision time:

```text
bind       put the request into the policy's input vocabulary
evaluate   determine which rules apply and whether they are satisfied
combine    merge rule results into one decision
explain    optionally record why
```

Three evaluation concepts are worth learning because they recur in every
engine you will meet.

### Default deny

A request that matches no rule must be denied. Every engine in this chapter
behaves this way: Cedar denies unless a `permit` matches, a Rego `default
allow := false` makes the fallback explicit, OpenFGA's Check returns
`allowed: false` when no relationship path exists, and this repo's evaluator
returns denied when every branch is exhausted. If you ever build your own
engine, this is the first invariant to test.

### Combining rules

When several rules apply, something must merge them. The common algorithms
(the names come from XACML):

```text
permit-overrides   any matching permit allows          (a union of grants)
deny-overrides     any matching deny wins              (exceptions beat grants)
first-applicable   rule order decides
```

Most sharing-style authorization is permit-overrides: access is a union of
grants, and finding any one grant is enough. That is exactly what "or" means
in the OpenFGA model and what "if any branch returns true, the whole check is
allowed" means in the graph evaluator. Cedar layers deny-overrides on top:
`forbid` policies always beat `permit` policies, which is how you express
"blocked users cannot read, even if they are editors."

### Goal-directed evaluation

There are two directions an engine can work:

```text
Forward    compute everything derivable from the facts,
           then look up the answer
Backward   start from the question, expand only the rules
           and facts that could answer it
```

Authorization engines are almost always backward (goal-directed): the
question `Check(user:alice, can_edit, document:roadmapDocument)` arrives
first, and the engine expands `can_edit -> editor -> owner / workspace
editor -> team member` only as far as needed, short-circuiting on the first
success. Re-read the big trace comment above `hasRelation` in
`internal/authz/evaluator.go` with this frame: it is a textbook
backward-chaining evaluation, and OpenFGA's Check resolves the same way.

Forward evaluation appears in one place in practice: materializing derived
permissions into storage so lookups are trivial. Doc 07 explains why that
trades write amplification and slow revocation for read speed; ReBAC systems
avoid it by default.

## One Rule, Four Notations

Nothing demystifies policy engines faster than writing the same rule in each
of them. The product rule, from doc 02:

```text
Members of team:platformTeam can edit documents in
workspace:productWorkspace.
```

The facts involved:

```text
alice is a member of platformTeam
platformTeam's members are editors of productWorkspace
roadmapDocument belongs to productWorkspace
```

### Notation 0: hardcoded

The handler at the top of this chapter. Rules and facts and enforcement all
live in code. Included as the baseline the others are escaping.

### Notation 1: OPA and Rego

[OPA](https://www.openpolicyagent.org/) (Open Policy Agent) is a
general-purpose policy engine. Policies are written in Rego, a Datalog-family
language. The application (PEP) sends an `input` document; the engine
evaluates rules against `input` plus a `data` document it has been given, and
returns the value of the rule you queried.

The policy:

```rego
package documents

import rego.v1

default can_edit := false

# Direct document editor.
can_edit if {
    input.user in data.document_editors[input.document]
}

# Workspace editor through team membership.
can_edit if {
    workspace := data.document_workspace[input.document]
    some team in data.workspace_editor_teams[workspace]
    input.user in data.team_members[team]
}
```

The data OPA must hold (or receive):

```json
{
  "document_workspace":    {"roadmapDocument": "productWorkspace"},
  "workspace_editor_teams": {"productWorkspace": ["platformTeam"]},
  "team_members":          {"platformTeam": ["alice"]}
}
```

The request:

```json
{"user": "alice", "document": "roadmapDocument"}
```

How to read the Rego: each `can_edit if { ... }` block is one way to satisfy
the rule, and multiple blocks union (permit-overrides). Inside a block, the
statements are conditions that must all hold, with `some` introducing a
search ("there exists a team such that..."). This is the same "OR of ANDs"
shape as the OpenFGA model, in expression form.

Two properties to notice, because they matter later:

1. **The traversal depth is written into the policy.** The second rule walks
   exactly document -> workspace -> team -> user. If the product adds teams
   within teams, the policy needs another rule (or a rewrite). Rego
   deliberately **prohibits recursive rules** — the compiler rejects them —
   so you cannot write "member of a group or of any nested subgroup" the
   natural way. The escape hatch is the `graph.reachable` built-in, which
   computes reachability over a graph you supply as data.
2. **Someone must ship the facts to the engine.** OPA evaluates against data
   it holds in memory (pushed or pulled as bundles) or data included in the
   input. Keeping every team membership replicated into a policy engine,
   fresh enough for revocation SLOs, is a real synchronization system you
   now own (doc 07, Data Ownership and Synchronization).

### Notation 2: Cedar

[Cedar](https://www.cedarpolicy.com/) is the policy language behind AWS
Verified Permissions, designed around the shape
`permit(principal, action, resource) when {condition}`. The application
passes the request plus a slice of relevant **entities**; entities can have
attributes and `parents`, and the `in` operator tests (transitive) hierarchy
membership over the parents you supplied.

The policy:

```cedar
permit (
  principal,
  action == Action::"editDocument",
  resource
) when {
  principal in resource.workspace.editors
};
```

The entities the application sends with the request:

```json
[
  {"uid": {"type": "User", "id": "alice"},
   "parents": [{"type": "Team", "id": "platformTeam"}]},

  {"uid": {"type": "Team", "id": "platformTeam"}, "parents": []},

  {"uid": {"type": "Workspace", "id": "productWorkspace"},
   "attrs": {"editors": [{"type": "Team", "id": "platformTeam"}]},
   "parents": []},

  {"uid": {"type": "Document", "id": "roadmapDocument"},
   "attrs": {"workspace": {"type": "Workspace", "id": "productWorkspace"}},
   "parents": []}
]
```

Reading the condition: `resource.workspace.editors` follows attributes to a
set of entities (`[Team::"platformTeam"]`), and `principal in <set>` is true
because alice's parents include the platform team. Cedar's combining
algorithm is fixed and safe: any matching `forbid` beats every `permit`, and
no match at all means deny.

The property to notice: **the application assembles the relationship data
per request.** Cedar does not go fetch alice's teams; the PEP must know
which entities could matter and include them (including ancestors, for `in`
to see them). The engine is simple and fast because the hard problem —
gathering the right slice of the relationship graph — was handed to the
caller. (AWS Verified Permissions adds managed storage for policies, but the
entity-slice model is the same.)

### Notation 3: OpenFGA

The model you already know from doc 05 (`deployments/openfga/model.fga`):

```text
type workspace
  relations
    define editor: [user, team#member] or owner

type document
  relations
    define workspace: [workspace]
    define editor: [user] or editor from workspace or owner
    define can_edit: editor
```

The facts, as tuples in the engine's own store:

```text
user:alice                 member    team:platformTeam
team:platformTeam#member   editor    workspace:productWorkspace
workspace:productWorkspace workspace document:roadmapDocument
```

The request:

```text
Check(user:alice, can_edit, document:roadmapDocument)
```

The model is the policy; the tuples are the data; Check is the decision API.
ReBAC engines are policy engines — specialized ones.

### What the comparison teaches

| | Hardcoded | OPA/Rego | Cedar | OpenFGA |
|---|---|---|---|---|
| Policy lives | in code | Rego modules | Cedar policies | authorization model |
| Facts live | app database | OPA's data (replicated) or input | entity slice per request | engine's tuple store |
| Who fetches facts | the handler | you, ahead of time | the PEP, per request | the engine, during evaluation |
| Nested/recursive relationships | hand-written loops | fixed-depth rules; recursion prohibited; `graph.reachable` escape hatch | via `parents`, closure per request slice | native graph traversal |
| "List what alice can edit" | scan and re-check | partial evaluation (advanced) | scan and re-check | ListObjects API |
| Contextual conditions (time, IP) | trivial | native strength | native strength | CEL conditions on tuples |

The rows are not ranked. They show one real fork:

```text
General policy engines: evaluate expressions over facts the caller
(or a replication pipeline) supplies.

ReBAC engines: evaluate path/set expressions over a relationship
graph the engine itself stores and traverses.
```

Which brings us to why that fork exists.

## Why ReBAC Wants Its Own Engine

Look again at what the decision needs:

```text
Check(user:alice, can_edit, document:roadmapDocument)
```

The answer is "there exists a policy-admitted path from the document's
`can_edit` userset to alice." Before evaluation starts, nobody knows which
facts are relevant: alice's access might come via a direct grant, any of her
teams, teams containing those teams, or the workspace hierarchy. The set of
facts needed is discovered *during* evaluation, one hop at a time.

That breaks the general engines' input models:

- Cedar-style "caller sends the entities": the PEP would have to fetch
  alice's teams, their parent teams, the document's workspace chain, and the
  grants on all of them — the PEP is now doing the graph traversal itself,
  which was the engine's job.
- OPA-style "replicate the data in": possible, but you are now running a
  synchronization pipeline for your most security-critical data, and
  revocation is only as fast as replication.

ReBAC engines resolve this by co-locating the data with the evaluator: the
engine owns tuple storage, so evaluation can fetch exactly the tuples the
traversal needs, when it needs them. That single design decision explains
most of what makes OpenFGA/Zanzibar-style systems distinctive:

```text
engine stores the tuples
  -> engine can traverse on demand           (recursive groups just work)
  -> engine can index for reverse expansion  (ListObjects, ListUsers)
  -> engine must offer write APIs            (tuple writes, doc 07 ownership)
  -> engine must define consistency          (zookies / consistency modes)
```

And the converse explains general engines: when a decision depends only on
facts the request already carries (token claims, resource attributes, time,
network), no stored graph is needed, and an expression evaluator with the
facts in hand is simpler, faster, and easier to operate.

```text
Decision depends on request-local attributes  -> policy-expression engine
Decision depends on stored relationships      -> relationship engine
```

Most products eventually have both kinds of rules. That is the hybrid
section below.

## Implementing ReBAC on a General Policy Engine

You may face this at work: "we already run OPA — can we do ReBAC in it?"
The honest answer is "up to a point," and the point is worth knowing
precisely. There are four strategies.

**1. Fixed-depth rules.** Write one rule per admitted path shape, as in the
Rego example above. Works when the graph is shallow and the shapes are
stable: user -> team -> workspace -> document and nothing deeper. Breaks the
day the product ships nested teams or folders-in-folders, because the policy
language cannot recurse; every new depth is a policy rewrite.

**2. Materialize the closure.** Precompute derived facts (all effective
members of every team; all effective editors of every document) and give the
engine flat data, or encode them as Cedar parents. Reads are trivial.
The costs are the ones doc 07 warns about for storing implied permissions:
write amplification (one team change fans out to many derived rows), and
revocation that is only as fast as your recomputation pipeline.

**3. Fetch-then-decide.** The PEP queries the product database for the
relevant relationships and passes them as input. Honest and common, but
notice what happened: deciding *which* facts to fetch is itself the
traversal, so the authorization logic is now split between the PEP's fetch
code and the engine's policy — two places to keep correct, and the policy
alone is no longer reviewable.

**4. Use both engines for what each is good at.** Keep relationships in a
relationship engine and treat it as a PIP for the policy engine (or simply
sequence the checks). This is the hybrid pattern, next section.

Signals that a general engine alone has run out of road:

```text
relationship data is deep or user-shaped (nested groups, folders)
"list everything alice can see" is a product requirement
revocation SLOs are tighter than your data-replication lag
policy rewrites are being driven by data-shape changes, not rule changes
```

The reverse door also exists and is easier: a tuple/userset model with
CEL conditions (OpenFGA supports conditions on tuples) covers many
"relationship plus a contextual condition" rules without a second engine.

## The Hybrid: Layered Decisions

Real requests pass through several gates, each owned by the layer that has
the facts:

```text
HTTP request
  |
  v
authenticate                     "this is user:alice"      (OIDC, doc 01)
  |
  v
coarse gate                      token has documents:write (scope check, code)
  |
  v
contextual policy (optional)     inside business hours,    (OPA/Cedar-style)
                                 from a trusted network
  |
  v
relationship check               Check(alice, can_edit,    (ReBAC engine)
                                 document:roadmapDocument)
  |
  v
execute
```

Deciding which layer owns a new rule is a design skill. The question to ask:
**what facts does the rule need, and who has them?**

| Rule | Facts needed | Layer |
|---|---|---|
| caller may use the documents API at all | token scopes | code at the boundary |
| no writes from untrusted networks | request context | contextual policy |
| editors can edit their workspace's documents | stored relationships | ReBAC model |
| alice is on the platform team | a fact, not a rule | tuple (data) |
| share link expires after 7 days | relationship + time | ReBAC with a condition (CEL), or contextual policy |
| blocked users lose access regardless of grants | relationships, deny semantics | ReBAC exclusion (`but not blocked`) — see doc 07 on the testing cost |

Keep the gates independent and fail-closed, and remember the testing warning
from doc 07: a request denied by the scope gate proves nothing about whether
the ReBAC model would have denied it. Test each layer for its own denials.

## This Repo's Evaluator, Reread as a Policy Engine

If policy engines still feel abstract, the cure is in this repository: the
custom evaluator is a policy engine small enough to read in one sitting.
Map the anatomy from earlier onto the code:

| Policy-engine concept | Where it is in this repo |
|---|---|
| policy language | four constructs: direct grant, subject set, implied-by rule, workspace inheritance |
| compiled policy | the `impliedBy` tables in `internal/authz/model.go` |
| PAP | editing those tables in a reviewed Go change |
| PIP / facts | `InMemoryStore` through `RelationshipReader` (`internal/authz/store.go`) |
| decision request | `rebac.CheckRequest` |
| evaluation | `hasRelation` in `internal/authz/evaluator.go` — backward chaining with short-circuit union |
| combining algorithm | permit-overrides: any successful branch allows |
| default deny | all branches exhausted -> denied |
| evaluation limits | cycle detection plus `defaultMaxDepth` |
| explanation | the `Trace` in `CheckResult` |
| policy versioning | none — the gap doc 26 discusses when migrating to OpenFGA's immutable model IDs |

The four steps `hasRelation` tries in order are the policy language. Each
step is one rule form the engine knows how to evaluate:

```text
step 1  direct tuple          "alice was granted editor on this document"
step 2  subject set           "a group was granted it, and alice is in the group"
step 3  implied-by rule       "a stronger relation she holds implies it"
step 4  parent inheritance    "the parent workspace granted it"
```

OpenFGA's model language is a richer version of the same idea — union,
intersection, exclusion, arbitrary type-to-type inheritance, conditions —
compiled from the DSL instead of hardcoded as Go tables, with the evaluation
loop generalized accordingly. When you read `model.fga` now, read it as a
program for that interpreter.

This reframing is also the practical skill for work: when you meet any
authorization product, locate its answers to the policy-engine questions —
What is the policy language? Where do facts live and who fetches them? What
is the combining algorithm? What are the evaluation limits? How is policy
versioned? How are decisions explained? Those six answers are the product.

## Exercise

For each product rule, decide: application code, contextual policy
(OPA/Cedar-style expression), ReBAC model change, or tuple write. Name the
facts the rule needs and who has them.

```text
1. Only requests with the documents:write scope may call write endpoints.
2. Members of a team can comment on documents in that team's workspaces.
3. Dana joins team:platformTeam.
4. Documents marked "legal hold" cannot be deleted by anyone.
5. Access from outside the corporate network is read-only.
6. A share link grants viewer access for 7 days.
```

Suggested answers (reason before reading):

```text
1. code at the boundary — facts: token claims; the request has them
2. model — a can_comment permission derived over existing relations
3. tuple — a fact changed, no rule changed
4. model (exclusion) or code guard — pick deliberately; exclusions carry
   testing cost, a code guard splits the policy; both defensible
5. contextual policy — facts: request network; ReBAC never sees them
6. tuple with a condition (expiry) — relationship plus time context
```

## Checkpoint

You understand policy-based authorization when you can answer:

- Why do code, policy, and data deserve different change cadences, and what
  goes wrong when they share one?
- Which PEP in this repo enforces document permissions, and why is it the
  domain service rather than the HTTP handler?
- What does default deny mean, and where does the graph evaluator implement
  it?
- Why does the Rego version of the workspace rule encode the traversal depth,
  and what breaks when nesting deepens?
- Why does an OpenFGA-style engine store the tuples itself instead of
  accepting them per request like Cedar?
- Where would you put "no deletes outside business hours," and why?

Next: continue with the [graph evaluator walkthrough](27-graph-evaluator-walkthrough.md)
to see this chapter's PDP evaluate a real check line by line.

## Sources and further study

Architecture and standards:

- [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)
- [XACML 3.0 specification](https://docs.oasis-open.org/xacml/3.0/xacml-3.0-core-spec-os-en.html)
  — origin of PEP/PDP/PIP/PAP and the combining algorithms
- [NIST SP 800-162 (ABAC)](https://csrc.nist.gov/pubs/sp/800/162/upd2/final)

General policy engines:

- [OPA documentation](https://www.openpolicyagent.org/docs/latest/)
- [Rego policy language](https://www.openpolicyagent.org/docs/latest/policy-language/)
  — including the no-recursion rule and `graph.reachable`
- [Cedar language guide](https://docs.cedarpolicy.com/)
- [Cedar: entities and the `in` operator](https://docs.cedarpolicy.com/policies/syntax-operators.html)

ReBAC engines as policy systems:

- [Zanzibar: Google's Consistent, Global Authorization System](https://www.usenix.org/conference/atc19/presentation/pang)
- [OpenFGA configuration language](https://openfga.dev/docs/configuration-language)
- [OpenFGA conditions (CEL)](https://openfga.dev/docs/modeling/conditions)
- [Common Expression Language (CEL)](https://github.com/cel-expr/cel-spec)
