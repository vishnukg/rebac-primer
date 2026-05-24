// Package authn provides the demo token verifier for the documents service.
//
// DemoTokenVerifier looks up bearer tokens in a static map — for local
// development only.  In production this would verify a signed JWT or call an IdP.
//
// Mirrors typescript/src/documents-service/adapters/authn/makeDemoTokenVerifier.ts.
package authn

import (
	"fmt"

	"rebac-primer/internal/documentsservice/core/ports"
	"rebac-primer/internal/shared"
)

// TokenClaims holds the raw claims extracted from a demo token.
type TokenClaims struct {
	Sub    string
	Scopes []string
}

// DemoTokenVerifier satisfies ports.Authenticator using a static token → claims map.
type DemoTokenVerifier struct {
	tokens map[string]TokenClaims
}

// New creates a verifier from a token → claims map.
func New(tokens map[string]TokenClaims) *DemoTokenVerifier {
	return &DemoTokenVerifier{tokens: tokens}
}

// VerifyAccessToken extracts the bearer token from the Authorization header,
// looks it up in the static map, and returns the verified identity.
func (v *DemoTokenVerifier) VerifyAccessToken(authorizationHeader string) (ports.AuthenticatedUser, error) {
	token, ok := extractBearer(authorizationHeader)
	if !ok {
		return ports.AuthenticatedUser{}, &ports.AuthenticationError{
			Message: "missing or malformed Authorization header",
		}
	}

	claims, ok := v.tokens[token]
	if !ok {
		return ports.AuthenticatedUser{}, &ports.AuthenticationError{
			Message: fmt.Sprintf("invalid token: %s", token),
		}
	}

	return ports.AuthenticatedUser{
		Subject: shared.User(claims.Sub),
		Scopes:  claims.Scopes,
	}, nil
}

// Compile-time assertion.
var _ ports.Authenticator = (*DemoTokenVerifier)(nil)

// extractBearer strips the "Bearer " prefix from an Authorization header value.
func extractBearer(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || header[:len(prefix)] != prefix {
		return "", false
	}
	return header[len(prefix):], true
}
