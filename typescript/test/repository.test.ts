import { describe, expect, it } from "vitest";
import makeInMemoryDocumentRepository from
    "../src/documents-service/adapters/db/makeInMemoryDocumentRepository.ts";
import { alice, productWorkspace } from "./fixtures.ts";

const sampleDoc = () => ({
    id:        "roadmapDocument",
    title:     "Roadmap",
    body:      "v1",
    workspace: productWorkspace,
    updatedBy: alice,
});

describe("makeInMemoryDocumentRepository", () => {
    it("stores snapshots instead of caller-owned objects", async () => {
        // Arrange
        const repository = makeInMemoryDocumentRepository();
        const doc = sampleDoc();
        await repository.save(doc);

        // Act: mutate the caller's value after saving.
        doc.body = "mutated outside repository";

        // Assert: the store kept its own snapshot.
        await expect(repository.findById("roadmapDocument")).resolves.toMatchObject({
            body: "v1",
        });
    });

    it("returns undefined for a document that does not exist", async () => {
        // Arrange
        const repository = makeInMemoryDocumentRepository();

        // Act
        const found = await repository.findById("doesNotExist");

        // Assert
        expect(found).toBeUndefined();
    });

    it("returns the latest version when the same id is saved twice", async () => {
        // Arrange
        const repository = makeInMemoryDocumentRepository();
        await repository.save(sampleDoc());

        // Act
        await repository.save({ ...sampleDoc(), body: "v2" });

        // Assert
        await expect(repository.findById("roadmapDocument")).resolves.toMatchObject({
            body: "v2",
        });
    });

    it("returns a snapshot so mutating the result does not affect the store", async () => {
        // Arrange
        const repository = makeInMemoryDocumentRepository();
        await repository.save(sampleDoc());

        // Act: mutate the value handed back by findById.
        const first = await repository.findById("roadmapDocument");
        first!.body = "mutated via returned reference";

        // Assert: a fresh read is unaffected.
        await expect(repository.findById("roadmapDocument")).resolves.toMatchObject({
            body: "v1",
        });
    });
});
