# OAuth and OIDC: who is this person?

Before your app can ask "can Alice edit this document?" it must know the request
actually came from Alice. That is the login problem. OAuth and OIDC are the
modern standards for solving it — without your app ever handling Alice's password
directly.

## Two acronyms, one sentence each

**OAuth 2.0** — a framework for delegated access: a user authorizes your app to
call APIs on their behalf without giving your app their password.

**OpenID Connect (OIDC)** — an identity layer built on top of OAuth: it adds a
standard way for your app to learn *who* the user is, not just *that* they
authenticated.

In practice, almost every "log in with GitHub / Google / Auth0" button uses
both:
- OAuth handles the token plumbing
- OIDC adds the "who is this person?" answer

## Authentication vs. authorization

```text
Authentication  →  Who are you?
Authorization   →  What can you do?
```

OAuth/OIDC handles authentication. Your ReBAC system handles authorization.
They hand off to each other:

```text
OAuth/OIDC proves:   "This request is from user:alice"
                            │
                            ▼
ReBAC decides:       "Can user:alice edit document:roadmapDocument?"
                            │
                            ▼
                     allow or deny
```

Keep these two questions separate. OAuth does not decide document permissions.
It only proves identity.

## The four players

Any OAuth/OIDC flow involves four roles:

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

The Authorization Server is also called an **Identity Provider (IdP)** when it
does OIDC (which it almost always does today).

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
Browser           Your App                   IdP (GitHub/Auth0)
   │                  │                              │
   │  1. click login  │                              │
   │─────────────────►│                              │
   │                  │  2. redirect to IdP          │
   │◄─────────────────│                              │
   │                                                 │
   │  3. Alice logs in at IdP                        │
   │────────────────────────────────────────────────►│
   │                                                 │
   │  4. IdP redirects back with a short-lived code  │
   │◄────────────────────────────────────────────────┤
   │                  │                              │
   │  5. browser sends code to app                   │
   │─────────────────►│                              │
   │                  │  6. app exchanges code       │
   │                  │     for tokens (server call) │
   │                  │─────────────────────────────►│
   │                  │  7. tokens                   │
   │                  │◄─────────────────────────────┤
   │                  │                              │
   │  8. app session  │                              │
   │◄─────────────────│                              │
```

In plain English:

1. Alice clicks "log in" on your app
2. Your app redirects Alice's browser to the IdP
3. Alice authenticates at the IdP (password, MFA, etc.) — your app never sees this
4. IdP redirects back to your app with a short-lived **authorization code**
5. Alice's browser carries that code back to your app's callback URL
6. Your app exchanges the code for real tokens — this is a server-to-server call,
   never in the browser URL
7. Your app receives tokens
8. Your app creates a session for Alice

Your app never sees Alice's password. That is the entire point.

## What are these tokens?

Step 7 gives you up to three things:

```text
Authorization code  →  short-lived, one-time value (like a coat-check ticket)
                        your app exchanges this for real tokens — then it expires
                        (not a token itself, just a stepping stone)

Access token        →  proves "this client is authorized to call this API"
                        sent with every API request as a Bearer token
                        expires quickly (minutes to an hour)

ID token            →  OIDC's contribution — proves "this person authenticated"
                        contains the user's identity (name, stable ID, email)
                        your app reads this to learn who just logged in

Refresh token       →  optional long-lived token used to get new access tokens
                        so Alice doesn't have to log in again every hour
```

The critical distinction:

```text
ID token     →  tells YOUR APP who the user is
Access token →  tells THE API that the client is authorized to call it
```

Do not use the ID token to call APIs. Do not use the access token to identify
the user.

## OIDC: the "who" layer

Without OIDC, OAuth only answers: "is this client allowed to call this API?" It
does not tell you *who* the user is.

OIDC adds the ID token — a JWT containing standard **claims** about the user:

```json
{
  "sub": "github|12345",
  "name": "Alice",
  "email": "alice@example.com",
  "iss": "https://accounts.google.com",
  "aud": "your-app-client-id",
  "exp": 1760000000
}
```

Common claims:

```text
sub   subject — stable user identifier (this is the one you care about most)
iss   issuer  — who issued this token
aud   audience — who this token is intended for (your app)
exp   expiration timestamp
```

The `sub` claim is the stable identifier to use as Alice's identity in your
system. Use it — not the email — because emails change. Provider subject IDs do
not.

In your app:

```text
OAuth subject "github|12345"  →  app user id "user:github:12345"
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

**Use when:** a service needs to call another service and there is no user
involved — background jobs, billing workers, microservice calls.

Instead of redirecting a browser, the service authenticates directly with its
own credentials:

```text
Service A                        Authorization Server
   │                                      │
   │  client_id + client_secret           │
   ├─────────────────────────────────────►│
   │                                      │  verify credentials
   │  access token                        │
   │◄─────────────────────────────────────┤
   │                                      │
   │  call API with access token          │
   ├─────────────────────────────────────► Resource Server (API)
```

In plain English:

1. Service A sends its client ID and secret directly to the Authorization Server
2. Authorization Server verifies them and returns an access token
3. Service A uses that token to call the API

No browser. No user. No redirect. No ID token — there is no user to identify.

The access token proves "Service A is authorized to call this API." The API can
still make authorization decisions about what the service is allowed to do.

ReBAC can model service identities just like user identities:

```text
user:service-billing-worker  →  can_read on document:invoiceTemplate
```

The `user:` prefix does not mean it has to be a human. It is just a subject in
your authorization model.

## Flow 3: Device Authorization

**Use when:** the app cannot open a browser — CLIs, smart TVs, IoT devices,
game consoles.

The problem these devices share: they can display text but cannot redirect the
user's browser to the IdP. The Device Authorization flow solves this by
splitting authentication across two devices.

```text
CLI / Device             Authorization Server          User's browser
      │                          │                            │
      │  1. request device code  │                            │
      ├─────────────────────────►│                            │
      │  2. device_code          │                            │
      │     user_code: ABCD-1234 │                            │
      │     verification_url     │                            │
      │◄─────────────────────────┤                            │
      │                          │                            │
      │  3. display to user:     │                            │
      │  "Visit example.com/activate"                         │
      │  "Enter code: ABCD-1234" │                            │
      │                          │   4. user visits URL       │
      │                          │◄───────────────────────────┤
      │                          │   5. user enters code      │
      │                          │◄───────────────────────────┤
      │                          │   6. user authenticates    │
      │                          │◄───────────────────────────┤
      │                          │                            │
      │  7. CLI polls for token  │                            │
      ├─────────────────────────►│                            │
      │  8. access token         │                            │
      │     + ID token (OIDC)    │                            │
      │◄─────────────────────────┤                            │
```

In plain English:

1. The CLI asks the Authorization Server for a device code
2. The Authorization Server returns a short code (e.g. `ABCD-1234`) and a URL
3. The CLI shows the user: "Visit example.com/activate and enter ABCD-1234"
4. The user opens their phone or laptop browser and visits that URL
5. The user enters the code and logs in normally
6. Meanwhile, the CLI polls the Authorization Server every few seconds: "is the
   user done yet?"
7. Once the user finishes, the next poll returns real tokens
8. The CLI now has an access token and ID token — same as Authorization Code flow

The user authenticates on a device that *can* open a browser. The CLI receives
tokens without ever needing to redirect anything.

This flow is what GitHub CLI, AWS CLI, and similar tools use for `gh auth login`
or `aws sso login`.

## From identity to ReBAC

Authentication gives you the user id. ReBAC uses it:

```text
HTTP request arrives
       │
       ▼
Auth middleware
  verify token
  extract sub: "github|12345"
  map to:      "user:github:12345"
       │
       ▼
ReBAC check
  Check(user:github:12345, can_edit, document:roadmapDocument)
       │
       ▼
allow or deny → handler runs business logic
```

Your `DocumentService` should receive an already-verified actor id. It should
not parse JWTs or call the IdP.

```text
Clean:  auth middleware → "user:github:12345" → DocumentService.check()
Messy:  DocumentService parses Authorization header, calls OpenFGA
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
- Tutorial mode (current): you type `alice`, `bob`, or `casey` — login is skipped
  to keep focus on authorization

## Two patterns to avoid

**Implicit flow** — an old SPA approach that returned tokens in the URL fragment.
Authorization Code + PKCE replaced it. Do not use it.

**Resource Owner Password Credentials** — your app collects the user's password
directly. This breaks the entire point of delegated login. Do not use it.

If a tutorial still recommends either of these, treat it as outdated.

## JWTs in thirty seconds

Tokens are often JWTs (JSON Web Tokens). A JWT has three parts:

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
Better: user:github:12345        (stable, tied to provider subject)
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
- Refresh token rotation
- No implicit flow for SPAs
- No Resource Owner Password Credentials grant

## Checkpoint

> What does OAuth/OIDC give ReBAC?

OAuth/OIDC verifies who is making the request and gives your app a stable user
identity. ReBAC then uses that identity to decide what the user can do on a
specific object.

Two separate questions:
- OAuth/OIDC answers: **who?**
- ReBAC answers: **what may they do with this?**

## Further reading

- [RFC 9700: OAuth 2.0 Security Best Current Practice](https://www.rfc-editor.org/rfc/rfc9700)
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0-final.html)
- [OAuth 2.1 Internet-Draft](https://datatracker.ietf.org/doc/draft-ietf-oauth-v2-1/)
- [RFC 9449: DPoP](https://www.rfc-editor.org/rfc/rfc9449) — sender-constrained tokens
