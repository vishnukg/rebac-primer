import { describe, expect, it } from "vitest";
import { workspace } from "../src/authz/types.js";
import { InMemoryDocumentRepository } from "../src/domain/repository.js";

describe("InMemoryDocumentRepository", () => {
  it("given_saved_document_when_listing_documents_then_document_is_returned", async () => {
    // Arrange
    const repository = new InMemoryDocumentRepository();

    // Act
    await repository.save({
      id: "roadmap",
      title: "Roadmap",
      body: "v1",
      workspace: workspace("acme"),
      updatedBy: "user:alice"
    });
    const documents = await repository.list();

    // Assert
    expect(documents).toHaveLength(1);
  });
});
