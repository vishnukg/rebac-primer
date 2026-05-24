// Integration tests for the Documents service HTTP API.
//
// The AuthzClient port is satisfied by makeInProcessAuthzClient — a test stub
// that runs the real graph evaluator in-process.  No HTTP calls to the authz
// service are made, but real authz logic runs on every request.
//
// Coverage: authn (401) → domain (403/404) → success (200/201)
import { describe, expect, it } from "vitest";
import makeDocumentsHttpHandler from
    "../src/documents-service/adapters/http/makeDocumentsHttpHandler.ts";
import makeInMemoryDocumentRepository from
    "../src/documents-service/adapters/db/makeInMemoryDocumentRepository.ts";
import makeDemoTokenVerifier from
    "../src/documents-service/adapters/authn/makeDemoTokenVerifier.ts";
import makeDocuments from "../src/documents-service/core/domain/makeDocuments.ts";
import {
    demoTokens, seedPolicyTuples,
    makeInProcessAuthzClient,
} from "./fixtures.ts";

type Handler = ReturnType<typeof makeDocumentsHttpHandler>;

const q = new URLSearchParams();

// Builds a fresh handler (and fresh in-memory stores) for each test.
const makeHandler = (): Handler => {
    const authzClient   = makeInProcessAuthzClient(seedPolicyTuples());
    const authenticator = makeDemoTokenVerifier({ tokens: demoTokens });
    const repository    = makeInMemoryDocumentRepository();
    const documents     = makeDocuments({ repository, authzClient });
    return makeDocumentsHttpHandler({ authenticator, documents });
};

// Builds a handler that already has a seeded document (alice as owner).
// Used by read/update tests that need a document to exist first.
const makeHandlerWithDocument = async (): Promise<Handler> => {
    const handler = makeHandler();
    await handler({
        method: "POST", path: "/documents", query: q,
        authorization: "Bearer demo-token-alice",
        body: { id: "roadmapDocument", title: "Roadmap", body: "v1", workspaceId: "productWorkspace" },
    });
    return handler;
};

// ── Health ────────────────────────────────────────────────────────────────────

describe("Documents service — GET /health", () => {
    it("returns 200 without a token", async () => {
        // Arrange
        const handler = makeHandler();

        // Act
        const response = await handler({ method: "GET", path: "/health", query: q, authorization: undefined });

        // Assert
        expect(response.statusCode).toBe(200);
    });
});

// ── Authentication ─────────────────────────────────────────────────────────────
// /whoami shows the authn flow: bearer token → verified identity.
// In production the token would be a signed JWT; here it is a demo lookup.

describe("Documents service — GET /whoami", () => {
    it("returns the verified user identity for a valid token", async () => {
        // Arrange
        const handler = makeHandler();

        // Act
        const response = await handler({
            method: "GET", path: "/whoami", query: q,
            authorization: "Bearer demo-token-alice",
        });

        // Assert
        expect(response.statusCode).toBe(200);
        expect(response.body.user).toBe("user:alice");
    });

    it("returns 401 when the Authorization header is absent", async () => {
        // Arrange
        const handler = makeHandler();

        // Act
        const response = await handler({ method: "GET", path: "/whoami", query: q, authorization: undefined });

        // Assert
        expect(response.statusCode).toBe(401);
    });
});

// ── Create ─────────────────────────────────────────────────────────────────────

describe("Documents service — POST /documents", () => {
    it("creates a document and returns 201 when the actor is a workspace editor", async () => {
        // Arrange
        const handler = makeHandler();

        // Act
        const response = await handler({
            method: "POST", path: "/documents", query: q,
            authorization: "Bearer demo-token-alice",
            body: { id: "newDoc", title: "New Doc", body: "Hello", workspaceId: "productWorkspace" },
        });

        // Assert
        expect(response.statusCode).toBe(201);
        expect((response.body.document as Record<string, unknown>).id).toBe("newDoc");
    });

    it("returns 403 when the actor is only a workspace viewer", async () => {
        // Arrange: bob is a viewer, not an editor.
        const handler = makeHandler();

        // Act
        const response = await handler({
            method: "POST", path: "/documents", query: q,
            authorization: "Bearer demo-token-bob",
            body: { id: "newDoc", title: "New", body: "Hi", workspaceId: "productWorkspace" },
        });

        // Assert
        expect(response.statusCode).toBe(403);
    });

    it("returns 401 when no token is provided", async () => {
        // Arrange
        const handler = makeHandler();

        // Act
        const response = await handler({
            method: "POST", path: "/documents", query: q,
            authorization: undefined,
            body: { id: "newDoc", title: "New", body: "Hi", workspaceId: "productWorkspace" },
        });

        // Assert
        expect(response.statusCode).toBe(401);
    });
});

// ── Read ──────────────────────────────────────────────────────────────────────

describe("Documents service — GET /documents/:id", () => {
    it("allows a workspace viewer to read (permission inherited via workspace)", async () => {
        // Arrange: document created by alice (writes workspace + owner tuples).
        const handler = await makeHandlerWithDocument();

        // Act
        const response = await handler({
            method: "GET", path: "/documents/roadmapDocument", query: q,
            authorization: "Bearer demo-token-bob",
        });

        // Assert
        expect(response.statusCode).toBe(200);
        expect((response.body.document as Record<string, unknown>).id).toBe("roadmapDocument");
    });

    it("returns 403 when the user has no path in the relationship graph", async () => {
        // Arrange
        const handler = await makeHandlerWithDocument();

        // Act
        const response = await handler({
            method: "GET", path: "/documents/roadmapDocument", query: q,
            authorization: "Bearer demo-token-casey",
        });

        // Assert
        expect(response.statusCode).toBe(403);
    });

    it("returns 401 when no token is provided", async () => {
        // Arrange
        const handler = await makeHandlerWithDocument();

        // Act
        const response = await handler({
            method: "GET", path: "/documents/roadmapDocument", query: q,
            authorization: undefined,
        });

        // Assert
        expect(response.statusCode).toBe(401);
    });

    it("returns 404 for a document that does not exist", async () => {
        // Arrange
        const handler = makeHandler();

        // Act
        const response = await handler({
            method: "GET", path: "/documents/doesNotExist", query: q,
            authorization: "Bearer demo-token-alice",
        });

        // Assert
        expect(response.statusCode).toBe(404);
    });
});

// ── Update ────────────────────────────────────────────────────────────────────

describe("Documents service — PATCH /documents/:id", () => {
    it("allows the document owner to update the body", async () => {
        // Arrange
        const handler = await makeHandlerWithDocument();

        // Act
        const response = await handler({
            method: "PATCH", path: "/documents/roadmapDocument", query: q,
            authorization: "Bearer demo-token-alice",
            body: { body: "v2" },
        });

        // Assert
        expect(response.statusCode).toBe(200);
        expect((response.body.document as Record<string, unknown>).body).toBe("v2");
    });

    it("returns 403 when a viewer tries to edit", async () => {
        // Arrange: bob is workspace viewer, has can_read but not can_edit.
        const handler = await makeHandlerWithDocument();

        // Act
        const response = await handler({
            method: "PATCH", path: "/documents/roadmapDocument", query: q,
            authorization: "Bearer demo-token-bob",
            body: { body: "hacked" },
        });

        // Assert
        expect(response.statusCode).toBe(403);
    });
});
