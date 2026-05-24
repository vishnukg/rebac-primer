import { AuthenticationError } from "../../core/index.ts";
import type { Authenticator, TokenClaims } from "../../core/index.ts";
import { user } from "../../core/index.ts";

type DemoTokenVerifierCfg = {
    tokens: Record<string, TokenClaims>;
};

const BEARER_AUTH_HEADER = /^Bearer\s+(.+)$/i;

const makeDemoTokenVerifier = ({ tokens }: DemoTokenVerifierCfg): Authenticator => {
    const verifyAccessToken: Authenticator["verifyAccessToken"] = async authorizationHeader => {
        const token = readBearerToken(authorizationHeader);
        const claims = tokens[token];
        if (!claims) throw new AuthenticationError("Invalid access token");

        return {
            subject: user(claims.sub),
            token,
            scopes:  [...claims.scopes],
        };
    };

    return { verifyAccessToken };
};

const readBearerToken = (authorizationHeader: string | undefined): string => {
    if (!authorizationHeader?.trim()) {
        throw new AuthenticationError("Missing Authorization header");
    }

    const match = BEARER_AUTH_HEADER.exec(authorizationHeader.trim());
    const token = match?.[1]?.trim();
    if (!token) {
        throw new AuthenticationError("Authorization header must be Bearer <token>");
    }

    return token;
};

export default makeDemoTokenVerifier;
