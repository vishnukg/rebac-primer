import { describe, expect, it, vi } from "vitest";
import { OpenFgaAuthorizer } from "../src/authz/openfga-client.js";
import { document, tuple, user } from "../src/authz/types.js";

const sdk = vi.hoisted(() => ({
  check: vi.fn(),
  write: vi.fn(),
  constructor: vi.fn()
}));

vi.mock("@openfga/sdk", () => ({
  OpenFgaClient: sdk.constructor.mockImplementation(() => ({
    check: sdk.check,
    write: sdk.write
  }))
}));

describe("OpenFgaAuthorizer", () => {
  it("given_allowed_sdk_response_when_checking_access_then_allowed_result_is_returned", async () => {
    // Arrange
    sdk.check.mockReset();
    sdk.write.mockReset();
    sdk.constructor.mockClear();
    sdk.check.mockResolvedValue({ allowed: true });
    const authorizer = new OpenFgaAuthorizer({
      apiUrl: "http://openfga.test",
      storeId: "store-1"
    });

    // Act
    const result = await authorizer.check({
      user: user("alice"),
      relation: "can_read",
      object: document("roadmapDocument")
    });

    // Assert
    expect(result.allowed).toBe(true);
    expect(result.trace).toEqual(["OpenFGA evaluated the relationship graph remotely"]);
    expect(sdk.check).toHaveBeenCalledWith({
      user: "user:alice",
      relation: "can_read",
      object: "document:roadmapDocument"
    });
  });

  it("given_tuple_keys_when_writing_tuples_then_sdk_receives_tuple_writes", async () => {
    // Arrange
    sdk.check.mockReset();
    sdk.write.mockReset();
    sdk.constructor.mockClear();
    sdk.write.mockResolvedValue({});
    const authorizer = new OpenFgaAuthorizer({
      apiUrl: "http://openfga.test",
      storeId: "store-1"
    });
    const ownerTuple = tuple(document("roadmapDocument"), "owner", user("alice"));

    // Act
    await authorizer.writeTuples([ownerTuple]);

    // Assert
    expect(sdk.write).toHaveBeenCalledWith({
      writes: [
        {
          user: "user:alice",
          relation: "owner",
          object: "document:roadmapDocument"
        }
      ]
    });
  });
});
