// Demo seed data — the small cast of actors, objects, tokens, and policy tuples
// the example scenario is built around.
//
// This lives under src/ (not test/) because runtime entrypoints seed from it:
//   src/authz-service/index.ts      → seedPolicyTuples()
//   src/documents-service/index.ts  → demoTokens
//
// Tests also import it (re-exported via test/fixtures.ts), so the demo and the
// tests stay in sync. This mirrors the Go implementation's internal/fixtures
// package, which cmd/server/main.go and the Go tests both depend on.

import { subjectSet, team, tuple, user, workspace } from "../shared/rebac.ts";
import type { TupleKey } from "../shared/rebac.ts";

// ── Demo actors ───────────────────────────────────────────────────────────────

export const alice = user("alice");
export const bob   = user("bob");
export const casey = user("casey");

// ── Demo objects ──────────────────────────────────────────────────────────────

export const platformTeam     = team("platformTeam");
export const productWorkspace = workspace("productWorkspace");

// ── Demo bearer tokens ────────────────────────────────────────────────────────

export const demoTokens: Record<string, { sub: string; scopes: string[] }> = {
    "demo-token-alice": { sub: "alice", scopes: ["documents:read", "documents:write"] },
    "demo-token-bob":   { sub: "bob",   scopes: ["documents:read"] },
    "demo-token-casey": { sub: "casey", scopes: ["documents:read"] },
};

// ── Policy tuples (workspace/team memberships) ────────────────────────────────
//
// These represent what the platform team writes to the authz service.
// Alice is a platform team member → editors of productWorkspace.
// Bob is a direct viewer of productWorkspace.

export const seedPolicyTuples = (): TupleKey[] => [
    tuple(platformTeam, "member", alice),
    tuple(productWorkspace, "editor", subjectSet(platformTeam, "member")),
    tuple(productWorkspace, "viewer", bob),
];
