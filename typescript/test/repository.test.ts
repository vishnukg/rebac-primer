import { describe, expect, it } from "vitest";
import makeInMemoryDocumentRepository from "../src/adapters/db/makeInMemoryDocumentRepository.ts";
import { alice, productWorkspace } from "../src/demo/fixtures.ts";

describe("makeInMemoryDocumentRepository", () => {
    it("stores snapshots instead of caller-owned objects", async () => {
        const repository = makeInMemoryDocumentRepository();
        const document = {
            id:        "roadmapDocument",
            title:     "Roadmap",
            body:      "v1",
            workspace: productWorkspace,
            updatedBy: alice,
        };

        await repository.save(document);
        document.body = "mutated outside repository";

        await expect(repository.findById("roadmapDocument")).resolves.toMatchObject({
            body: "v1",
        });
    });
});
