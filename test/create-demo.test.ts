import { describe, expect, it } from "vitest";
import { createDemoApp } from "../src/app/create-demo.js";

describe("createDemoApp", () => {
  it("given_default_demo_composition_when_app_is_created_then_tutorial_authorization_graph_is_ready", async () => {
    // Arrange
    const app = createDemoApp();

    // Act
    const decision = await app.authorizer.check({
      user: "user:alice",
      relation: "can_edit",
      object: app.document
    });

    // Assert
    expect(app.actors).toEqual(["user:alice", "user:bob", "user:chandra"]);
    expect(app.document).toBe("document:roadmap");
    expect(decision.allowed).toBe(true);
  });
});
