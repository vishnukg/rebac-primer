import { describe, expect, it } from "vitest";
import { openFgaModel, relationshipGraphExample } from "../src/authz/model.js";

describe("OpenFGA model", () => {
  it("given_openfga_model_when_reading_permissions_then_expected_hierarchy_is_present", () => {
    // Arrange + Act + Assert
    expect(openFgaModel).toContain("define can_read: viewer");
    expect(openFgaModel).toContain("define can_edit: editor");
    expect(openFgaModel).toContain("define can_delete: owner");
  });

  it("given_openfga_model_when_reading_team_relations_then_admins_are_members", () => {
    // Arrange + Act + Assert
    expect(openFgaModel).toContain("define admin: [user]");
    expect(openFgaModel).toContain("define member: [user] or admin");
  });

  it("given_openfga_model_when_reading_document_relations_then_workspace_inheritance_is_present", () => {
    // Arrange + Act + Assert
    expect(openFgaModel).toContain("define workspace: [workspace]");
    expect(openFgaModel).toContain("workspace#editor from workspace");
  });

  it("given_model_documentation_when_reading_graph_example_then_plain_english_path_is_present", () => {
    // Arrange + Act + Assert
    expect(relationshipGraphExample).toContain("therefore user:alice can_edit document:roadmap");
  });
});
