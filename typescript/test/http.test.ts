import { describe, expect, it } from "vitest";
import { makeDemoTokenVerifier } from "../src/modules/authn/index.ts";
import { makeGraphAuthorizer, makeInMemoryTupleStore } from "../src/modules/authz/index.ts";
import { makeInMemoryDocumentRepository } from "../src/modules/db/index.ts";
import {
  makeCreateDocument,
  makeDocuments,
  makeReadDocument,
  makeRequireAllowed,
  makeUpdateDocument,
} from "../src/modules/documents/index.ts";
import { demoTokens, seedRelationshipTuples, seedRoadmapDocument } from "../src/modules/fixtures/index.ts";
import { makeHttpHandler } from "../src/modules/http/index.ts";

const makeHandler = async () => {
  const repository = makeInMemoryDocumentRepository();
  const tupleStore = makeInMemoryTupleStore({ seed: seedRelationshipTuples() });
  const authorizer = makeGraphAuthorizer({ tupleStore });
  const requireAllowed = makeRequireAllowed({ authorizer });
  const documents = makeDocuments({
    create: makeCreateDocument({ repository, requireAllowed }),
    read:   makeReadDocument({ repository, requireAllowed }),
    update: makeUpdateDocument({ repository, requireAllowed }),
  });
  await documents.create(seedRoadmapDocument);
  return makeHttpHandler({
    authenticator: makeDemoTokenVerifier({ tokens: demoTokens }),
    documents,
  });
};

describe("makeHttpHandler", () => {
  it("returns health without authentication", async () => {
    const handler = await makeHandler();

    await expect(handler({
      method:        "GET",
      path:          "/health",
      query:         new URLSearchParams(),
      authorization: undefined,
    })).resolves.toEqual({ statusCode: 200, body: { ok: true } });
  });

  it("authenticates bearer tokens on whoami", async () => {
    const handler = await makeHandler();

    const response = await handler({
      method:        "GET",
      path:          "/whoami",
      query:         new URLSearchParams(),
      authorization: "Bearer demo-token-alice",
    });

    expect(response).toEqual({
      statusCode: 200,
      body:       { user: "user:alice", scopes: ["documents:read", "documents:write"] },
    });
  });

  it("reads a document when ReBAC allows the actor", async () => {
    const handler = await makeHandler();

    const response = await handler({
      method:        "GET",
      path:          "/documents/roadmapDocument",
      query:         new URLSearchParams({ actorId: "bob" }),
      authorization: undefined,
    });

    expect(response.statusCode).toBe(200);
    expect(response.body.document).toMatchObject({ id: "roadmapDocument" });
  });

  it("returns 403 when authentication succeeds but ReBAC denies authorization", async () => {
    const handler = await makeHandler();

    const response = await handler({
      method:        "PATCH",
      path:          "/documents/roadmapDocument",
      query:         new URLSearchParams(),
      authorization: "Bearer demo-token-bob",
      body:          { body: "not allowed" },
    });

    expect(response.statusCode).toBe(403);
    expect(response.body.error).toContain("cannot edit");
  });
});
