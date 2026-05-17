import { describe, expect, it } from "vitest";
import { GraphAuthorizer } from "../src/authz/graph-authorizer.js";
import { InMemoryTupleStore } from "../src/authz/memory-store.js";
import { tuple } from "../src/authz/types.js";
import {
  casey,
  platformTeam,
  roadmapDocument,
  seedRelationshipTuples,
  alice,
  bob
} from "../src/testing/fixtures.js";

describe("GraphAuthorizer", () => {
  it("given_team_member_workspace_editor_when_checking_document_edit_then_access_is_allowed", async () => {
    // Arrange
    const authorizer = new GraphAuthorizer(new InMemoryTupleStore(seedRelationshipTuples()));

    // Act
    const result = await authorizer.check({
      user: alice,
      relation: "can_edit",
      object: roadmapDocument
    });

    // Assert
    expect(result.allowed).toBe(true);
    expect(result.trace).toContain("Resolve subject set team:platformTeam#member: does it contain user:alice?");
  });

  it("given_workspace_viewer_when_checking_document_permissions_then_read_is_allowed_and_edit_is_denied", async () => {
    // Arrange
    const authorizer = new GraphAuthorizer(new InMemoryTupleStore(seedRelationshipTuples()));

    // Act
    const readResult = await authorizer.check({
      user: bob,
      relation: "can_read",
      object: roadmapDocument
    });
    const editResult = await authorizer.check({
      user: bob,
      relation: "can_edit",
      object: roadmapDocument
    });

    // Assert
    expect(readResult.allowed).toBe(true);
    expect(editResult.allowed).toBe(false);
  });

  it("given_actor_without_relationship_path_when_checking_read_then_access_is_denied", async () => {
    // Arrange
    const authorizer = new GraphAuthorizer(new InMemoryTupleStore(seedRelationshipTuples()));

    // Act
    const result = await authorizer.check({
      user: casey,
      relation: "can_read",
      object: roadmapDocument
    });

    // Assert
    expect(result.allowed).toBe(false);
    expect(result.trace.at(-1)).toBe("Result: denied");
  });

  it("given_team_admin_when_checking_team_membership_then_access_is_allowed_by_model_hierarchy", async () => {
    // Arrange
    const authorizer = new GraphAuthorizer(
      new InMemoryTupleStore([
        ...seedRelationshipTuples(),
        tuple(platformTeam, "admin", casey)
      ])
    );

    // Act
    const result = await authorizer.check({
      user: casey,
      relation: "member",
      object: platformTeam
    });

    // Assert
    expect(result.allowed).toBe(true);
    expect(result.trace).toContain("team.member includes team.admin");
  });
});
