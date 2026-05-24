import { describe, expect, it } from "vitest";
import makeDemoTokenVerifier from
    "../src/documents-service/adapters/authn/makeDemoTokenVerifier.ts";

// makeDemoTokenVerifier stands in for a real OAuth2 token verifier.
// In production this would validate a JWT against an IdP's public key and
// return the same { subject, scopes } shape.  The port is identical; only
// the adapter changes.

describe("makeDemoTokenVerifier", () => {
    it("extracts subject and scopes from a valid bearer token", async () => {
        // Arrange
        const authenticator = makeDemoTokenVerifier({
            tokens: { "token-alice": { sub: "alice", scopes: ["documents:read"] } },
        });

        // Act
        const result = await authenticator.verifyAccessToken("Bearer token-alice");

        // Assert
        expect(result).toEqual({
            subject: "user:alice",
            scopes:  ["documents:read"],
        });
    });

    it("throws AuthenticationError when the Authorization header is missing", async () => {
        // Arrange
        const authenticator = makeDemoTokenVerifier({ tokens: {} });

        // Act + Assert
        await expect(authenticator.verifyAccessToken(undefined))
            .rejects.toMatchObject({ name: "AuthenticationError" });
    });

    it("throws AuthenticationError when the token is not in the registry", async () => {
        // Arrange
        const authenticator = makeDemoTokenVerifier({ tokens: {} });

        // Act + Assert
        await expect(authenticator.verifyAccessToken("Bearer nope"))
            .rejects.toMatchObject({ name: "AuthenticationError" });
    });
});
