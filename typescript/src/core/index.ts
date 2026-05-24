// ── Core public API ───────────────────────────────────────────────────────────
//
// This barrel is the ONLY import path adapters and tests should use for core
// types, ports, and domain factories. Direct imports into sub-folders are
// reserved for files within the core layer itself.
//
// What lives in core/:
//   domain/   — pure business logic (no framework, no I/O, no SDK)
//   ports/    — interfaces the domain declares but does not implement
//
// What lives in adapters/:
//   authn/    — Authenticator implementations (demo token verifier, JWT, …)
//   authz/    — Authorizer implementations (graph traversal, OpenFGA SDK, …)
//   db/       — DocumentRepository implementations (in-memory, Postgres, …)
//   http/     — HTTP request/response translation
//   client/   — terminal and HTTP API client

export * from "./domain/documents/index.ts";
export * from "./ports/index.ts";
