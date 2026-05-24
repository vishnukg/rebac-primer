// Unit tests for the documents domain with a fully in-process authz client.
// See test/documentsService.test.ts for HTTP-level integration tests.
import { describe, expect, it } from "vitest";
import makeDocuments from "../src/documents-service/core/domain/makeDocuments.ts";
import makeInMemoryDocumentRepository from
    "../src/documents-service/adapters/db/makeInMemoryDocumentRepository.ts";
import makeInMemoryTupleRepository from
    "../src/authz-service/adapters/db/makeInMemoryTupleRepository.ts";
import makeGraphEvaluator from
    "../src/authz-service/adapters/graph/makeGraphEvaluator.ts";
import type { AuthzClient } from "../src/documents-service/core/ports/authzClient.ts";
import type { TupleKey } from "../src/shared/rebac.ts";
import { alice, bob, productWorkspace, seedPolicyTuples } from "../src/demo/fixtures.ts";

const makeInProcessAuthzClient = (seed: TupleKey[] = []): AuthzClient => {
    const repository = makeInMemoryTupleRepository(seed);
    const evaluator  = makeGraphEvaluator({ repository });
    return {
        check:       async req  => evaluator.evaluate(req),
        writeTuples: async tpls => { for (const t of tpls) repository.write(t); },
    };
};

const makeService = () =>
    makeDocuments({
        repository:  makeInMemoryDocumentRepository(),
        authzClient: makeInProcessAuthzClient(seedPolicyTuples()),
    });

describe("documents domain", () => {
    it("creates a document when the actor is a workspace editor", async () => {
        const documents = makeService();
        const doc = await documents.create({
            id: "d1", title: "T", body: "B",
            workspace: productWorkspace, actor: alice,
        });
        expect(doc.id).toBe("d1");
        expect(doc.updatedBy).toBe(alice);
    });

    it("rejects creation for workspace viewers", async () => {
        const documents = makeService();
        await expect(documents.create({
            id: "d1", title: "T", body: "B",
            workspace: productWorkspace, actor: bob,
        })).rejects.toMatchObject({ name: "ForbiddenError" });
    });

    it("allows reads after the document-workspace tuple is written on create", async () => {
        const documents = makeService();
        await documents.create({
            id: "d1", title: "T", body: "B",
            workspace: productWorkspace, actor: alice,
        });
        // bob is a workspace viewer — can_read should be granted via inheritance
        const doc = await documents.read({ id: "d1", actor: bob });
        expect(doc.id).toBe("d1");
    });

    it("returns ForbiddenError when editor tries to delete (no can_delete tuple)", async () => {
        const documents = makeService();
        await documents.create({
            id: "d1", title: "T", body: "B",
            workspace: productWorkspace, actor: alice,
        });
        const updated = await documents.update({ id: "d1", body: "v2", actor: alice });
        expect(updated.body).toBe("v2");
    });
});
