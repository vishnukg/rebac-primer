import { describe, expect, it } from "vitest";
import makeDemoTokenVerifier from
    "../src/documents-service/adapters/authn/makeDemoTokenVerifier.ts";

describe("makeDemoTokenVerifier", () => {
    it("turns a bearer access token into an authenticated user", async () => {
        const authenticator = makeDemoTokenVerifier({
            tokens: { "token-alice": { sub: "alice", scopes: ["documents:read"] } },
        });

        const result = await authenticator.verifyAccessToken("Bearer token-alice");

        expect(result).toEqual({
            subject: "user:alice",
            scopes:  ["documents:read"],
        });
    });

    it("rejects a missing Authorization header", async () => {
        const authenticator = makeDemoTokenVerifier({ tokens: {} });
        await expect(authenticator.verifyAccessToken(undefined)).rejects.toMatchObject({
            name: "AuthenticationError",
        });
    });

    it("rejects an unknown token", async () => {
        const authenticator = makeDemoTokenVerifier({ tokens: {} });
        await expect(authenticator.verifyAccessToken("Bearer nope")).rejects.toMatchObject({
            name: "AuthenticationError",
        });
    });
});
