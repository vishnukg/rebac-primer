import { describe, expect, it } from "vitest";
import { GraphAuthorizer } from "../src/authz/graph-authorizer.js";
import { InMemoryTupleStore } from "../src/authz/memory-store.js";
import { document, tuple } from "../src/authz/types.js";
import { DocumentNotFoundError, ForbiddenError } from "../src/domain/document.js";
import { InMemoryDocumentRepository } from "../src/domain/repository.js";
import { DocumentService } from "../src/domain/service.js";
import {
  casey,
  productWorkspace,
  seedRelationshipTuples,
  alice,
  bob
} from "../src/testing/fixtures.js";

describe("DocumentService", () => {
  it("given_workspace_editor_when_creating_document_then_document_is_created", async () => {
    // Arrange
    const store = new InMemoryTupleStore(seedRelationshipTuples());
    const service = new DocumentService(
      new InMemoryDocumentRepository(),
      new GraphAuthorizer(store)
    );

    // Act
    const created = await service.create({
      id: "strategy",
      title: "Strategy",
      body: "Ship carefully.",
      workspace: productWorkspace,
      actor: alice
    });

    // Assert
    expect(created.updatedBy).toBe(alice);
  });

  it("given_workspace_viewer_when_creating_document_then_forbidden_error_is_thrown", async () => {
    // Arrange
    const store = new InMemoryTupleStore(seedRelationshipTuples());
    const service = new DocumentService(
      new InMemoryDocumentRepository(),
      new GraphAuthorizer(store)
    );

    // Act
    const createPromise = service.create({
      id: "incident-plan",
      title: "Incident Plan",
      body: "Draft",
      workspace: productWorkspace,
      actor: bob
    });

    // Assert
    await expect(createPromise).rejects.toBeInstanceOf(ForbiddenError);
  });

  it("given_document_owner_when_updating_document_then_content_is_saved", async () => {
    // Arrange
    const store = new InMemoryTupleStore([
      ...seedRelationshipTuples(),
      tuple(document("roadmapDocument"), "owner", casey)
    ]);
    const service = new DocumentService(
      new InMemoryDocumentRepository(),
      new GraphAuthorizer(store)
    );
    await service.create({
      id: "roadmapDocument",
      title: "Roadmap",
      body: "v1",
      workspace: productWorkspace,
      actor: alice
    });

    // Act
    const updated = await service.update({
      id: "roadmapDocument",
      body: "v2",
      actor: casey
    });

    // Assert
    expect(updated.body).toBe("v2");
    expect(updated.updatedBy).toBe(casey);
  });

  it("given_actor_without_read_path_when_reading_document_then_forbidden_error_is_thrown", async () => {
    // Arrange
    const store = new InMemoryTupleStore(seedRelationshipTuples());
    const service = new DocumentService(
      new InMemoryDocumentRepository(),
      new GraphAuthorizer(store)
    );
    await service.create({
      id: "private-plan",
      title: "Private Plan",
      body: "v1",
      workspace: productWorkspace,
      actor: alice
    });

    // Act
    const readPromise = service.read("private-plan", casey);

    // Assert
    await expect(readPromise).rejects.toBeInstanceOf(ForbiddenError);
  });

  it("given_missing_document_when_updating_then_not_found_error_is_thrown", async () => {
    // Arrange
    const store = new InMemoryTupleStore(seedRelationshipTuples());
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
