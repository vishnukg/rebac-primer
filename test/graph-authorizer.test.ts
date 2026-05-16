import { describe, expect, it } from "vitest";
import { GraphAuthorizer } from "../src/authz/graph-authorizer.js";
import { MemoryTupleStore } from "../src/authz/memory-store.js";
import { alice, bob, chandra, roadmap, tutorialTuples } from "../src/testing/fixtures.js";

describe("GraphAuthorizer", () => {
  it("given_team_member_workspace_editor_when_checking_document_edit_then_access_is_allowed", async () => {
    // Arrange
    const authorizer = new GraphAuthorizer(new MemoryTupleStore(tutorialTuples()));

    // Act
    const result = await authorizer.check({
      user: alice,
      relation: "can_edit",
      object: roadmap
    });

    // Assert
    expect(result.allowed).toBe(true);
    expect(result.trace).toContain("Resolve subject set team:platform#member: does it contain user:alice?");
  });

  it("given_workspace_viewer_when_checking_document_permissions_then_read_is_allowed_and_edit_is_denied", async () => {
    // Arrange
    const authorizer = new GraphAuthorizer(new MemoryTupleStore(tutorialTuples()));

    // Act
    const readResult = await authorizer.check({ user: bob, relation: "can_read", object: roadmap });
    const editResult = await authorizer.check({ user: bob, relation: "can_edit", object: roadmap });

    // Assert
    expect(readResult.allowed).toBe(true);
    expect(editResult.allowed).toBe(false);
  });

  it("given_actor_without_relationship_path_when_checking_read_then_access_is_denied", async () => {
    // Arrange
    const authorizer = new GraphAuthorizer(new MemoryTupleStore(tutorialTuples()));

    // Act
    const result = await authorizer.check({
      user: chandra,
      relation: "can_read",
      object: roadmap
    });

    // Assert
    expect(result.allowed).toBe(false);
    expect(result.trace.at(-1)).toBe("Result: denied");
  });

  it("given_team_member_when_checking_team_admin_then_access_is_allowed_by_model_hierarchy", async () => {
    // Arrange
    const authorizer = new GraphAuthorizer(new MemoryTupleStore(tutorialTuples()));

    // Act
    const result = await authorizer.check({
      user: alice,
      relation: "admin",
      object: "team:platform"
    });

    // Assert
    expect(result.allowed).toBe(true);
    expect(result.trace).toContain("team.admin includes team.member");
  });
});
