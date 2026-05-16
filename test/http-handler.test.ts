import { describe, expect, it } from "vitest";
import { createServices } from "../src/app/create-services.js";
import { handleHttpRequest } from "../src/http/handler.js";

describe("handleHttpRequest", () => {
  it("given_health_request_when_handling_http_request_then_ok_response_is_returned", async () => {
    // Arrange
    const services = await createServices();

    // Act
    const response = await handleHttpRequest(
      { documents: services.documents },
      {
        method: "GET",
        path: "/health",
        query: new URLSearchParams()
      }
    );

    // Assert
    expect(response).toEqual({ statusCode: 200, body: { ok: true } });
  });

  it("given_workspace_viewer_when_reading_document_route_then_document_is_returned", async () => {
    // Arrange
    const services = await createServices();

    // Act
    const response = await handleHttpRequest(
      { documents: services.documents },
      {
        method: "GET",
        path: "/documents/roadmap",
        query: new URLSearchParams({ actorId: "bob" })
      }
    );

    // Assert
    expect(response.statusCode).toBe(200);
    expect(response.body).toMatchObject({
      document: {
        id: "roadmap",
        updatedBy: "user:alice"
      }
    });
  });

  it("given_forbidden_domain_action_when_handling_http_request_then_403_response_is_returned", async () => {
    // Arrange
    const services = await createServices();

    // Act
    const response = await handleHttpRequest(
      { documents: services.documents },
      {
        method: "PATCH",
        path: "/documents/roadmap",
        query: new URLSearchParams(),
        body: { actorId: "bob", body: "Should not save" }
      }
    );

    // Assert
    expect(response.statusCode).toBe(403);
    expect(response.body.error).toContain("cannot edit");
  });

  it("given_invalid_create_body_when_handling_document_create_then_400_response_is_returned", async () => {
    // Arrange
    const services = await createServices();

    // Act
    const response = await handleHttpRequest(
      { documents: services.documents },
      {
        method: "POST",
        path: "/documents",
        query: new URLSearchParams(),
        body: { id: "new-doc", title: "Missing fields" }
      }
    );

    // Assert
    expect(response.statusCode).toBe(400);
    expect(response.body.error).toBe("Missing string field: body");
  });

  it("given_missing_actor_query_when_reading_document_route_then_400_response_is_returned", async () => {
    // Arrange
    const services = await createServices();

    // Act
    const response = await handleHttpRequest(
      { documents: services.documents },
      {
        method: "GET",
        path: "/documents/roadmap",
        query: new URLSearchParams()
      }
    );

    // Assert
    expect(response.statusCode).toBe(400);
    expect(response.body.error).toBe("Missing query parameter: actorId");
  });

  it("given_unknown_route_when_handling_http_request_then_404_response_is_returned", async () => {
    // Arrange
    const services = await createServices();

    // Act
    const response = await handleHttpRequest(
      { documents: services.documents },
      {
        method: "GET",
        path: "/unknown",
        query: new URLSearchParams()
      }
    );

    // Assert
    expect(response).toEqual({ statusCode: 404, body: { error: "Route not found" } });
  });
});
