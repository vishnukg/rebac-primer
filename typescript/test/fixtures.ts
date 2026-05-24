// Shared test fixtures and stubs.
//
// fixtures:         demo actors, objects, tokens, and seed policy tuples
// makeInProcessAuthzClient: a test stub that satisfies the AuthzClient port
//                  using the real graph evaluator in-process — no HTTP calls,
//                  but real authz logic.  Shared across documents.test.ts and
//                  documentsService.test.ts.

import { subjectSet, team, tuple, user, workspace } from "../src/shared/rebac.ts";
import type { TupleKey } from "../src/shared/rebac.ts";
import makeInMemoryTupleRepository from "../src/authz-service/adapters/db/makeInMemoryTupleRepository.ts";
import makeGraphEvaluator from "../src/authz-service/adapters/graph/makeGraphEvaluator.ts";
import type { AuthzClient } from "../src/documents-service/core/ports/authzClient.ts";

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

// ── AuthzClient stub ──────────────────────────────────────────────────────────
//
// Satisfies the AuthzClient port using the real graph evaluator in-process.
// Eliminates HTTP round-trips in tests while keeping real authz logic.
// The shared repository means tuples written by domain.create() are immediately
// visible to subsequent domain.read() calls — same behaviour as the real service.

export const makeInProcessAuthzClient = (seed: TupleKey[] = []): AuthzClient => {
    const repository = makeInMemoryTupleRepository(seed);
    const evaluator  = makeGraphEvaluator({ repository });
    return {
        check:       req  => evaluator.evaluate(req),
        writeTuples: async tpls => { for (const t of tpls) repository.write(t); },
    };
};
