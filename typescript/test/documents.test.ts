// Unit tests for the documents domain.
// The AuthzClient port is satisfied by makeInProcessAuthzClient — a test stub
// that runs the real graph evaluator in-process instead of making HTTP calls.
// See test/documentsService.test.ts for HTTP-level integration tests.
import { describe, expect, it } from "vitest";
import makeDocuments from "../src/documents-service/core/domain/makeDocuments.ts";
import makeInMemoryDocumentRepository from
    "../src/documents-service/adapters/db/makeInMemoryDocumentRepository.ts";
import { alice, bob, productWorkspace, seedPolicyTuples, makeInProcessAuthzClient } from "./fixtures.ts";

const makeService = () =>
    makeDocuments({
        repository:  makeInMemoryDocumentRepository(),
        authzClient: makeInProcessAuthzClient(seedPolicyTuples()),
    });

describe("documents domain — create", () => {
    it("creates a document when the actor is a workspace editor", async () => {
        // Arrange
        const documents = makeService();

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
        const documents = makeService();

        // Act + Assert
        await expect(documents.create({
            id: "d1", title: "Roadmap", body: "v1",
            workspace: productWorkspace, actor: bob,
        })).rejects.toMatchObject({ name: "ForbiddenError" });
    });
});

describe("documents domain — read", () => {
    it("allows a workspace viewer to read after the creator writes ownership tuples", async () => {
        // Arrange: alice creates the document, which writes workspace + owner tuples
        //          to the shared authz stub.
        const documents = makeService();
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
        const documents = makeService();
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
        const documents = makeService();
        await documents.create({
            id: "d1", title: "Roadmap", body: "v1",
            workspace: productWorkspace, actor: alice,
        });

        // Act + Assert
        await expect(documents.update({ id: "d1", body: "hacked", actor: bob }))
            .rejects.toMatchObject({ name: "ForbiddenError" });
    });
});
