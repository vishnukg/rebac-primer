import { describe, expect, it } from "vitest";
import makeGraphAuthorizer from "../src/adapters/authz/makeGraphAuthorizer.ts";
import makeInMemoryTupleStore from "../src/adapters/authz/makeInMemoryTupleStore.ts";
import makeInMemoryDocumentRepository from "../src/adapters/db/makeInMemoryDocumentRepository.ts";
import {
    ForbiddenError,
    makeDocuments,
} from "../src/core/index.ts";
import { alice, bob, productWorkspace, seedRelationshipTuples } from "../src/demo/fixtures.ts";

const makeDocumentService = () => {
    const repository = makeInMemoryDocumentRepository();
    const authorizer = makeGraphAuthorizer({
        tupleStore: makeInMemoryTupleStore({ seed: seedRelationshipTuples() }),
    });
    return makeDocuments({ repository, authorizer });
};

describe("documents", () => {
    it("creates a document when the actor is a workspace editor", async () => {
        const documents = makeDocumentService();

        const created = await documents.create({
            id:        "strategy",
            title:     "Strategy",
            body:      "Ship carefully",
            workspace: productWorkspace,
            actor:     alice,
        });

        expect(created.updatedBy).toBe(alice);
    });

    it("rejects creation for workspace viewers", async () => {
        const documents = makeDocumentService();

        await expect(
            documents.create({
                id:        "incident-plan",
                title:     "Incident Plan",
                body:      "Draft",
                workspace: productWorkspace,
                actor:     bob,
            }),
        ).rejects.toBeInstanceOf(ForbiddenError);
    });

    it("updates a document when ReBAC allows can_edit", async () => {
        const documents = makeDocumentService();
        await documents.create({
            id:        "roadmapDocument",
            title:     "Roadmap",
            body:      "v1",
            workspace: productWorkspace,
            actor:     alice,
        });

        const updated = await documents.update({
            id:    "roadmapDocument",
            body:  "v2",
            actor: alice,
        });

        expect(updated.body).toBe("v2");
    });
});
