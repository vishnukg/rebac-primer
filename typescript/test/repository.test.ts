import { describe, expect, it } from "vitest";
import makeInMemoryDocumentRepository from
    "../src/documents-service/adapters/db/makeInMemoryDocumentRepository.ts";
import { alice, productWorkspace } from "./fixtures.ts";

describe("makeInMemoryDocumentRepository", () => {
    it("stores snapshots instead of caller-owned objects", async () => {
        const repository = makeInMemoryDocumentRepository();
        const doc = {
            id:        "roadmapDocument",
            title:     "Roadmap",
            body:      "v1",
            workspace: productWorkspace,
            updatedBy: alice,
        };

        await repository.save(doc);
        doc.body = "mutated outside repository";

        await expect(repository.findById("roadmapDocument")).resolves.toMatchObject({
            body: "v1",
        });
    });
});
