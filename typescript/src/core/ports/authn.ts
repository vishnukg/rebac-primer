// ── Authentication (authn) port ───────────────────────────────────────────────
//
// Authn answers: "Who is calling?"
// Authz answers: "What may they do?" (see authz.ts)
//
// The HTTP adapter calls verifyAccessToken() to turn a raw bearer token into a
// verified caller identity.  That identity is then passed to domain operations,
// which consult the Authorizer to decide what the caller is allowed to do.
//
// Port direction (hexagonal architecture):
//   HTTP adapter  →  Authenticator  (secondary/driven port — adapter calls out)
//   HTTP adapter  →  Documents      (primary/driving port  — adapter calls into domain)

import type { RebacObject } from "./authz.ts";

// The verified identity returned after a successful token check.
export type AuthenticatedUser = {
    subject: RebacObject<"user">; // e.g. "user:alice" — becomes the actor in authz checks
    scopes:  string[];            // OAuth scopes granted to this token
};

// Raw claims extracted from the token (before converting to domain types).
// In production this comes from a JWT payload or an IdP introspection response.
export type TokenClaims = {
    sub:    string;
    scopes: string[];
};

export type VerifyAccessTokenFn = (
    authorizationHeader: string | undefined,
) => Promise<AuthenticatedUser>;

// Port that HTTP adapters call to establish caller identity.
// makeDemoTokenVerifier is the local-dev adapter.
// A production adapter would verify a signed JWT or call an IdP.
export interface Authenticator {
    verifyAccessToken: VerifyAccessTokenFn;
}

// ── Error ─────────────────────────────────────────────────────────────────────

// A tagged Error — same interface as a standard Error but with a fixed `name`
// for safe discrimination at boundaries (e.g. HTTP error mapping in makeHttpHandler).
export type AuthenticationError = Error & { readonly name: "AuthenticationError" };

export const AuthenticationError = (message: string): AuthenticationError =>
    Object.assign(new Error(message), { name: "AuthenticationError" as const });

export const isAuthenticationError = (e: unknown): e is AuthenticationError =>
    e instanceof Error && e.name === "AuthenticationError";
