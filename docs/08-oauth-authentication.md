# OAuth-based authentication

Authentication and authorization are related, but they are not the same thing.

```text
Authentication: who are you?
Authorization:  what are you allowed to do?
```

OAuth is mostly about delegated access and token-based flows. In modern web
apps, OAuth is often paired with OpenID Connect so the app can also learn who
the user is.

This repo does not implement OAuth yet, but you need the mental model because
real ReBAC systems almost always start with an authenticated user.

## Scene

Alice opens the document app. Before the app can ask:

```text
Can Alice edit document:roadmap?
```

it must first know:

```text
Is this request really from Alice?
```

That is authentication.

## Current standard landscape

As of 2026, the practical modern OAuth picture is:

```text
OAuth 2.0             base authorization framework
OpenID Connect 1.0   identity layer on top of OAuth 2.0
RFC 9700             OAuth 2.0 Security Best Current Practice
OAuth 2.1            active draft, not yet the final replacement RFC
```

The important teaching point:

```text
Use OAuth 2.0/OIDC with current security best practices.
Do not learn old OAuth 2.0 examples that still recommend implicit flow or
password grant for browser/mobile apps.
```

OAuth 2.1 is useful to study because it consolidates modern OAuth guidance, but
until it is final, the safest wording is "OAuth 2.0 plus current BCP guidance."

This course teaches that modern posture:

- Authorization Code flow
- PKCE
- exact redirect URI matching
- no implicit flow for SPAs
- no resource owner password credentials grant
- refresh token rotation or sender-constrained refresh tokens
- sender-constrained access tokens when the threat model requires it
- OpenID Connect for authentication

## The big picture

Typical OAuth/OIDC login flow:

```text
┌────────────┐       1. Login        ┌──────────────┐
│            │ ────────────────────► │              │
│  Browser   │                       │  Your App    │
│            │ ◄──────────────────── │              │
└────────────┘ 2. Redirect to IdP    └──────┬───────┘
                                            │
                                            │ 3. Authorization request
                                            ▼
                                     ┌──────────────┐
                                     │ Identity     │
                                     │ Provider     │
                                     │ GitHub/Auth0 │
                                     └──────┬───────┘
                                            │
                                            │ 4. Redirect back with code
                                            ▼
┌────────────┐                       ┌──────────────┐
│            │ 5. Callback + code    │              │
│  Browser   │ ────────────────────► │  Your App    │
│            │                       │              │
└────────────┘                       └──────┬───────┘
                                            │
                                            │ 6. Exchange code for tokens
                                            ▼
                                     ┌──────────────┐
                                     │ Identity     │
                                     │ Provider     │
                                     └──────────────┘
```

The app eventually gets tokens that prove the login happened.

## OAuth roles

OAuth uses a few standard roles:

```text
Resource Owner       the user, such as Alice
Client               your app
Authorization Server the identity provider issuing tokens
Resource Server      API protected by tokens
```

In a small app, "client" and "resource server" may both be your backend.

```text
Alice             -> resource owner
TS ReBAC app      -> client and resource server
GitHub/Auth0/etc. -> authorization server
```

## Authorization Code flow

For backend web apps, the common secure flow is Authorization Code with PKCE.

High-level sequence:

```text
Browser          App                 Identity Provider
   │              │                          │
   │ login        │                          │
   ├─────────────►│                          │
   │              │ redirect with challenge  │
   │◄─────────────┤                          │
   ├────────────────────────────────────────►│
   │              │                          │ user authenticates
   │◄────────────────────────────────────────┤ redirect with code
   ├─────────────►│ callback code            │
   │              ├─────────────────────────►│ exchange code + verifier
   │              │◄─────────────────────────┤ tokens
   │              │ create app session       │
   │◄─────────────┤                          │
```

PKCE protects the code exchange so a stolen authorization code is not enough by
itself.

## Which flow should I use?

Different application shapes need different OAuth patterns.

```text
┌────────────────────┬──────────────────────────────┬──────────────────────┐
│ App type            │ Recommended approach         │ Notes                │
├────────────────────┼──────────────────────────────┼──────────────────────┤
│ Server web app      │ Auth Code + PKCE + session   │ Store tokens server  │
│ SPA/browser app     │ Auth Code + PKCE             │ Avoid implicit flow  │
│ Native/mobile app   │ Auth Code + PKCE             │ Custom URI/app links │
│ Machine-to-machine  │ Client Credentials           │ No user present      │
│ CLI user login      │ Device Authorization or code │ Depends on UX        │
└────────────────────┴──────────────────────────────┴──────────────────────┘
```

For this repo's future browser/server version, the clean path is:

```text
Authorization Code + PKCE + OIDC
```

For the current terminal client, a real production CLI would usually use either:

```text
Device Authorization flow
```

or:

```text
Authorization Code + PKCE with localhost callback
```

This repo's current TUI does not implement login yet. It lets you type actor ids
so you can focus on ReBAC first.

## Deprecated or discouraged flows

You will still find old tutorials showing these:

```text
Implicit flow
Resource Owner Password Credentials grant
```

Treat them as historical context, not your default design.

### Implicit flow

Old SPA tutorials used implicit flow because browsers could not safely keep
client secrets.

Modern guidance prefers Authorization Code with PKCE for browser-based apps.

### Resource Owner Password Credentials

Password grant asks your app to collect the user's password directly.

That breaks the point of delegated login and should not be used for normal
modern applications.

## PKCE

PKCE stands for Proof Key for Code Exchange.

The client creates:

```text
code_verifier  -> secret random value
code_challenge -> transformed value sent in authorization request
```

Flow:

```text
Client                         Authorization Server
  │                                      │
  │ auth request + code_challenge        │
  ├─────────────────────────────────────►│
  │                                      │ user login
  │ authorization code                   │
  │◄─────────────────────────────────────┤
  │ token request + code_verifier        │
  ├─────────────────────────────────────►│
  │ verifies challenge matches verifier  │
  │ access/id tokens                     │
  │◄─────────────────────────────────────┤
```

If an attacker steals only the authorization code, they still do not have the
`code_verifier`, so the token exchange should fail.

## Tokens

You will hear about three token-ish things:

```text
Authorization code: short-lived value exchanged for tokens
Access token:       presented to APIs
ID token:           OIDC token describing the authenticated user
Refresh token:      long-lived token used to get new access tokens
```

Important distinction:

```text
Access token says: this client may call this API.
ID token says: this user authenticated with this identity provider.
```

OpenID Connect is the layer that standardizes ID tokens.

## Access token scopes vs app authorization

OAuth scopes are coarse permissions granted to a client.

Example:

```text
scope: documents.read
scope: documents.write
```

Scopes answer:

```text
May this client call this category of API?
```

ReBAC answers:

```text
May this user edit this exact document?
```

You usually need both:

```text
Access token has documents.write scope
AND
OpenFGA says user:alice can_edit document:roadmap
```

Diagram:

```text
┌──────────────┐
│ Access token │ has scope documents.write?
└──────┬───────┘
       │ yes
       ▼
┌──────────────┐
│ ReBAC check  │ can user edit this object?
└──────┬───────┘
       │ yes
       ▼
 allow action
```

OAuth scopes should not become a replacement for object-level authorization.

## JWTs

Tokens are often JWTs.

JWT shape:

```text
header.payload.signature
```

The payload may contain claims:

```json
{
  "sub": "github|12345",
  "iss": "https://auth.example.com/",
  "aud": "ts-rebac-primer",
  "exp": 1760000000
}
```

Common claims:

```text
sub  subject: stable user identifier
iss  issuer: who minted the token
aud  audience: who the token is for
exp  expiration
```

Never trust a JWT just because it decodes. Verification means checking the
signature and claims.

## Sessions vs bearer tokens

Two common backend patterns:

```text
Browser session:
  browser sends secure cookie
  server looks up session
  server knows user id

Bearer token:
  client sends Authorization: Bearer <token>
  server validates token
  server extracts user id
```

For server-rendered web apps, secure HTTP-only cookies are common. For APIs and
CLIs, bearer tokens are common.

## Sender-constrained tokens

A bearer token works like cash:

```text
whoever possesses it can use it
```

That is why token theft is serious.

Modern OAuth deployments may use sender-constrained tokens for stronger
security. Two important standards are:

```text
mTLS  binds token use to a client certificate
DPoP  binds token use to a public/private key proof at the application layer
```

Mental model:

```text
Bearer token:
  request has token -> accepted if token is valid

Sender-constrained token:
  request has token + proof of key/certificate -> accepted if both are valid
```

For a beginner project, learn bearer tokens first. For high-risk APIs, learn
sender-constrained tokens next.

## How authentication feeds ReBAC

Authentication gives you the user id.

Authorization uses that user id in a check.

```text
┌──────────────┐
│ HTTP Request │
│ Bearer JWT   │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Authenticate │ verify token, extract sub
└──────┬───────┘
       │ user:github:12345
       ▼
┌──────────────┐
│ Authorize    │ Check(user, relation, object)
└──────┬───────┘
       │ allowed / denied
       ▼
┌──────────────┐
│ Handler      │ perform business action
└──────────────┘
```

In this repo's tutorial data:

```text
OAuth subject -> user:alice
```

In a real app:

```text
OAuth subject github|12345 -> user:github:12345
```

That mapping should be stable. Do not use display names or emails as permanent
authorization ids.

## Multiple concrete cases

### Case 1: server-rendered app

```text
Browser -> App
App redirects to IdP
App receives code
App stores tokens server-side
Browser receives session cookie
App maps session to user id for ReBAC checks
```

Use when the backend owns the web session.

### Case 2: SPA plus API

```text
Browser SPA -> IdP with Authorization Code + PKCE
SPA receives tokens according to provider guidance
SPA calls API with access token
API validates token
API maps subject to ReBAC user id
API performs ReBAC check
```

Be careful with token storage in browsers. Avoid old implicit flow examples.

### Case 3: native/mobile app

```text
Mobile app -> system browser
IdP login -> app callback
App exchanges code with PKCE
App calls API with access token
API performs ReBAC check
```

Native apps are public clients. PKCE is essential.

### Case 4: machine-to-machine

```text
Service A -> token endpoint using client credentials
Service A -> API with access token
API checks service identity/scopes
```

There may be no human user. ReBAC can still model service principals:

```text
user:service-billing-worker
```

or a separate `service` type if your model needs it.

### Case 5: CLI

```text
CLI starts login
User authenticates in browser or enters device code
CLI receives token
CLI calls API
API validates token and performs ReBAC check
```

For this tutorial, the CLI is intentionally simpler and lets you type `alice`,
`bob`, or `chandra`. That keeps the first lesson focused on authorization.

## Where OAuth should live in the architecture

Authentication is an HTTP boundary concern.

```text
HTTP server
  -> authenticate request
  -> create Actor/User identity
  -> call domain service
  -> domain service authorizes action
```

Do not make `DocumentService` parse JWTs. It should receive an already
authenticated actor id.

Clean boundary:

```text
auth middleware extracts user:github:12345
DocumentService checks can_edit for user:github:12345
```

Messy boundary:

```text
DocumentService parses Authorization header and calls OpenFGA
```

Keep authentication, authorization, and business logic separate.

## OAuth does not replace ReBAC

OAuth can tell the app:

```text
this request is from user:alice
```

OAuth does not naturally answer:

```text
can user:alice edit document:roadmap because she is in team:platform?
```

That second question is the ReBAC question.

## Common mistakes

### Mistake 1: Treating login as permission

Bad:

```text
if user is logged in, allow edit
```

Better:

```text
if user is logged in, check can_edit on this document
```

### Mistake 2: Putting too much in tokens

You might be tempted to put every team and permission in a JWT.

That becomes stale quickly:

```text
Alice leaves team:platform
Alice still has old token saying she is in team:platform
```

Prefer stable identity in the token and fresh authorization checks for important
resource actions.

### Mistake 3: Using email as the ReBAC user id

Emails change. Provider subject ids are more stable.

Prefer:

```text
user:github:12345
```

over:

```text
user:alice@example.com
```

### Mistake 4: Treating OAuth scopes as object permissions

Bad:

```text
documents.write scope -> can edit every document
```

Better:

```text
documents.write scope -> may call write API
ReBAC check -> may edit this document
```

### Mistake 5: Learning from outdated flow diagrams

If a tutorial recommends implicit flow for SPAs or password grant for first-party
apps, treat it as outdated unless there is a very specific legacy reason.

## Exercise

Design the authentication boundary for this repo:

1. pretend a request has `Authorization: Bearer <jwt>`
2. verify the JWT in middleware
3. extract `sub`
4. convert it to `user:<provider-subject>`
5. pass that actor id to `DocumentService`

Do not change `DocumentService` to know about JWTs.

## Checkpoint

Answer this:

```text
What does OAuth give ReBAC?
```

Good answer: OAuth/OIDC gives the app a verified user identity. ReBAC then uses
that identity to decide what the user can do on a specific object.

## Further reading

Primary sources worth knowing:

- RFC 9700: OAuth 2.0 Security Best Current Practice
- OAuth 2.1 draft in the IETF OAuth Working Group
- OpenID Connect Core 1.0
- RFC 9449: DPoP
- RFC 8705: OAuth mTLS
