import { describe, expect, it } from "vitest";
import { openFgaModel, relationshipGraphExample } from "../src/authz/model.js";

describe("OpenFGA model", () => {
  it("documents the permission hierarchy used by the code examples", () => {
    expect(openFgaModel).toContain("define can_read: viewer");
    expect(openFgaModel).toContain("define can_edit: editor");
    expect(openFgaModel).toContain("define can_delete: owner");
  });

  it("models document access inherited from the parent workspace", () => {
    expect(openFgaModel).toContain("define workspace: [workspace]");
    expect(openFgaModel).toContain("workspace#editor from workspace");
  });

  it("keeps a plain-English graph path next to the DSL", () => {
    expect(relationshipGraphExample).toContain("therefore user:alice can_edit document:roadmap");
  });
});
