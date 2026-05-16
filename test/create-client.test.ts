import { describe, expect, it } from "vitest";
import { createClientApp } from "../src/app/create-client.js";

describe("createClientApp", () => {
  it("given_api_url_environment_variable_when_client_app_is_created_then_terminal_and_runner_are_returned", () => {
    // Arrange
    const env = { REBAC_API_URL: "http://127.0.0.1:4999" };

    // Act
    const app = createClientApp(env);
    app.terminal.close();

    // Assert
    expect(typeof app.run).toBe("function");
    expect(typeof app.terminal.close).toBe("function");
  });
});
