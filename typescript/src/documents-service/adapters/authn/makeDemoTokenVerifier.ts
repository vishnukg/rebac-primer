import { user } from "../../../shared/rebac.ts";
import { AuthenticationError } from "../../core/ports/authenticator.ts";
import type { Authenticator, AuthenticatedUser, TokenClaims } from "../../core/ports/authenticator.ts";

type Cfg = { tokens: Record<string, TokenClaims> };

const makeDemoTokenVerifier = ({ tokens }: Cfg): Authenticator => ({
    verifyAccessToken: async (header): Promise<AuthenticatedUser> => {
        const token = extractBearer(header);
        if (!token) throw AuthenticationError("Missing or malformed Authorization header");
        const claims = tokens[token];
        if (!claims) throw AuthenticationError(`Invalid token: ${token}`);
        return { subject: user(claims.sub), scopes: claims.scopes };
    },
});

const extractBearer = (header: string | undefined): string | undefined => {
    if (!header?.startsWith("Bearer ")) return undefined;
    const token = header.slice(7).trim();
    return token || undefined;
};

export default makeDemoTokenVerifier;
