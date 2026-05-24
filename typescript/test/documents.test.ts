import { describe, expect, it } from "vitest";
import {
  makeCreateDocument,
  makeDocuments,
  makeReadDocument,
  makeRequireAllowed,
  makeUpdateDocument,
  ForbiddenError,
} from "../src/modules/documents/index.ts";
import { makeGraphAuthorizer, makeInMemoryTupleStore } from "../src/modules/authz/index.ts";
import { makeInMemoryDocumentRepository } from "../src/modules/db/index.ts";
import { alice, bob, productWorkspace, seedRelationshipTuples } from "../src/modules/fixtures/index.ts";

const makeDocumentService = () => {
  const repository = makeInMemoryDocumentRepository();
  const tupleStore = makeInMemoryTupleStore({ seed: seedRelationshipTuples() });
  const authorizer = makeGraphAuthorizer({ tupleStore });
  const requireAllowed = makeRequireAllowed({ authorizer });
  const create = makeCreateDocument({ repository, requireAllowed });
  const read = makeReadDocument({ repository, requireAllowed });
  const update = makeUpdateDocument({ repository, requireAllowed });
  return makeDocuments({ create, read, update });
};

describe("documents module", () => {
  it("creates documents when the actor is a workspace editor", async () => {
    const documents = makeDocumentService();

    const created = await documents.create({
      id:        "strategy",
      title:     "Strategy",
      body:      "Ship carefully",
      workspace: productWorkspace,
      actor:     alice,
    });

    expect(created.updatedBy).toBe(alice);
  });

  it("rejects document creation for workspace viewers", async () => {
    const documents = makeDocumentService();

    await expect(documents.create({
      id:        "incident-plan",
      title:     "Incident Plan",
      body:      "Draft",
      workspace: productWorkspace,
      actor:     bob,
    })).rejects.toBeInstanceOf(ForbiddenError);
  });

  it("updates documents only when ReBAC allows can_edit", async () => {
    const documents = makeDocumentService();
    await documents.create({
      id:        "roadmapDocument",
      title:     "Roadmap",
      body:      "v1",
      workspace: productWorkspace,
      actor:     alice,
    });

    const updated = await documents.update({ id: "roadmapDocument", body: "v2", actor: alice });

    expect(updated.body).toBe("v2");
  });
});
