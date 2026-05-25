// Package authn provides the demo token verifier for the documents service.
//
// DemoTokenVerifier looks up bearer tokens in a static map — for local
// development only.  In production this would verify a signed JWT or call an IdP.
//
// Mirrors typescript/src/documents-service/adapters/authn/makeDemoTokenVerifier.ts.
package authn

import (
	"fmt"

	"rebac-primer/internal/documents"
	"rebac-primer/internal/shared"
)

// TokenClaims holds the raw claims extracted from a demo token.
type TokenClaims struct {
	Sub    string
	Scopes []string
}

// DemoTokenVerifier satisfies [documents.Authenticator] using a static token → claims map.
type DemoTokenVerifier struct {
	tokens map[string]TokenClaims
}

// New creates a verifier from a token → claims map.
func New(tokens map[string]TokenClaims) *DemoTokenVerifier {
	return &DemoTokenVerifier{tokens: tokens}
}

// Compile-time assertion: *DemoTokenVerifier must satisfy documents.Authenticator.
var _ documents.Authenticator = (*DemoTokenVerifier)(nil)

// VerifyAccessToken extracts the bearer token from the Authorization header,
// looks it up in the static map, and returns the verified identity.
func (v *DemoTokenVerifier) VerifyAccessToken(authorizationHeader string) (documents.AuthenticatedUser, error) {
	token, ok := extractBearer(authorizationHeader)
	if !ok {
		return documents.AuthenticatedUser{}, &documents.AuthenticationError{
			Message: "missing or malformed Authorization header",
		}
	}

	claims, ok := v.tokens[token]
	if !ok {
		return documents.AuthenticatedUser{}, &documents.AuthenticationError{
			Message: fmt.Sprintf("invalid token: %s", token),
		}
	}

	return documents.AuthenticatedUser{
		Subject: shared.User(claims.Sub),
		Scopes:  claims.Scopes,
	}, nil
}

// extractBearer strips the "Bearer " prefix from an Authorization header value.
func extractBearer(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || header[:len(prefix)] != prefix {
		return "", false
	}
	return header[len(prefix):], true
}
