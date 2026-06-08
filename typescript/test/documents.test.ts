// Unit tests for the documents domain.
// The AuthzClient port is satisfied by composeInProcessAuthzClient — a test stub
// that runs the real graph evaluator in-process instead of making HTTP calls.
// See test/documentsService.test.ts for HTTP-level integration tests.
import { describe, expect, it } from "vitest";
import makeDocuments from "../src/documents-service/core/domain/makeDocuments.ts";
import makeInMemoryDocumentRepository from
    "../src/documents-service/adapters/db/makeInMemoryDocumentRepository.ts";
import { document } from "../src/shared/rebac.ts";
import { alice, bob, productWorkspace, seedPolicyTuples, composeInProcessAuthzClient } from "./fixtures.ts";

const composeService = () =>
    makeDocuments({
        repository:  makeInMemoryDocumentRepository(),
        authzClient: composeInProcessAuthzClient(seedPolicyTuples()),
    });

describe("documents domain — create", () => {
    it("creates a document when the actor is a workspace editor", async () => {
        // Arrange
        const documents = composeService();

        // Act
        const doc = await documents.create({
            id: "d1", title: "Roadmap", body: "v1",
            workspace: productWorkspace, actor: alice,
        });

        // Assert
        expect(doc.id).toBe("d1");
        expect(doc.updatedBy).toBe(alice);
    });

    it("throws ForbiddenError when the actor is only a workspace viewer", async () => {
        // Arrange: bob is a viewer, not an editor.
        const documents = composeService();

        // Act + Assert
        await expect(documents.create({
            id: "d1", title: "Roadmap", body: "v1",
            workspace: productWorkspace, actor: bob,
        })).rejects.toMatchObject({ name: "ForbiddenError" });
    });

    it("makes the creator the document owner (grants can_delete)", async () => {
        // Arrange: share the authz stub so we can inspect the tuples create writes.
        const authzClient = composeInProcessAuthzClient(seedPolicyTuples());
        const documents = makeDocuments({
            repository: makeInMemoryDocumentRepository(),
            authzClient,
        });

        // Act: alice (a workspace editor) creates a document.
        await documents.create({
            id: "d1", title: "Roadmap", body: "v1",
            workspace: productWorkspace, actor: alice,
        });

        // Assert: alice can_delete d1. can_delete requires document owner, and a
        // workspace editor only inherits document editor (can_edit) — never owner.
        // So this passes only because create wrote a direct (d1, owner, alice) tuple.
        const aliceDelete = await authzClient.check({
            user: alice, relation: "can_delete", object: document("d1"),
        });
        expect(aliceDelete.allowed).toBe(true);

        // And bob (a workspace viewer) is not an owner — cannot delete.
        const bobDelete = await authzClient.check({
            user: bob, relation: "can_delete", object: document("d1"),
        });
        expect(bobDelete.allowed).toBe(false);
    });
});

describe("documents domain — read", () => {
    it("allows a workspace viewer to read after the creator writes ownership tuples", async () => {
        // Arrange: alice creates the document, which writes workspace + owner tuples
        //          to the shared authz stub.
        const documents = composeService();
        await documents.create({
            id: "d1", title: "Roadmap", body: "v1",
            workspace: productWorkspace, actor: alice,
        });

        // Act: bob is a workspace viewer — can_read should resolve via inheritance.
        const doc = await documents.read({ id: "d1", actor: bob });

        // Assert
        expect(doc.id).toBe("d1");
    });
});

describe("documents domain — update", () => {
    it("allows the document owner to update the body", async () => {
        // Arrange
        const documents = composeService();
        await documents.create({
            id: "d1", title: "Roadmap", body: "v1",
            workspace: productWorkspace, actor: alice,
        });

        // Act
        const updated = await documents.update({ id: "d1", body: "v2", actor: alice });

        // Assert
        expect(updated.body).toBe("v2");
        expect(updated.updatedBy).toBe(alice);
    });

    it("throws ForbiddenError when a viewer tries to edit", async () => {
        // Arrange
        const documents = composeService();
        await documents.create({
            id: "d1", title: "Roadmap", body: "v1",
            workspace: productWorkspace, actor: alice,
        });

        // Act + Assert
        await expect(documents.update({ id: "d1", body: "hacked", actor: bob }))
            .rejects.toMatchObject({ name: "ForbiddenError" });
    });
});
