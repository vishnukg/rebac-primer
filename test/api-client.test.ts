import { describe, expect, it } from "vitest";
import { RebacApiClient, type Fetcher } from "../src/client/api-client.js";

describe("RebacApiClient", () => {
  it("given_healthy_server_when_checking_health_then_true_is_returned", async () => {
    // Arrange
    const client = new RebacApiClient("http://server.test", async (url) => {
      expect(url.toString()).toBe("http://server.test/health");
      return new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "content-type": "application/json" }
      });
    });

    // Act
    const healthy = await client.health();

    // Assert
    expect(healthy).toBe(true);
  });

  it("given_document_response_when_reading_document_then_document_is_returned", async () => {
    // Arrange
    const fetcher: Fetcher = async (url) => {
      expect(url.toString()).toBe("http://server.test/documents/roadmap?actorId=alice");
      return new Response(JSON.stringify({
        document: { id: "roadmap", title: "Roadmap", body: "v1", workspace: "workspace:acme", updatedBy: "user:alice" }
      }), {
        status: 200,
        headers: { "content-type": "application/json" }
      });
    };
    const client = new RebacApiClient("http://server.test", fetcher);

    // Act
    const document = await client.readDocument("roadmap", "alice");

    // Assert
    expect(document.id).toBe("roadmap");
  });

  it("given_denied_response_when_updating_document_then_server_error_message_is_thrown", async () => {
    // Arrange
    const fetcher: Fetcher = async () =>
      new Response(JSON.stringify({ error: "user:bob cannot edit document:roadmap" }), {
        status: 403,
        headers: { "content-type": "application/json" }
      });
    const client = new RebacApiClient("http://server.test", fetcher);

    // Act
    const updatePromise = client.updateDocument("roadmap", "bob", "nope");

    // Assert
    await expect(updatePromise).rejects.toThrow("user:bob cannot edit document:roadmap");
  });

  it("given_error_response_without_error_field_when_reading_document_then_status_text_is_thrown", async () => {
    // Arrange
    const client = new RebacApiClient("http://server.test", async () =>
      new Response(JSON.stringify({ message: "nope" }), {
        status: 500,
        statusText: "Internal Server Error",
        headers: { "content-type": "application/json" }
      })
    );

    // Act
    const readPromise = client.readDocument("roadmap", "alice");

    // Assert
    await expect(readPromise).rejects.toThrow("Internal Server Error");
  });

  it("given_success_response_with_invalid_document_body_when_reading_document_then_validation_error_is_thrown", async () => {
    // Arrange
    const client = new RebacApiClient("http://server.test", async () =>
      new Response(JSON.stringify({ document: { id: "roadmap" } }), {
        status: 200,
        headers: { "content-type": "application/json" }
      })
    );

    // Act
    const readPromise = client.readDocument("roadmap", "alice");

    // Assert
    await expect(readPromise).rejects.toThrow("Response body did not contain a document");
  });
});
