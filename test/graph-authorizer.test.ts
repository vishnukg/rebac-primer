import { describe, expect, it } from "vitest";
import { GraphAuthorizer } from "../src/authz/graph-authorizer.js";
import { MemoryTupleStore } from "../src/authz/memory-store.js";
import { tuple } from "../src/authz/types.js";
import {
  outsideCollaborator,
  platformTeam,
  roadmapDocument,
  seedRelationshipTuples,
  workspaceEditor,
  workspaceViewer
} from "../src/testing/fixtures.js";

describe("GraphAuthorizer", () => {
  it("given_team_member_workspace_editor_when_checking_document_edit_then_access_is_allowed", async () => {
    // Arrange
    const authorizer = new GraphAuthorizer(new MemoryTupleStore(seedRelationshipTuples()));

    // Act
    const result = await authorizer.check({
      user: workspaceEditor,
      relation: "can_edit",
      object: roadmapDocument
    });

    // Assert
    expect(result.allowed).toBe(true);
    expect(result.trace).toContain("Resolve subject set team:platformTeam#member: does it contain user:workspaceEditor?");
  });

  it("given_workspace_viewer_when_checking_document_permissions_then_read_is_allowed_and_edit_is_denied", async () => {
    // Arrange
    const authorizer = new GraphAuthorizer(new MemoryTupleStore(seedRelationshipTuples()));

    // Act
    const readResult = await authorizer.check({
      user: workspaceViewer,
      relation: "can_read",
      object: roadmapDocument
    });
    const editResult = await authorizer.check({
      user: workspaceViewer,
      relation: "can_edit",
      object: roadmapDocument
    });

    // Assert
    expect(readResult.allowed).toBe(true);
    expect(editResult.allowed).toBe(false);
  });

  it("given_actor_without_relationship_path_when_checking_read_then_access_is_denied", async () => {
    // Arrange
    const authorizer = new GraphAuthorizer(new MemoryTupleStore(seedRelationshipTuples()));

    // Act
    const result = await authorizer.check({
      user: outsideCollaborator,
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
      new MemoryTupleStore([
        ...seedRelationshipTuples(),
        tuple(platformTeam, "admin", outsideCollaborator)
      ])
    );

    // Act
    const result = await authorizer.check({
      user: outsideCollaborator,
      relation: "member",
      object: "team:platformTeam"
    });

    // Assert
    expect(result.allowed).toBe(true);
    expect(result.trace).toContain("team.member includes team.admin");
  });
});
