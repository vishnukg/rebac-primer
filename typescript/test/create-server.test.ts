import { describe, expect, it } from "vitest";
import { createServerApp } from "../src/app/create-server.js";

describe("createServerApp", () => {
  it("given_port_environment_variable_when_server_app_is_created_then_configured_port_and_http_server_are_returned", async () => {
    // Arrange
    const env = { PORT: "4999" };

    // Act
    const app = await createServerApp(env);

    // Assert
    expect(app.port).toBe(4999);
    expect(app.server.listening).toBe(false);
  });

  it("given_invalid_port_environment_variable_when_server_app_is_created_then_error_is_thrown", async () => {
    // Arrange
    const env = { PORT: "not-a-port" };

    // Act
    const createAction = () => createServerApp(env);

    // Assert
    await expect(createAction).rejects.toThrow("Invalid PORT");
  });
});
