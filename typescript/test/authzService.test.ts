// Integration tests for the AuthZ service HTTP API.
// Exercises POST /check, POST /tuples, DELETE /tuples, GET /tuples end-to-end
// through the handler — no HTTP server started, but full domain + adapter stack.
import { describe, expect, it } from "vitest";
import makeAuthzDomain from "../src/authz-service/core/domain/makeAuthzDomain.ts";
import makeInMemoryTupleRepository from "../src/authz-service/adapters/db/makeInMemoryTupleRepository.ts";
import makeGraphEvaluator from "../src/authz-service/adapters/graph/makeGraphEvaluator.ts";
import makeAuthzHttpHandler from "../src/authz-service/adapters/http/makeAuthzHttpHandler.ts";
import type { AuthzHttpHandler } from "../src/authz-service/adapters/http/makeAuthzHttpHandler.ts";
import { seedPolicyTuples, productWorkspace, alice, bob, casey } from "./fixtures.ts";
import { document, tuple } from "../src/shared/rebac.ts";

const roadmapDoc   = document("roadmapDocument");
const workspaceTpl = tuple(roadmapDoc, "workspace", productWorkspace);
const q            = new URLSearchParams();

// Builds a fully wired handler, optionally pre-seeded with extra tuples.
const composeHandler = (extra: ReturnType<typeof seedPolicyTuples> = []): AuthzHttpHandler => {
    const repository      = makeInMemoryTupleRepository({ seed: [...seedPolicyTuples(), ...extra] });
    const evaluator       = makeGraphEvaluator({ repository });
    const domain          = makeAuthzDomain({ repository, evaluator });
    return makeAuthzHttpHandler({ authz: domain });
};

describe("AuthZ service — GET /health", () => {
    it("returns 200 ok", async () => {
        // Arrange
        const handler = composeHandler();

        // Act
        const response = await handler({ method: "GET", path: "/health", query: q });

        // Assert
        expect(response.statusCode).toBe(200);
        expect(response.body).toEqual({ ok: true });
    });
});

describe("AuthZ service — POST /check", () => {
    it("returns allowed:true when the graph permits", async () => {
        // Arrange: workspace tuple links document to its parent workspace.
        const handler = composeHandler([workspaceTpl]);

        // Act
        const response = await handler({
            method: "POST", path: "/check", query: q,
            body: { user: alice, relation: "can_edit", object: roadmapDoc },
        });

        // Assert
        expect(response.statusCode).toBe(200);
        expect(response.body.allowed).toBe(true);
    });

    it("returns allowed:false when the graph denies", async () => {
        // Arrange: bob is workspace viewer, not editor.
        const handler = composeHandler([workspaceTpl]);

        // Act
        const response = await handler({
            method: "POST", path: "/check", query: q,
            body: { user: bob, relation: "can_edit", object: roadmapDoc },
        });

        // Assert
        expect(response.statusCode).toBe(200);
        expect(response.body.allowed).toBe(false);
    });

    it("includes a human-readable trace in the response", async () => {
        // Arrange
        const handler = composeHandler([workspaceTpl]);

        // Act
        const response = await handler({
            method: "POST", path: "/check", query: q,
            body: { user: casey, relation: "can_read", object: roadmapDoc },
        });

        // Assert
        expect(Array.isArray(response.body.trace)).toBe(true);
        expect((response.body.trace as string[]).at(-1)).toBe("Result: denied");
    });
});

describe("AuthZ service — POST /tuples then POST /check", () => {
    it("makes a permission available immediately after writing a tuple", async () => {
        // Arrange: start with no document-workspace link.
        const handler = composeHandler();

        // Act: write the workspace tuple (simulates what documents service does on create).
        const writeResponse = await handler({
            method: "POST", path: "/tuples", query: q,
            body: { tuples: [{ object: roadmapDoc, relation: "workspace", user: productWorkspace }] },
        });

        // Assert: tuple was accepted.
        expect(writeResponse.statusCode).toBe(200);
        expect(writeResponse.body.written).toBe(1);

        // Act: check the permission that depends on that tuple.
        const checkResponse = await handler({
            method: "POST", path: "/check", query: q,
            body: { user: alice, relation: "can_read", object: roadmapDoc },
        });

        // Assert: alice can now read via workspace inheritance.
        expect(checkResponse.body.allowed).toBe(true);
    });
});

describe("AuthZ service — DELETE /tuples", () => {
    it("revokes a permission after its tuple is deleted", async () => {
        // Arrange: bob can read because the workspace tuple exists.
        const handler = composeHandler([workspaceTpl]);
        const beforeResponse = await handler({
            method: "POST", path: "/check", query: q,
            body: { user: bob, relation: "can_read", object: roadmapDoc },
        });
        expect(beforeResponse.body.allowed).toBe(true);

        // Act: remove the workspace tuple (as if the document was moved or deleted).
        await handler({
            method: "DELETE", path: "/tuples", query: q,
            body: { tuples: [{ object: roadmapDoc, relation: "workspace", user: productWorkspace }] },
        });

        // Assert: the inheritance path no longer exists — bob is denied.
        const afterResponse = await handler({
            method: "POST", path: "/check", query: q,
            body: { user: bob, relation: "can_read", object: roadmapDoc },
        });
        expect(afterResponse.body.allowed).toBe(false);
    });
});
