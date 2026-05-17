import { describe, expect, it } from "vitest";
import { workspace } from "../src/authz/types.js";
import { InMemoryDocumentRepository } from "../src/domain/repository.js";

describe("InMemoryDocumentRepository", () => {
  it("given_saved_document_when_listing_documents_then_document_is_returned", async () => {
    // Arrange
    const repository = new InMemoryDocumentRepository();

    // Act
    await repository.save({
      id: "roadmapDocument",
      title: "Roadmap",
      body: "v1",
      workspace: workspace("productWorkspace"),
      updatedBy: "user:alice"
    });
    const documents = await repository.list();

    // Assert
    expect(documents).toHaveLength(1);
  });

  it("given_saved_document_when_caller_mutates_original_then_repository_keeps_snapshot", async () => {
    // Arrange
    const repository = new InMemoryDocumentRepository();
    const document = {
      id: "roadmapDocument",
      title: "Roadmap",
      body: "v1",
      workspace: workspace("productWorkspace"),
      updatedBy: "user:alice" as const
    };

    // Act
    await repository.save(document);
    document.body = "mutated outside repository";
    const found = await repository.findById("roadmapDocument");

    // Assert
    expect(found?.body).toBe("v1");
  });
});
