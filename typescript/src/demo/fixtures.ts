// Demo fixtures — shared by the server entrypoint and by tests.
//
// seedPolicyTuples: the workspace/team relationships a platform team would
// configure by calling POST /tuples on the authz service.  These represent
// "policy" — they rarely change.
//
// Document-level tuples (workspace relation, owner) are written by the
// documents service at document-creation time.

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
// These are what the platform team writes to the authz service.
// They express: Alice is a platform team member, platform team members are
// editors of productWorkspace, and Bob is a viewer of productWorkspace.

export const seedPolicyTuples = (): TupleKey[] => [
    tuple(platformTeam, "member", alice),
    tuple(productWorkspace, "editor", subjectSet(platformTeam, "member")),
    tuple(productWorkspace, "viewer", bob),
];
