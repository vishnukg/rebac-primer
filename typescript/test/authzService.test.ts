// Tests for the AuthZ service HTTP API.
// Exercises POST /check, POST /tuples, DELETE /tuples, GET /tuples.
import { describe, expect, it, beforeEach } from "vitest";
import makeAuthzDomain from "../src/authz-service/core/domain/makeAuthzDomain.ts";
import makeInMemoryTupleRepository from "../src/authz-service/adapters/db/makeInMemoryTupleRepository.ts";
import makeGraphEvaluator from "../src/authz-service/adapters/graph/makeGraphEvaluator.ts";
import makeAuthzHttpHandler from "../src/authz-service/adapters/http/makeAuthzHttpHandler.ts";
import type { AuthzHttpHandler } from "../src/authz-service/adapters/http/makeAuthzHttpHandler.ts";
import { seedPolicyTuples, productWorkspace, alice, bob, casey } from "../src/demo/fixtures.ts";
import { document, tuple } from "../src/shared/rebac.ts";

const roadmapDoc   = document("roadmapDocument");
const workspaceTpl = tuple(roadmapDoc, "workspace", productWorkspace);

const makeHandler = (extra = []): AuthzHttpHandler => {
    const repository = makeInMemoryTupleRepository([...seedPolicyTuples(), ...extra]);
    const evaluator  = makeGraphEvaluator({ repository });
    const domain     = makeAuthzDomain({ repository, evaluator });
    return makeAuthzHttpHandler(domain);
};

const q = new URLSearchParams();

describe("AuthZ service — GET /health", () => {
    it("returns 200 ok", async () => {
        const h = makeHandler();
        const r = await h({ method: "GET", path: "/health", query: q });
        expect(r.statusCode).toBe(200);
        expect(r.body).toEqual({ ok: true });
    });
});

describe("AuthZ service — POST /check", () => {
    it("returns allowed=true when the graph permits", async () => {
        const h = makeHandler([workspaceTpl]);
        const r = await h({
            method: "POST", path: "/check", query: q,
            body: { user: alice, relation: "can_edit", object: roadmapDoc },
        });
        expect(r.statusCode).toBe(200);
        expect(r.body.allowed).toBe(true);
    });

    it("returns allowed=false when the graph denies", async () => {
        const h = makeHandler([workspaceTpl]);
        const r = await h({
            method: "POST", path: "/check", query: q,
            body: { user: bob, relation: "can_edit", object: roadmapDoc },
        });
        expect(r.statusCode).toBe(200);
        expect(r.body.allowed).toBe(false);
    });

    it("includes a trace in the response", async () => {
        const h = makeHandler([workspaceTpl]);
        const r = await h({
            method: "POST", path: "/check", query: q,
            body: { user: casey, relation: "can_read", object: roadmapDoc },
        });
        expect(Array.isArray(r.body.trace)).toBe(true);
        expect((r.body.trace as string[]).at(-1)).toBe("Result: denied");
    });
});

describe("AuthZ service — POST /tuples and GET /tuples", () => {
    it("writes tuples and makes them queryable", async () => {
        const h = makeHandler();
        // Write a document-workspace tuple (as the documents service would do)
        const write = await h({
            method: "POST", path: "/tuples", query: q,
            body: { tuples: [{ object: roadmapDoc, relation: "workspace", user: productWorkspace }] },
        });
        expect(write.statusCode).toBe(200);
        expect(write.body.written).toBe(1);

        // Check that alice can now read (workspace inheritance should work)
        const check = await h({
            method: "POST", path: "/check", query: q,
            body: { user: alice, relation: "can_read", object: roadmapDoc },
        });
        expect(check.body.allowed).toBe(true);
    });
});

describe("AuthZ service — DELETE /tuples", () => {
    it("removes a tuple so the permission is revoked", async () => {
        const h = makeHandler([workspaceTpl]);
        // bob can currently read
        const before = await h({
            method: "POST", path: "/check", query: q,
            body: { user: bob, relation: "can_read", object: roadmapDoc },
        });
        expect(before.body.allowed).toBe(true);

        // delete the workspace tuple
        await h({
            method: "DELETE", path: "/tuples", query: q,
            body: { tuples: [{ object: roadmapDoc, relation: "workspace", user: productWorkspace }] },
        });

        // bob can no longer read
        const after = await h({
            method: "POST", path: "/check", query: q,
            body: { user: bob, relation: "can_read", object: roadmapDoc },
        });
        expect(after.body.allowed).toBe(false);
    });
});
