// Kept for backwards compatibility — full HTTP coverage now lives in
// test/authzService.test.ts and test/documentsService.test.ts.
// This file just confirms the authz service health endpoint is reachable.
import { describe, expect, it } from "vitest";
import makeAuthzDomain from "../src/authz-service/core/domain/makeAuthzDomain.ts";
import makeInMemoryTupleRepository from
    "../src/authz-service/adapters/db/makeInMemoryTupleRepository.ts";
import makeGraphEvaluator from
    "../src/authz-service/adapters/graph/makeGraphEvaluator.ts";
import makeAuthzHttpHandler from
    "../src/authz-service/adapters/http/makeAuthzHttpHandler.ts";

describe("authz service handler", () => {
    it("returns health", async () => {
        const repository = makeInMemoryTupleRepository();
        const evaluator  = makeGraphEvaluator({ repository });
        const domain     = makeAuthzDomain({ repository, evaluator });
        const handler    = makeAuthzHttpHandler(domain);

        const r = await handler({
            method: "GET", path: "/health",
            query: new URLSearchParams(),
        });
        expect(r.statusCode).toBe(200);
        expect(r.body).toEqual({ ok: true });
    });
});
