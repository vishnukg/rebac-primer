// Driven port — "who is calling?"
// Authz is answered by AuthzClient — "what may they do?"

import type { RebacObject } from "../../../shared/rebac.ts";

export type AuthenticatedUser = {
    subject: RebacObject<"user">;
    scopes:  string[];
};

export type VerifyAccessTokenFn = (header: string | undefined) => Promise<AuthenticatedUser>;

export interface Authenticator {
    verifyAccessToken: VerifyAccessTokenFn;
}

// Tagged error — missing or invalid bearer token → 401.
export type AuthenticationError = Error & { readonly name: "AuthenticationError" };
export const AuthenticationError = (message: string): AuthenticationError =>
    Object.assign(new Error(message), { name: "AuthenticationError" as const });
export const isAuthenticationError = (e: unknown): e is AuthenticationError =>
    e instanceof Error && e.name === "AuthenticationError";
