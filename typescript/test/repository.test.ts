import { describe, expect, it } from "vitest";
import { makeInMemoryDocumentRepository } from "../src/modules/db/index.ts";
import { productWorkspace, alice } from "../src/modules/fixtures/index.ts";

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

    await expect(repository.findById("roadmapDocument")).resolves.toMatchObject({ body: "v1" });
  });
});
