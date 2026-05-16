import { describe, expect, it } from "vitest";
import { GraphAuthorizer } from "../src/authz/graph-authorizer.js";
import { MemoryTupleStore } from "../src/authz/memory-store.js";
import { alice, bob, chandra, roadmap, tutorialTuples } from "../src/testing/fixtures.js";

describe("GraphAuthorizer", () => {
  it("allows a team member to edit a document through workspace inheritance", async () => {
    const authorizer = new GraphAuthorizer(new MemoryTupleStore(tutorialTuples()));

    const result = await authorizer.check({
      user: alice,
      relation: "can_edit",
      object: roadmap
    });

    expect(result.allowed).toBe(true);
    expect(result.trace).toContain("Resolve subject set team:platform#member: does it contain user:alice?");
  });

  it("allows workspace viewers to read but not edit documents", async () => {
    const authorizer = new GraphAuthorizer(new MemoryTupleStore(tutorialTuples()));

    await expect(
      authorizer.check({ user: bob, relation: "can_read", object: roadmap })
    ).resolves.toMatchObject({ allowed: true });

    await expect(
      authorizer.check({ user: bob, relation: "can_edit", object: roadmap })
    ).resolves.toMatchObject({ allowed: false });
  });

  it("denies access when no relationship path exists", async () => {
    const authorizer = new GraphAuthorizer(new MemoryTupleStore(tutorialTuples()));

    const result = await authorizer.check({
      user: chandra,
      relation: "can_read",
      object: roadmap
    });

    expect(result.allowed).toBe(false);
    expect(result.trace.at(-1)).toBe("Result: denied");
  });
});
