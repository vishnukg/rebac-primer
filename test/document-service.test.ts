import { describe, expect, it } from "vitest";
import { GraphAuthorizer } from "../src/authz/graph-authorizer.js";
import { MemoryTupleStore } from "../src/authz/memory-store.js";
import { document, tuple } from "../src/authz/types.js";
import { ForbiddenError } from "../src/domain/document.js";
import { InMemoryDocumentRepository } from "../src/domain/repository.js";
import { DocumentService } from "../src/domain/service.js";
import { acme, alice, bob, chandra, roadmap, tutorialTuples } from "../src/testing/fixtures.js";

describe("DocumentService", () => {
  it("checks workspace editor permission before creating a document", async () => {
    const service = serviceWithTuples(tutorialTuples());

    const created = await service.create({
      id: "strategy",
      title: "Strategy",
      body: "Ship carefully.",
      workspace: acme,
      actor: alice
    });

    expect(created.updatedBy).toBe(alice);
  });

  it("rejects creates when the actor has no workspace editor path", async () => {
    const service = serviceWithTuples(tutorialTuples());

    await expect(
      service.create({
        id: "incident-plan",
        title: "Incident Plan",
        body: "Draft",
        workspace: acme,
        actor: bob
      })
    ).rejects.toBeInstanceOf(ForbiddenError);
  });

  it("checks document edit permission before updating content", async () => {
    const service = serviceWithTuples([
      ...tutorialTuples(),
      tuple(document("roadmap"), "owner", chandra)
    ]);
    await service.create({
      id: "roadmap",
      title: "Roadmap",
      body: "v1",
      workspace: acme,
      actor: alice
    });

    const updated = await service.update({
      id: "roadmap",
      body: "v2",
      actor: chandra
    });

    expect(updated.body).toBe("v2");
    expect(updated.updatedBy).toBe(chandra);
    expect(roadmap).toBe("document:roadmap");
  });

  it("rejects reads when the actor has no document read path", async () => {
    const service = serviceWithTuples(tutorialTuples());
    await service.create({
      id: "private-plan",
      title: "Private Plan",
      body: "v1",
      workspace: acme,
      actor: alice
    });

    await expect(service.read("private-plan", chandra)).rejects.toBeInstanceOf(ForbiddenError);
  });
});

function serviceWithTuples(seed: ConstructorParameters<typeof MemoryTupleStore>[0]): DocumentService {
  const store = new MemoryTupleStore(seed);
  return new DocumentService(new InMemoryDocumentRepository(), new GraphAuthorizer(store));
}
