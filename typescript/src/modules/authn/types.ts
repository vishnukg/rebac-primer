import type { RebacObject } from "../authz/index.ts";

export type AuthenticatedUser = {
  subject: RebacObject<"user">;
  token:   string;
  scopes:  string[];
};

export type TokenClaims = {
  sub:    string;
  scopes: string[];
};

export type VerifyAccessTokenFn = (authorizationHeader: string | undefined) => Promise<AuthenticatedUser>;

export interface Authenticator {
  verifyAccessToken: VerifyAccessTokenFn;
}

export class AuthenticationError extends Error {
  constructor(message: string) {
    super(message);
  }
}
