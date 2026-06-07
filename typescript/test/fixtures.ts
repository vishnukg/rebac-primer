// Test fixtures.
//
// The demo data (actors, objects, tokens, seed policy tuples) lives in
// src/demo/fixtures.ts so runtime entrypoints can seed from it too. This file
// re-exports that data and adds one test-only helper:
//
//   composeInProcessAuthzClient — a stub that satisfies the AuthzClient port using
//   the real graph evaluator in-process (no HTTP), so tests exercise real authz
//   logic without a network hop. Shared across documents.test.ts and
//   documentsService.test.ts.

import type { TupleKey } from "../src/shared/rebac.ts";
import makeInMemoryTupleRepository from "../src/authz-service/adapters/db/makeInMemoryTupleRepository.ts";
import makeGraphEvaluator from "../src/authz-service/adapters/graph/makeGraphEvaluator.ts";
import type { AuthzClient } from "../src/documents-service/core/ports/authzClient.ts";

// Re-export the demo data so tests can keep importing it from "./fixtures.ts".
export {
    alice, bob, casey,
    platformTeam, productWorkspace,
    demoTokens, seedPolicyTuples,
} from "../src/demo/fixtures.ts";

// ── AuthzClient stub ──────────────────────────────────────────────────────────
//
// Satisfies the AuthzClient port using the real graph evaluator in-process.
// Eliminates HTTP round-trips in tests while keeping real authz logic.
// The shared repository means tuples written by domain.create() are immediately
// visible to subsequent domain.read() calls — same behaviour as the real service.

export const composeInProcessAuthzClient = (seed: TupleKey[] = []): AuthzClient => {
    const repository = makeInMemoryTupleRepository({ seed });
    const evaluator  = makeGraphEvaluator({ repository });
    return {
        check:       req  => evaluator.evaluate(req),
        writeTuples: async tpls => { for (const t of tpls) repository.write(t); },
    };
};
