import { describe, expect, it } from "vitest";
import {
    makeGraphPermissionEvaluator,
    makeTupleStoreRelationshipReader,
} from "../src/adapters/authz/graphEvaluation.ts";
import makeInMemoryTupleStore from "../src/adapters/authz/makeInMemoryTupleStore.ts";
import { staticAuthorizationPolicy } from "../src/adapters/authz/graphPolicy.ts";
import makeDemoTokenVerifier from "../src/adapters/authn/makeDemoTokenVerifier.ts";
import makeInMemoryDocumentRepository from "../src/adapters/db/makeInMemoryDocumentRepository.ts";
import makeHttpHandler from "../src/adapters/http/makeHttpHandler.ts";
import { makeDocuments } from "../src/core/index.ts";
import type { Authorizer } from "../src/core/index.ts";
import {
    demoTokens,
    seedRelationshipTuples,
    seedRoadmapDocument,
} from "../src/demo/fixtures.ts";

const makeHandler = async () => {
    const repository = makeInMemoryDocumentRepository();
    const tupleStore = makeInMemoryTupleStore({ seed: seedRelationshipTuples() });
    const relationships = makeTupleStoreRelationshipReader(tupleStore);
    const evaluator = makeGraphPermissionEvaluator({
        relationships,
        policy: staticAuthorizationPolicy,
    });
    const authorizer: Authorizer = {
        check: async request => evaluator.check(request),
    };
    const documents = makeDocuments({ repository, authorizer });
    await documents.create(seedRoadmapDocument);
    return makeHttpHandler({
        authenticator: makeDemoTokenVerifier({ tokens: demoTokens }),
        documents,
    });
};

describe("makeHttpHandler", () => {
    it("returns health without authentication", async () => {
        const handler = await makeHandler();

        await expect(
            handler({
                method:        "GET",
                path:          "/health",
                query:         new URLSearchParams(),
                authorization: undefined,
            }),
        ).resolves.toEqual({ statusCode: 200, body: { ok: true } });
    });

    it("authenticates bearer tokens on whoami", async () => {
        const handler = await makeHandler();

        const response = await handler({
            method:        "GET",
            path:          "/whoami",
            query:         new URLSearchParams(),
            authorization: "Bearer demo-token-alice",
        });

        expect(response).toEqual({
            statusCode: 200,
            body:       { user: "user:alice", scopes: ["documents:read", "documents:write"] },
        });
    });

    it("reads a document when ReBAC allows the actor", async () => {
        const handler = await makeHandler();

        const response = await handler({
            method:        "GET",
            path:          "/documents/roadmapDocument",
            query:         new URLSearchParams({ actorId: "bob" }),
            authorization: undefined,
        });

        expect(response.statusCode).toBe(200);
        expect(response.body.document).toMatchObject({ id: "roadmapDocument" });
    });

    it("returns 403 when ReBAC denies the action", async () => {
        const handler = await makeHandler();

        const response = await handler({
            method:        "PATCH",
            path:          "/documents/roadmapDocument",
            query:         new URLSearchParams(),
            authorization: "Bearer demo-token-bob",
            body:          { body: "not allowed" },
        });

        expect(response.statusCode).toBe(403);
        expect(response.body.error).toContain("cannot edit");
    });
});
