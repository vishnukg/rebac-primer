// Demo bearer token verifier — for local development and tests only.
//
// In production, replace this with a JWT verifier (e.g. jose / jsonwebtoken) or
// an OAuth introspection call to your IdP. The Authenticator interface stays the
// same; only this file changes.

import { AuthenticationError } from "../../core/index.ts";
import type { Authenticator, TokenClaims } from "../../core/index.ts";
import { user } from "../../core/index.ts";

type DemoTokenVerifierCfg = {
    tokens: Record<string, TokenClaims>; // static token → claims lookup table
};

const BEARER_HEADER = /^Bearer\s+(.+)$/i;

const makeDemoTokenVerifier = ({ tokens }: DemoTokenVerifierCfg): Authenticator => {
    const verifyAccessToken: Authenticator["verifyAccessToken"] = async authorizationHeader => {
        const token  = extractBearerToken(authorizationHeader);
        const claims = tokens[token];
        if (!claims) throw AuthenticationError("Invalid access token");

        return {
            subject: user(claims.sub), // "alice" → "user:alice"
            scopes:  [...claims.scopes],
        };
    };

    return { verifyAccessToken };
};

// Extracts the raw token value from "Authorization: Bearer <token>".
// Throws AuthenticationError for missing or malformed headers.
const extractBearerToken = (header: string | undefined): string => {
    if (!header?.trim()) {
        throw AuthenticationError("Missing Authorization header");
    }

    const token = BEARER_HEADER.exec(header.trim())?.[1]?.trim();
    if (!token) {
        throw AuthenticationError("Authorization header must be: Bearer <token>");
    }

    return token;
};

export default makeDemoTokenVerifier;
