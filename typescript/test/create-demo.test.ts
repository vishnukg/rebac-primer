import { describe, expect, it } from "vitest";
import { createDemoApp } from "../src/app/create-demo.js";

describe("createDemoApp", () => {
  it("given_default_demo_composition_when_app_is_created_then_seed_relationship_graph_is_ready", async () => {
    // Arrange
    const app = createDemoApp();

    // Act
    const decision = await app.authorizer.check({
      user: "user:alice",
      relation: "can_edit",
      object: app.document
    });

    // Assert
    expect(app.actors).toEqual(["user:alice", "user:bob", "user:casey"]);
    expect(app.document).toBe("document:roadmapDocument");
    expect(decision.allowed).toBe(true);
  });
});
