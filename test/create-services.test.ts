import { describe, expect, it } from "vitest";
import { createServices } from "../src/app/create-services.js";

describe("createServices", () => {
  it("given_default_composition_when_services_are_created_then_tutorial_document_and_authorizer_are_ready", async () => {
    // Arrange + Act
    const services = await createServices();

    // Act
    const document = await services.documents.read("roadmap", "user:bob");
    const decision = await services.authorizer.check({
      user: "user:alice",
      relation: "can_edit",
      object: "document:roadmap"
    });

    // Assert
    expect(document.title).toBe("Roadmap");
    expect(decision).toMatchObject({ allowed: true });
  });
});
