import { describe, expect, it } from "vitest";
import { GraphAuthorizer } from "../src/authz/graph-authorizer.js";
import { MemoryTupleStore } from "../src/authz/memory-store.js";
import { document, tuple } from "../src/authz/types.js";
import { DocumentNotFoundError, ForbiddenError } from "../src/domain/document.js";
import { InMemoryDocumentRepository } from "../src/domain/repository.js";
import { DocumentService } from "../src/domain/service.js";
import { acme, alice, bob, chandra, roadmap, tutorialTuples } from "../src/testing/fixtures.js";

describe("DocumentService", () => {
  it("given_workspace_editor_when_creating_document_then_document_is_created", async () => {
    // Arrange
    const store = new MemoryTupleStore(tutorialTuples());
    const service = new DocumentService(
      new InMemoryDocumentRepository(),
      new GraphAuthorizer(store)
    );

    // Act
    const created = await service.create({
      id: "strategy",
      title: "Strategy",
      body: "Ship carefully.",
      workspace: acme,
      actor: alice
    });

    // Assert
    expect(created.updatedBy).toBe(alice);
  });

  it("given_workspace_viewer_when_creating_document_then_forbidden_error_is_thrown", async () => {
    // Arrange
    const store = new MemoryTupleStore(tutorialTuples());
    const service = new DocumentService(
      new InMemoryDocumentRepository(),
      new GraphAuthorizer(store)
    );

    // Act
    const createPromise = service.create({
      id: "incident-plan",
      title: "Incident Plan",
      body: "Draft",
      workspace: acme,
      actor: bob
    });

    // Assert
    await expect(createPromise).rejects.toBeInstanceOf(ForbiddenError);
  });

  it("given_document_owner_when_updating_document_then_content_is_saved", async () => {
    // Arrange
    const store = new MemoryTupleStore([
      ...tutorialTuples(),
      tuple(document("roadmap"), "owner", chandra)
    ]);
    const service = new DocumentService(
      new InMemoryDocumentRepository(),
      new GraphAuthorizer(store)
    );
    await service.create({
      id: "roadmap",
      title: "Roadmap",
      body: "v1",
      workspace: acme,
      actor: alice
    });

    // Act
    const updated = await service.update({
      id: "roadmap",
      body: "v2",
      actor: chandra
    });

    // Assert
    expect(updated.body).toBe("v2");
    expect(updated.updatedBy).toBe(chandra);
    expect(roadmap).toBe("document:roadmap");
  });

  it("given_actor_without_read_path_when_reading_document_then_forbidden_error_is_thrown", async () => {
    // Arrange
    const store = new MemoryTupleStore(tutorialTuples());
    const service = new DocumentService(
      new InMemoryDocumentRepository(),
      new GraphAuthorizer(store)
    );
    await service.create({
      id: "private-plan",
      title: "Private Plan",
      body: "v1",
      workspace: acme,
      actor: alice
    });

    // Act
    const readPromise = service.read("private-plan", chandra);

    // Assert
    await expect(readPromise).rejects.toBeInstanceOf(ForbiddenError);
  });

  it("given_missing_document_when_updating_then_not_found_error_is_thrown", async () => {
    // Arrange
    const store = new MemoryTupleStore(tutorialTuples());
    const service = new DocumentService(
      new InMemoryDocumentRepository(),
      new GraphAuthorizer(store)
    );

    // Act
    const updatePromise = service.update({
      id: "missing",
      body: "v2",
      actor: alice
    });

    // Assert
    await expect(updatePromise).rejects.toBeInstanceOf(DocumentNotFoundError);
  });
});
