import { Readable } from "node:stream";
import { describe, expect, it } from "vitest";
import { isJsonObject, readJson, stringField } from "../src/http/json.js";

describe("JSON HTTP helpers", () => {
  it("given_object_json_body_when_reading_json_then_object_is_returned", async () => {
    // Arrange
    const request = Readable.from([JSON.stringify({ actorId: "alice" })]);

    // Act
    const body = await readJson(request);

    // Assert
    expect(body).toEqual({ actorId: "alice" });
  });

  it("given_empty_request_body_when_reading_json_then_empty_object_is_returned", async () => {
    // Arrange
    const request = Readable.from([]);

    // Act
    const body = await readJson(request);

    // Assert
    expect(body).toEqual({});
  });

  it("given_array_json_body_when_reading_json_then_error_is_thrown", async () => {
    // Arrange
    const request = Readable.from([JSON.stringify(["not", "object"])]);

    // Act
    const readPromise = readJson(request);

    // Assert
    await expect(readPromise).rejects.toThrow("Request body must be a JSON object");
  });

  it("given_unknown_values_when_checking_json_object_then_only_plain_objects_match", () => {
    // Arrange
    const objectValue = { ok: true };
    const nullValue = null;
    const arrayValue = ["nope"];

    // Act
    const objectResult = isJsonObject(objectValue);
    const nullResult = isJsonObject(nullValue);
    const arrayResult = isJsonObject(arrayValue);

    // Assert
    expect(objectResult).toBe(true);
    expect(nullResult).toBe(false);
    expect(arrayResult).toBe(false);
  });

  it("given_json_object_when_reading_string_field_then_non_empty_string_is_required", () => {
    // Arrange
    const validBody = { title: "Roadmap" };
    const invalidBody = { title: "" };

    // Act
    const title = stringField(validBody, "title");

    // Assert
    expect(title).toBe("Roadmap");
    expect(() => stringField(invalidBody, "title")).toThrow("Missing string field: title");
  });
});
