# OAuth and OIDC: who is this person?

Before your app can ask "can Alice edit this document?" it must know the request
actually came from Alice. That is the login problem. OAuth and OIDC are the
modern standards for solving it — without your app ever handling Alice's password
directly.

## How to read this chapter

This chapter contains both the essential identity handoff and production-depth
material. On a first pass:

1. read through Authorization Code + PKCE
2. jump to [From identity to ReBAC](#from-identity-to-rebac)
3. read Scopes vs. ReBAC, patterns to avoid, and the checkpoint

Return later for client credentials, device authorization, token validation,
token exchange, and service-to-service identity. Those sections are important,
but they are not prerequisites for understanding the graph evaluator.

## Two acronyms, one sentence each

**OAuth 2.0** — a framework for delegated access: a user authorizes your app to
call APIs on their behalf without giving your app their password.

**OpenID Connect (OIDC)** — an identity layer built on top of OAuth: it adds a
standard way for your app to learn *who* the user is, not just *that* they
authenticated.

Many "log in with ..." integrations use both:

- OAuth provides delegated authorization and token delivery.
- OIDC adds the interoperable authentication and identity layer.

## Authentication vs. authorization

```text
Authentication  →  Who are you?
Authorization   →  What can you do?
```

OIDC handles user authentication. OAuth access tokens authorize API access.
Your ReBAC system handles application-specific, object-level authorization.
They hand off to each other:

```text
OIDC/token validation establishes: "This request is from user:alice"
                            │
                            ▼
ReBAC decides:       "Can user:alice edit document:roadmapDocument?"
                            │
                            ▼
                     allow or deny
```

Keep these questions separate. OAuth scopes do not decide document permissions,
and OAuth by itself is not an authentication protocol. OIDC supplies the login
identity; the resource server validates the access token presented to its API.

## The four players

OAuth authorization flows use four roles:

```text
Resource Owner       Alice — the user who wants to use the app
Client               your app
Authorization Server the IdP (GitHub, Auth0, Google) that handles login
                     and issues tokens
Resource Server      the API your app calls, protected by tokens
```

In a small app, your backend is both the Client and the Resource Server. That is
fine — they are different roles in the protocol, not necessarily different
servers.

An Authorization Server is also acting as an **Identity Provider (IdP)** when it
implements OIDC. The roles are related, but not interchangeable in every system.

## OAuth flows

Different app shapes have different security constraints, so OAuth defines
several flows. A browser app can redirect the user. A background service has no
user at all. A CLI running on a TV cannot open a browser. Each flow is the right
tool for one of those shapes.

The three flows worth knowing:

```text
Authorization Code + PKCE  →  any app where a user logs in
Client Credentials         →  machine-to-machine, no user present
Device Authorization       →  CLIs and devices that cannot open a browser
```

## Flow 1: Authorization Code + PKCE

**Use when:** a user needs to log in — browser apps, server apps, mobile apps,
desktop apps.

Here is a typical login, step by step:

```text
Browser                  Your App                  IdP
   │                         │                       │
   │  1. GET /login          │                       │
   │────────────────────────►│                       │
   │                         │ generates:            │
   │                         │  state (random nonce) │
   │                         │  OIDC nonce           │
   │                         │  code_verifier        │
   │                         │  code_challenge       │
   │                         │  = S256(verifier)     │
   │  2. 302 → /authorize    │                       │
   │     ?response_type=code │                       │
   │     &client_id=...      │                       │
   │     &redirect_uri=...   │                       │
   │     &scope=openid       │                       │
   │     &state=<nonce>      │                       │
   │     &nonce=<oidc_nonce> │                       │
   │     &code_challenge=... │                       │
   │     &code_challenge_    │                       │
   │       method=S256       │                       │
   │◄────────────────────────│                       │
   │  3. follows redirect    │                       │
   │────────────────────────────────────────────────►│
   │                         │    Alice logs in      │
   │  4. 302 → /callback     │                       │
   │     ?code=AUTH_CODE     │                       │
   │     &state=<nonce>      │                       │
   │◄────────────────────────────────────────────────┤
   │  5. GET /callback       │                       │
   │     ?code=AUTH_CODE     │                       │
   │     &state=<nonce>      │                       │
   │────────────────────────►│                       │
   │                         │ verify state matches  │
   │                         │  6. POST /token       │
   │                         │  grant_type=          │
   │                         │   authorization_code  │
   │                         │  code=AUTH_CODE       │
   │                         │  code_verifier=...    │
   │                         │  redirect_uri=...     │
   │                         │  + client auth when   │
   │                         │    confidential       │
   │                         │──────────────────────►│
   │                         │                       │ verify
   │                         │                       │ code_verifier
   │                         │  7. access_token,     │
   │                         │     id_token          │
   │                         │◄──────────────────────┤
   │  8. Set-Cookie: session │                       │
   │◄────────────────────────│                       │
```

In plain English:

1. Alice's browser requests `/login` from your app
2. Your app generates `state` (request/callback correlation and CSRF defense),
   an OIDC `nonce` (binds the ID token to this login), a random
   `code_verifier`, and an S256 `code_challenge`. It responds with a redirect to
   the authorization endpoint carrying the public values. The browser follows.
3. Alice's browser arrives at the IdP. She logs in (password, MFA, etc.) — your
   app is not involved and never sees her credentials.
4. The IdP redirects Alice's browser back to your `redirect_uri` (your app's
   `/callback` route), adding the one-time authorization `code` and the same
   `state`.
5. Alice's browser follows that redirect to your callback route.
6. Your app verifies the `state` matches what it stored (prevents CSRF), then
   makes a **server-to-server** POST to the IdP's token endpoint — never in the
   browser URL — sending the `code` and the original `code_verifier`. A
   confidential server-side client also authenticates using its registered
   method, such as `private_key_jwt`, mTLS, or a client secret. PKCE does not
   replace client authentication. Public clients such as SPAs and native apps
   cannot safely keep a client credential.
7. The authorization server verifies the PKCE value and returns an access token
   plus an ID token for the OIDC request. The client validates the ID token,
   including issuer, audience, signature, expiry, and nonce.
8. Your app creates a session cookie. Alice is logged in.

Your app never sees Alice's password. The token exchange in step 6 is
server-to-server, so access and refresh tokens do not appear in the browser URL.
The short-lived authorization code does appear in the callback URL and must
still be protected against leakage and replay.

## What are these tokens?

Step 7 gives you up to three things:

```text
Authorization code  →  short-lived, one-time credential (like a coat-check ticket)
                        your app exchanges this for real tokens — then it expires

Access token        →  carries delegated authority for a resource server
                        commonly sent as a Bearer token
                        should be short-lived according to the threat model

ID token            →  OIDC's contribution — proves "this person authenticated"
                        always contains a subject identifier; profile and email
                        claims are optional and depend on scopes and provider
                        your app reads this to learn who just logged in

Refresh token       →  optional long-lived token used to get new access tokens
                        so Alice doesn't have to log in again every hour
```

The critical distinction:

```text
ID token     →  tells YOUR APP who the user is
Access token →  tells THE API that the client is authorized to call it
```

Do not use the ID token to call APIs. At the API, use the validated access
token's subject and authorization details according to the authorization
server's documented token profile. Do not assume an arbitrary access token is
an OIDC ID token or that it always contains user identity claims.

## OIDC: the "who" layer

OAuth defines delegated API authorization, not a login protocol. OIDC adds the
standard authentication response and identity claims needed for login.

OIDC adds the ID token — a JWT containing standard **claims** about the user:

```json
{
  "sub": "github|12345",
  "name": "Alice",
  "email": "alice@example.com",
  "iss": "https://accounts.google.com",
  "aud": "your-app-client-id",
  "exp": 1893456000
}
```

Common claims:

```text
sub   subject — stable, locally unique identifier within the issuer
iss   issuer  — who issued this token
aud   audience — who this token is intended for (your app)
exp   expiration timestamp
```

The identity key is the pair `(iss, sub)`: `sub` is locally unique and stable
within an issuer, but is not globally unique by itself. Providers may also issue
pairwise subject values that differ between clients. Map the trusted issuer and
subject pair to an internal app user id. Use that mapping — not email — because
emails change and may have separate verification semantics.

In your app:

```text
OIDC identity:
  iss = "https://auth.example.com/"
  sub = "github|12345"

(trusted iss, sub)  →  internal app user id "user:01JABC..."
```

## PKCE: why the code alone isn't enough

PKCE (pronounced "pixie") stands for Proof Key for Code Exchange. It solves a
real threat: what if someone intercepts the authorization code from step 4?

Without PKCE, stealing the code is enough to get tokens. With PKCE, the code is
useless without a secret your app generated locally.

How it works:

```text
Before redirect:
  Your app generates:  code_verifier  (random secret, kept in memory)
                       code_challenge (hash of code_verifier, sent to IdP)

Step 2 — redirect to IdP sends:
  code_challenge

Step 6 — token exchange sends:
  the original code_verifier

IdP verifies:  hash(code_verifier) == stored code_challenge → ok
```

An attacker who intercepts the authorization code in step 4 does not have the
`code_verifier`, so the token exchange fails. Use PKCE for all new apps.

```text
Client                           Authorization Server
  │                                       │
  │  auth request + code_challenge        │
  ├──────────────────────────────────────►│
  │                                       │  user logs in
  │  authorization code                   │
  │◄──────────────────────────────────────┤
  │  token request + code_verifier        │
  ├──────────────────────────────────────►│
  │                                       │  verifies hash matches
  │  access token + ID token              │
  │◄──────────────────────────────────────┤
```

## Flow 2: Client Credentials

> Optional depth begins here. If your immediate goal is ReBAC, jump to
> [From identity to ReBAC](#from-identity-to-rebac).

**Use when:** a service needs to call another service and there is no user
involved — background jobs, billing workers, microservice calls.

Instead of redirecting a browser, the service authenticates directly with its
own credentials:

```text
Service A                         Authorization Server
   │                                       │
   │  POST /token                          │
   │  grant_type=client_credentials        │
   │  client_id=svc-billing                │
   │  client_secret=<secret>               │
   │  scope=documents.read                 │
   ├──────────────────────────────────────►│
   │                                       │  verify credentials
   │  {                                    │
   │    "access_token": "...",             │
   │    "token_type": "Bearer",            │
   │    "expires_in": 3600                 │
   │  }                                    │
   │◄──────────────────────────────────────┤
   │                                       │
   │  GET /api/resource                    │
   │  Authorization: Bearer <access_token> │
   ├──────────────────────────────────────► Resource Server (API)
```

In plain English:

1. Service A POSTs its `client_id` and `client_secret` to the IdP's token
   endpoint with `grant_type=client_credentials`
2. The IdP verifies the credentials and returns an access token
3. Service A attaches that token to every API call as a `Bearer` token

No browser. No user. No redirect. No ID token — there is no user to identify.

The access token proves "Service A is authorized to call this API." The API can
still make authorization decisions about what the service is allowed to do.

ReBAC can model service identities too:

```text
service:billing-worker  →  can_read on document:invoiceTemplate
```

This repository models only `user`, `team`, `workspace`, and `document`, so the
example would require adding a `service` type. In production, distinct principal
types make policy and audit records clearer than placing machines under `user:`.

## Flow 3: Device Authorization

**Use when:** the app cannot open a browser — CLIs, smart TVs, IoT devices,
game consoles.

The problem these devices share: they can display text but cannot redirect the
user's browser to the IdP. The Device Authorization flow solves this by
splitting authentication across two devices.

Two codes are involved and they serve different purposes:

```text
device_code  →  long opaque token, used by the CLI to poll the IdP
user_code    →  short human-readable code (e.g. ABCD-1234), shown to the user
```

The user never sees the `device_code`. The CLI never shows the `user_code` to
the IdP — it only uses the `device_code` for polling.

```text
CLI / Device                Authorization Server         User's browser
      │                               │                         │
      │  1. POST /device_authorization│                         │
      │  client_id=my-cli             │                         │
      ├──────────────────────────────►│                         │
      │  2. {                         │                         │
      │    "device_code": "Ag1x...",  │  ← CLI uses this        │
      │    "user_code": "ABCD-1234",  │  ← user types this      │
      │    "verification_uri":        │                         │
      │      "example.com/activate",  │                         │
      │    "expires_in": 900,         │                         │
      │    "interval": 5              │  ← poll every 5s        │
      │  }                            │                         │
      │◄──────────────────────────────┤                         │
      │                               │                         │
      │  3. CLI prints:               │                         │
      │  "Open example.com/activate"  │                         │
      │  "Enter: ABCD-1234"           │                         │
      │                               │  4. user visits URL,    │
      │                               │     enters ABCD-1234,   │
      │                               │     logs in             │
      │                               │◄────────────────────────┤
      │  5. POST /token (poll #1)     │                         │
      │  grant_type=urn:ietf:params:  │                         │
      │   oauth:grant-type:device_code│                         │
      │  device_code=Ag1x...          │                         │
      ├──────────────────────────────►│                         │
      │  { "error":                   │                         │
      │    "authorization_pending" }  │  ← user not done yet    │
      │◄──────────────────────────────┤                         │
      │  ... waits 5 seconds, polls again                       │
      │  6. POST /token (poll #N)     │                         │
      ├──────────────────────────────►│                         │
      │  {                            │                         │
      │    "access_token": "...",     │                         │
      │    "id_token": "..."          │                         │
      │  }                            │  ← user finished        │
      │◄──────────────────────────────┤                         │
```

In plain English:

1. The CLI POSTs to the device authorization endpoint with its `client_id`
2. The IdP returns two codes and a polling interval
3. The CLI shows the user the `user_code` and the URL — nothing else to do on
   the CLI side yet
4. The user opens a browser on any other device, visits the URL, types the
   `user_code`, and logs in normally
5. Meanwhile the CLI polls the token endpoint every `interval` seconds using the
   `device_code`. The IdP returns `authorization_pending` until the user finishes.
6. Once the user completes login, the next poll returns an access token and,
   when OIDC was requested and supported, an ID token

The user authenticates on a device that *can* open a browser. The CLI receives
tokens without redirecting anything.

This pattern is commonly used by CLIs and input-constrained devices. A specific
tool may instead use Authorization Code with a loopback or claimed-HTTPS
redirect, so check that tool's current documentation.

## How the API validates an access token

Once the client has a token (from any flow), it attaches it to every API call:

```text
GET /api/documents/roadmap
Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleS0xIn0...
```

The API must verify the token before trusting it. There are two standard
approaches.

### Option 1: JWT validation (local)

Some authorization servers issue JWT access tokens. When the server documents
that token profile, the API can validate them locally without calling the
authorization server on every request. Other access tokens are intentionally
opaque and must not be decoded as JWTs.

The authorization server publishes trusted metadata, including its endpoints,
supported algorithms, issuer identifier, and `jwks_uri`. Configure the expected
issuer, fetch its metadata using the appropriate discovery mechanism, and do not
hard-code the example JWKS path below:

```text
GET https://idp.example.com/.well-known/jwks.json
```

The API fetches and caches these keys, and refreshes them safely when keys
rotate. When a request arrives:

```text
1. Require the authorization server's documented access-token profile
2. Parse the JWT header   →  get token type, algorithm, and key ID (kid)
3. Enforce the expected token type and permitted algorithms; reject alg=none
4. Find the matching public key from the cached JWKS and verify the signature
5. Check claims:
     exp  →  is the token still valid?
     nbf  →  if present, is the token active yet?
     iss  →  does it exactly match the configured issuer?
     aud  →  is it intended for this API?
     scope → does it include the permission this endpoint requires?
6. Interpret sub, client_id, scopes, and other claims according to that profile
7. Map the validated principal to an internal id  →  run the ReBAC check
```

No per-request authorization-server call is needed, but key rotation still
requires safe JWKS refresh and caching.

For access tokens following RFC 9068, the API also requires a `typ` header of
`at+jwt` or `application/at+jwt`. Explicit typing helps prevent an OIDC ID token
from being accepted as an access token.

For a user token (from Flow 1 or Flow 3), the JWT payload looks like this:

```json
{
  "sub": "github|12345",
  "iss": "https://auth.example.com/",
  "aud": "your-api",
  "scope": "documents.write",
  "exp": 1893456000
}
```

For a service token (from Flow 2 — Client Credentials), it looks like this:

```json
{
  "sub": "svc-billing-worker",
  "iss": "https://auth.example.com/",
  "aud": "your-api",
  "scope": "documents.read",
  "exp": 1893456000
}
```

The difference: in a user token `sub` commonly identifies the user, while a
client-credentials token represents the client or workload according to the
authorization server's documented profile. They can share validation
infrastructure, but they are different principal types with different policy
and audit semantics. Do not assume every provider places a client-credentials
identity in `sub`; use the profile's defined claims and map users and services
to distinct internal identities.

### Option 2: Token introspection (remote)

If the access token is opaque (not a JWT), or you need centrally evaluated token
status, the API calls the IdP's introspection endpoint (RFC 7662):

```text
Your API                         Authorization Server
   │                                      │
   │  POST /oauth/introspect              │
   │  Authorization: Basic <api_creds>    │
   │  token=<access_token>                │
   ├─────────────────────────────────────►│
   │                                      │
   │  {                                   │
   │    "active": true,                   │
   │    "sub": "github|12345",            │
   │    "scope": "documents.write",       │
   │    "exp": 1893456000                 │
   │  }                                   │
   │◄─────────────────────────────────────┤
```

If `active` is `false`, reject the request immediately.

### Which approach to use?

```text
JWT validation (local)
  + fast — no network call after keys are cached
  + scales — the API is self-contained
  - revocation takes until token expiry to take effect

Token introspection (remote)
  + can provide fresher status and detect revocation before token expiry
  - adds network calls unless responses are cached
  - the IdP becomes a bottleneck at scale
```

In practice, choose short access-token lifetimes from your threat model rather
than copying a universal number. JWT validation trades immediate revocation for
local verification. Introspection responses may be cached, so choose a cache
lifetime that explicitly balances revocation freshness against latency and
authorization-server load.

## When your API calls another service

Once your API validates the user token and knows who Alice is, it may need to
call a downstream service — a billing service, a notification service, a
documents service. This creates a problem.

### The audience problem

The user's access token has an `aud` (audience) claim set to *your* API:

```json
{ "sub": "github|12345", "aud": "your-api", "scope": "documents.write" }
```

If your API forwards that token to Service B, Service B should reject it —
the token was not issued for Service B. A strict API enforcing audience
validation will normally return `401 Unauthorized`.

Even if it does not enforce `aud` today, forwarding a user token to internal
services violates least-privilege: Service B receives more credential than it
needs, and a compromise of Service B exposes the user's full token.

```text
Browser ──user token (aud=your-api)──► Your API
Your API ──same user token──────────► Service B  ← aud mismatch, should fail
```

There are two proper patterns.

### Pattern 1: Token Exchange (RFC 8693)

Your API exchanges the user token for a new token specifically scoped for
Service B. In the standard delegation representation, `sub` remains the user
whose authority is being delegated and `act.sub` identifies the current actor
acting for that user.

```text
Browser   Your API (Resource Server)        Authorization Server     Service B
   │                   │                              │                  │
   │  user token       │                              │                  │
   │  aud=your-api     │                              │                  │
   │──────────────────►│                              │                  │
   │                   │ 1. validate user token       │                  │
   │                   │                              │                  │
   │                   │ 2. POST /token               │                  │
   │                   │  grant_type=                 │                  │
   │                   │   urn:ietf:params:oauth:     │                  │
   │                   │   grant-type:token-exchange  │                  │
   │                   │  subject_token=<user token>  │                  │
   │                   │  audience=service-b          │                  │
   │                   │  scope=invoices.read         │                  │
   │                   │─────────────────────────────►│                  │
   │                   │                              │ validates        │
   │                   │                              │ user token       │
   │                   │ 3. new token                 │                  │
   │                   │  {                           │                  │
   │                   │    "sub": "github|12345",    │                  │
   │                   │    "aud": "service-b",       │                  │
   │                   │    "act": {                  │                  │
   │                   │      "sub": "your-api"       │                  │
   │                   │    },                        │                  │
   │                   │    "scope": "invoices.read"  │                  │
   │                   │  }                           │                  │
   │                   │◄─────────────────────────────│                  │
   │                   │                              │                  │
   │                   │ 4. Authorization: Bearer new token              │
   │                   │────────────────────────────────────────────────►│
```

Service B sees:
- `sub=github|12345` — the user whose authority is delegated
- `act.sub=your-api` — the service currently acting for that user
- `aud=service-b` — this token was issued specifically for it
- `scope=invoices.read` — narrower scope than the original user token

Service B can run the ReBAC check using the delegated subject from `sub`, while
retaining `act.sub` for actor-aware policy and audit records.

Use token exchange when:
- Services are across trust boundaries
- You need to preserve the user's identity downstream
- Downstream services enforce audience validation

### Pattern 2: Client Credentials + user identity forwarding

Your API uses its own Client Credentials to get a service-level token for
calling Service B, and passes the user's identity as a separate field in the
request.

```text
Browser   Your API (Resource Server)        Authorization Server     Service B
   │                   │                              │                  │
   │  user token       │                              │                  │
   │──────────────────►│                              │                  │
   │                   │ 1. validate user token       │                  │
   │                   │    map (iss, sub) to user id │                  │
   │                   │                              │                  │
   │                   │ 2. POST /token               │                  │
   │                   │  grant_type=                 │                  │
   │                   │   client_credentials         │                  │
   │                   │  client_id=your-api          │                  │
   │                   │  client_secret=...           │                  │
   │                   │  scope=invoices.read         │                  │
   │                   │─────────────────────────────►│                  │
   │                   │  service token               │                  │
   │                   │  { "sub": "your-api",        │                  │
   │                   │    "aud": "service-b" }      │                  │
   │                   │◄─────────────────────────────│                  │
   │                   │                              │                  │
   │                   │ 3. Authorization: Bearer service_token          │
   │                   │    signed/internal user context                 │
   │                   │────────────────────────────────────────────────►│
```

Service B receives:
- A valid service token proving the caller is your API
- The user identity as authenticated internal request context

Service B must accept that identity only from authenticated, authorized callers
and must strip or overwrite any equivalent value supplied by an external
request. A plain forwarded header plus "internal network" trust is not enough.

Use client credentials + forwarding when:
- Services are within one controlled trust boundary
- You authenticate the calling workload and integrity-protect the user context
- You want simplicity over the overhead of token exchange

### What NOT to do: forwarding the user token

```text
Your API ──user token (aud=your-api)──► Service B  ← wrong
```

Problems:
- `aud` mismatch — Service B should reject it
- Least-privilege violation — Service B receives the user's full credential
- If Service B is compromised, the attacker has the user's token

### Which pattern to choose

```text
Token exchange (RFC 8693)
  + cryptographically proper — each service gets its own scoped token
  + user identity preserved via act claim
  + works across trust boundaries and external services
  - requires IdP support for token exchange grant
  - requires token exchanges or safe reuse of cached exchanged tokens

Client credentials + user identity forwarding
  + simpler — one token per service, user id passed in request
  + no IdP support needed beyond standard client credentials
  - relies on authenticated workload identity and integrity-protected context
  - only appropriate for internal services you control
```

For a controlled internal architecture, workload authentication plus
integrity-protected user context can be pragmatic. For calls that cross
organizational or trust boundaries, prefer a token issued for the downstream
audience, such as token exchange when your authorization server supports it.

### ReBAC across services

Both patterns give Service B what it needs to run a ReBAC check:

```text
Token exchange:
  Service B validates the delegated identity → maps (iss, sub) → user:01JABC...
  Check(user:01JABC..., can_read, invoice:INV-001)

Client credentials + forwarding:
  Service B reads authenticated internal user context → "user:01JABC..."
  Check(user:01JABC..., can_read, invoice:INV-001)
```

The ReBAC check is identical. The difference is only in how Service B learns
which user to check.

## From identity to ReBAC

> Core path resumes here.

Authentication gives you the user id. ReBAC uses it:

```text
HTTP request arrives
       │
       ▼
Auth middleware
  verify token
  extract: iss="https://auth.example.com/", sub="github|12345"
  map pair to: "user:01JABC..."
       │
       ▼
ReBAC check
  Check(user:01JABC..., can_edit, document:roadmapDocument)
       │
       ▼
allow or deny → handler runs business logic
```

Your document domain should receive an already-verified actor id. It should not
parse JWTs or call the IdP.

```text
Clean:  auth middleware -> "user:01JABC..." -> documents.update()
Messy:  document domain parses Authorization header, calls OpenFGA directly
```

Keep authentication, authorization, and business logic as three separate layers.

## Scopes vs. ReBAC

OAuth scopes are coarse-grained permissions granted to a client application:

```text
scope: documents.read   →  may this client call the read API?
scope: documents.write  →  may this client call the write API?
```

These are not per-object decisions. They just say whether the client application
is allowed to call a category of API at all.

ReBAC is fine-grained and object-specific:

```text
can user:alice edit document:roadmapDocument?  ←  specific object, specific user
```

You usually need both:

```text
Access token has documents.write scope?    yes, client is authorized to call the API
       ↓
ReBAC: can alice edit this document?       yes, this specific object is allowed
       ↓
allow action
```

OAuth scopes are not a replacement for object-level authorization. That is
exactly the gap ReBAC fills.

## Which flow should I use?

```text
App type              Flow                                Notes
────────────────────  ──────────────────────────────────  ──────────────────────
Server web app        Flow 1: Auth Code + PKCE + session  Store tokens server-side
SPA (browser)         Flow 1: Auth Code + PKCE            Avoid implicit flow
Native/mobile app     Flow 1: Auth Code + PKCE            Use system browser + PKCE
Machine-to-machine    Flow 2: Client Credentials          No user, no browser
CLI / device          Flow 3: Device Authorization        Or Auth Code with localhost
```

For this repo:

- Future browser/server version: Flow 1 (Authorization Code + PKCE + OIDC)
- Current terminal client: Flow 3 (Device Authorization) or Flow 1 with localhost callback
- Tutorial mode (current): you paste a demo bearer token (`demo-token-alice`,
  `demo-token-bob`, or `demo-token-casey`) instead of a real OAuth login, to keep
  the focus on authorization

## Refresh tokens and browser sessions

Refresh tokens are credentials used only with the authorization server's token
endpoint. Never send one to a resource API. Store refresh tokens as carefully as
passwords and avoid exposing them to browser JavaScript where the architecture
allows tokens to remain in a backend.

For public clients, use refresh-token rotation with reuse detection or
sender-constrained refresh tokens when supported. Apply absolute and inactivity
expiry appropriate to the threat model, revoke tokens when compromise is
suspected, and remember that ending an app session does not automatically end
the IdP session or revoke every token.

For a server web app, the browser should normally receive an application session
cookie rather than OAuth tokens. Protect that cookie with `Secure`, `HttpOnly`,
and an appropriate `SameSite` setting; rotate the session identifier after
login; and add CSRF protection to authenticated state-changing requests.

## Two patterns to avoid

**Implicit flow** — an old SPA approach that returned tokens in the URL fragment.
Authorization Code + PKCE replaced it. Do not use it.

**Resource Owner Password Credentials** — your app collects the user's password
directly. This breaks the entire point of delegated login. Do not use it.

If a tutorial still recommends either of these, treat it as outdated.

## JWTs in thirty seconds

OIDC ID tokens are JWTs. Access tokens may be JWTs or opaque values, depending
on the authorization server. A JWT has three parts:

```text
header.payload.signature
```

The payload contains claims. The signature proves the IdP issued it. You must
verify the signature before trusting any claims.

```text
Never trust a JWT just because it decodes. Always verify the signature.
```

## Common mistakes

**1. Treating login as permission**

```text
Bad:    if user is logged in → allow edit
Better: if user is logged in → check can_edit for this specific document
```

**2. Putting permissions in the token**

```text
Bad:    JWT contains all of Alice's teams and roles
Problem: Alice leaves a team; her old token still claims she is in it
Better: JWT contains Alice's stable identity; authorization checks happen live
```

**3. Using email as the user id**

```text
Bad:    user:alice@example.com   (emails change)
Better: map (trusted issuer, subject) to an internal immutable user id
```

**4. Treating OAuth scopes as object permissions**

```text
Bad:    documents.write scope → can edit every document
Better: documents.write scope → may call the write API
        then ReBAC decides which documents alice can actually edit
```

**5. Learning from outdated examples**

If a tutorial recommends implicit flow for SPAs or password grant for
first-party apps, it is outdated. Stop reading it.

## Standards landscape (as of 2026)

```text
OAuth 2.0            base authorization framework
OpenID Connect 1.0   identity layer on top of OAuth 2.0
RFC 9700             OAuth 2.0 Security Best Current Practice
OAuth 2.1            active draft — consolidates modern guidance, not yet final
```

This course teaches the modern posture:

- Authorization Code flow with PKCE
- OpenID Connect for authentication
- Exact redirect URI matching
- Refresh token rotation or sender-constraining for public clients
- No implicit flow for SPAs
- No Resource Owner Password Credentials grant

## Checkpoint

> What do OIDC and access-token validation give ReBAC?

OIDC establishes the login identity, and the resource server validates the
access token used for the API request. ReBAC then uses the resulting stable
subject identity to decide what it may do on a specific object.

Two separate questions:
- OIDC/token validation establishes: **who is calling, and may this token call this API?**
- ReBAC answers: **what may they do with this?**

## Further reading

- [RFC 9700: OAuth 2.0 Security Best Current Practice](https://www.rfc-editor.org/rfc/rfc9700)
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0-final.html)
- [RFC 9068: JWT Profile for OAuth 2.0 Access Tokens](https://www.rfc-editor.org/rfc/rfc9068)
- [RFC 7662: OAuth 2.0 Token Introspection](https://www.rfc-editor.org/rfc/rfc7662)
- [RFC 8693: OAuth 2.0 Token Exchange](https://www.rfc-editor.org/rfc/rfc8693)
- [OAuth 2.1 Internet-Draft](https://datatracker.ietf.org/doc/draft-ietf-oauth-v2-1/)
- [RFC 9449: DPoP](https://www.rfc-editor.org/rfc/rfc9449) — sender-constrained tokens
