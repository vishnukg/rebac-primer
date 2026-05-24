import { describe, expect, it } from "vitest";
import makeDemoTokenVerifier from "../src/adapters/authn/makeDemoTokenVerifier.ts";

describe("makeDemoTokenVerifier", () => {
    it("turns a bearer access token into an authenticated user", async () => {
        const authenticator = makeDemoTokenVerifier({
            tokens: { "token-alice": { sub: "alice", scopes: ["documents:read"] } },
        });

        const result = await authenticator.verifyAccessToken("bearer   token-alice");

        expect(result).toEqual({
            subject: "user:alice",
            token:   "token-alice",
            scopes:  ["documents:read"],
        });
    });

    it("rejects missing and unknown tokens", async () => {
        const authenticator = makeDemoTokenVerifier({ tokens: {} });

        await expect(authenticator.verifyAccessToken(undefined)).rejects.toThrow(
            "Missing Authorization header",
        );
        await expect(authenticator.verifyAccessToken("Bearer nope")).rejects.toThrow(
            "Invalid access token",
        );
    });
});
