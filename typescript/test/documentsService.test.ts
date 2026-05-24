// Integration tests for the Documents service HTTP API.
//
// The authz client is a fake in-memory implementation — no HTTP calls.
// This keeps tests fast and deterministic while still exercising the full
// authn → domain → authz-client path.
import { describe, expect, it, beforeEach } from "vitest";
import makeDocumentsHttpHandler from
    "../src/documents-service/adapters/http/makeDocumentsHttpHandler.ts";
import makeInMemoryDocumentRepository from
    "../src/documents-service/adapters/db/makeInMemoryDocumentRepository.ts";
import makeDemoTokenVerifier from
    "../src/documents-service/adapters/authn/makeDemoTokenVerifier.ts";
import makeDocuments from "../src/documents-service/core/domain/makeDocuments.ts";
import type { AuthzClient } from "../src/documents-service/core/ports/authzClient.ts";
import { demoTokens, seedPolicyTuples, productWorkspace, alice, bob, casey } from
    "../src/demo/fixtures.ts";
import makeInMemoryTupleRepository from
    "../src/authz-service/adapters/db/makeInMemoryTupleRepository.ts";
import makeGraphEvaluator from
    "../src/authz-service/adapters/graph/makeGraphEvaluator.ts";
import { document, tuple } from "../src/shared/rebac.ts";
import type { TupleKey } from "../src/shared/rebac.ts";

// ── Fake AuthzClient ──────────────────────────────────────────────────────────
//
// Uses the real graph evaluator in-process so tests exercise actual authz logic
// without making HTTP calls.  The tuple repository is shared so writes from
// createDocument are visible to subsequent checks.

const makeInProcessAuthzClient = (seed: TupleKey[] = []): AuthzClient => {
    const repository = makeInMemoryTupleRepository(seed);
    const evaluator  = makeGraphEvaluator({ repository });
    return {
        check:       async req  => evaluator.evaluate(req),
        writeTuples: async tpls => { for (const t of tpls) repository.write(t); },
    };
};

// ── Handler factory ───────────────────────────────────────────────────────────

const makeHandler = () => {
    const authzClient   = makeInProcessAuthzClient(seedPolicyTuples());
    const authenticator = makeDemoTokenVerifier({ tokens: demoTokens });
    const repository    = makeInMemoryDocumentRepository();
    const documents     = makeDocuments({ repository, authzClient });
    return makeDocumentsHttpHandler({ authenticator, documents });
};

const q = new URLSearchParams();

// ── Tests ─────────────────────────────────────────────────────────────────────

describe("Documents service — GET /health", () => {
    it("returns 200 without authentication", async () => {
        const h = makeHandler();
        const r = await h({ method: "GET", path: "/health", query: q, authorization: undefined });
        expect(r.statusCode).toBe(200);
    });
});

describe("Documents service — GET /whoami", () => {
    it("returns user identity for a valid token", async () => {
        const h = makeHandler();
        const r = await h({
            method: "GET", path: "/whoami", query: q,
            authorization: "Bearer demo-token-alice",
        });
        expect(r.statusCode).toBe(200);
        expect(r.body.user).toBe("user:alice");
    });

    it("returns 401 for a missing token", async () => {
        const h = makeHandler();
        const r = await h({ method: "GET", path: "/whoami", query: q, authorization: undefined });
        expect(r.statusCode).toBe(401);
    });
});

describe("Documents service — POST /documents (create)", () => {
    it("creates a document and writes tuples to authz client", async () => {
        const h = makeHandler();
        const r = await h({
            method: "POST", path: "/documents", query: q,
            authorization: "Bearer demo-token-alice",
            body: { id: "newDoc", title: "New Doc", body: "Hello", workspaceId: "productWorkspace" },
        });
        expect(r.statusCode).toBe(201);
        expect((r.body.document as Record<string, unknown>).id).toBe("newDoc");
    });

    it("returns 403 when the actor cannot create in that workspace", async () => {
        const h = makeHandler();
        const r = await h({
            method: "POST", path: "/documents", query: q,
            authorization: "Bearer demo-token-bob",  // bob is viewer, not editor
            body: { id: "newDoc", title: "New", body: "Hi", workspaceId: "productWorkspace" },
        });
        expect(r.statusCode).toBe(403);
    });

    it("returns 401 when no token is provided", async () => {
        const h = makeHandler();
        const r = await h({
            method: "POST", path: "/documents", query: q,
            authorization: undefined,
            body: { id: "newDoc", title: "New", body: "Hi", workspaceId: "productWorkspace" },
        });
        expect(r.statusCode).toBe(401);
    });
});

describe("Documents service — GET /documents/:id (read)", () => {
    const setup = async () => {
        const authzClient   = makeInProcessAuthzClient(seedPolicyTuples());
        const authenticator = makeDemoTokenVerifier({ tokens: demoTokens });
        const repository    = makeInMemoryDocumentRepository();
        const documents     = makeDocuments({ repository, authzClient });
        const handler       = makeDocumentsHttpHandler({ authenticator, documents });

        // Create document first — this writes workspace + owner tuples
        await handler({
            method: "POST", path: "/documents", query: q,
            authorization: "Bearer demo-token-alice",
            body: { id: "roadmapDocument", title: "Roadmap", body: "v1", workspaceId: "productWorkspace" },
        });

        return handler;
    };

    it("allows bob to read (workspace viewer)", async () => {
        const h = await setup();
        const r = await h({
            method: "GET", path: "/documents/roadmapDocument", query: q,
            authorization: "Bearer demo-token-bob",
        });
        expect(r.statusCode).toBe(200);
        expect((r.body.document as Record<string, unknown>).id).toBe("roadmapDocument");
    });

    it("returns 403 when casey has no path", async () => {
        const h = await setup();
        const r = await h({
            method: "GET", path: "/documents/roadmapDocument", query: q,
            authorization: "Bearer demo-token-casey",
        });
        expect(r.statusCode).toBe(403);
    });

    it("returns 401 when no token is provided", async () => {
        const h = await setup();
        const r = await h({
            method: "GET", path: "/documents/roadmapDocument", query: q,
            authorization: undefined,
        });
        expect(r.statusCode).toBe(401);
    });

    it("returns 404 for a non-existent document", async () => {
        const h = makeHandler();
        const r = await h({
            method: "GET", path: "/documents/doesNotExist", query: q,
            authorization: "Bearer demo-token-alice",
        });
        expect(r.statusCode).toBe(404);
    });
});

describe("Documents service — PATCH /documents/:id (update)", () => {
    const setup = async () => {
        const authzClient   = makeInProcessAuthzClient(seedPolicyTuples());
        const authenticator = makeDemoTokenVerifier({ tokens: demoTokens });
        const repository    = makeInMemoryDocumentRepository();
        const documents     = makeDocuments({ repository, authzClient });
        const handler       = makeDocumentsHttpHandler({ authenticator, documents });

        await handler({
            method: "POST", path: "/documents", query: q,
            authorization: "Bearer demo-token-alice",
            body: { id: "roadmapDocument", title: "Roadmap", body: "v1", workspaceId: "productWorkspace" },
        });

        return handler;
    };

    it("allows alice to edit (platform team editor)", async () => {
        const h = await setup();
        const r = await h({
            method: "PATCH", path: "/documents/roadmapDocument", query: q,
            authorization: "Bearer demo-token-alice",
            body: { body: "v2" },
        });
        expect(r.statusCode).toBe(200);
        expect((r.body.document as Record<string, unknown>).body).toBe("v2");
    });

    it("returns 403 when bob tries to edit (viewer only)", async () => {
        const h = await setup();
        const r = await h({
            method: "PATCH", path: "/documents/roadmapDocument", query: q,
            authorization: "Bearer demo-token-bob",
            body: { body: "hacked" },
        });
        expect(r.statusCode).toBe(403);
    });
});
