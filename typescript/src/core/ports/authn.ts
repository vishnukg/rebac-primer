import type { RebacObject } from "./authz.ts";

// Authn answers "who is calling?" It does not decide what the caller may do.
// Authorization decisions belong to the Authorizer port in authz.ts.
export type AuthenticatedUser = {
    subject: RebacObject<"user">;
    token:   string;
    scopes:  string[];
};

export type TokenClaims = {
    sub:    string;
    scopes: string[];
};

export type VerifyAccessTokenFn = (
    authorizationHeader: string | undefined,
) => Promise<AuthenticatedUser>;

// Driven port: inbound adapters call this to verify bearer access tokens.
export interface Authenticator {
    verifyAccessToken: VerifyAccessTokenFn;
}

export class AuthenticationError extends Error {
    constructor(message: string) {
        super(message);
        this.name = "AuthenticationError";
    }
}
