import { describe, expect, it } from "vitest";
import { openFgaModel, relationshipGraphExample } from "../src/authz/model.js";

describe("OpenFGA model", () => {
  it("given_openfga_model_when_reading_permissions_then_expected_hierarchy_is_present", () => {
    // Arrange
    const model = openFgaModel;

    // Act
    const hasRead = model.includes("define can_read: viewer");
    const hasEdit = model.includes("define can_edit: editor");
    const hasDelete = model.includes("define can_delete: owner");

    // Assert
    expect(hasRead).toBe(true);
    expect(hasEdit).toBe(true);
    expect(hasDelete).toBe(true);
  });

  it("given_openfga_model_when_reading_team_relations_then_admins_are_members", () => {
    // Arrange
    const model = openFgaModel;

    // Act
    const hasAdminRelation = model.includes("define admin: [user]");
    const hasAdminMembership = model.includes("define member: [user] or admin");

    // Assert
    expect(hasAdminRelation).toBe(true);
    expect(hasAdminMembership).toBe(true);
  });

  it("given_openfga_model_when_reading_document_relations_then_workspace_inheritance_is_present", () => {
    // Arrange
    const model = openFgaModel;

    // Act
    const hasWorkspaceRelation = model.includes("define workspace: [workspace]");
    const hasWorkspaceEditorInheritance = model.includes("workspace#editor from workspace");

    // Assert
    expect(hasWorkspaceRelation).toBe(true);
    expect(hasWorkspaceEditorInheritance).toBe(true);
  });

  it("given_model_documentation_when_reading_graph_example_then_plain_english_path_is_present", () => {
    // Arrange
    const example = relationshipGraphExample;

    // Act
    const hasWorkspaceEditorEditPath = example.includes(
      "therefore user:alice can_edit document:roadmapDocument"
    );

    // Assert
    expect(hasWorkspaceEditorEditPath).toBe(true);
  });
});
