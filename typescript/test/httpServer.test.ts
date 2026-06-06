// Integration test for the AuthZ HTTP server, and a demo of `await using`
// (ES2025 explicit resource management).
//
// startTestServer returns an AsyncDisposable. `await using server = …` calls its
// [Symbol.asyncDispose] automatically when the block ends — even if an assertion
// throws — so the server is always closed without an afterEach hook.

import { describe, expect, it } from "vitest";
import type { AddressInfo } from "node:net";
import makeInMemoryTupleRepository from "../src/authz-service/adapters/db/makeInMemoryTupleRepository.ts";
import makeGraphEvaluator from "../src/authz-service/adapters/graph/makeGraphEvaluator.ts";
import makeAuthzDomain from "../src/authz-service/core/domain/makeAuthzDomain.ts";
import makeAuthzHttpHandler from "../src/authz-service/adapters/http/makeAuthzHttpHandler.ts";
import makeAuthzHttpServer from "../src/authz-service/adapters/http/makeAuthzHttpServer.ts";
import { alice, productWorkspace, seedPolicyTuples } from "../src/demo/fixtures.ts";

const startTestServer = async (): Promise<{ baseUrl: string } & AsyncDisposable> => {
    const repository = makeInMemoryTupleRepository({ seed: seedPolicyTuples() });
    const evaluator  = makeGraphEvaluator({ repository });
    const domain     = makeAuthzDomain({ repository, evaluator });
    const handler    = makeAuthzHttpHandler({ authz: domain });
    const server     = makeAuthzHttpServer({ handler });

    // Bind to an ephemeral port (0) and wait for "listening" via Promise.withResolvers.
    const listening = Promise.withResolvers<void>();
    server.listen(0, "127.0.0.1", () => listening.resolve());
    await listening.promise;

    const { port } = server.address() as AddressInfo;
    return {
        baseUrl: `http://127.0.0.1:${port}`,
        [Symbol.asyncDispose]: () =>
            new Promise<void>((resolve, reject) =>
                server.close(err => (err ? reject(err) : resolve())),
            ),
    };
};

describe("authz HTTP server (integration)", () => {
    it("answers a real /check over HTTP and auto-closes via `await using`", async () => {
        await using server = await startTestServer();

        const response = await fetch(`${server.baseUrl}/check`, {
            method:  "POST",
            headers: { "content-type": "application/json" },
            body:    JSON.stringify({
                user:     alice,
                relation: "editor",
                object:   productWorkspace,
            }),
        });

        expect(response.ok).toBe(true);
        const result = (await response.json()) as { allowed: boolean; trace: string[] };
        // alice is a platformTeam member, and platformTeam#member are editors of
        // productWorkspace → allowed via subject-set resolution.
        expect(result.allowed).toBe(true);
        expect(result.trace.length).toBeGreaterThan(0);
    });
});
