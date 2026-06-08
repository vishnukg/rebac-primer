package documents

import (
	"fmt"

	"rebac-primer/internal/rebac"
)

// TokenClaims holds the raw claims extracted from a demo token.
type TokenClaims struct {
	Sub    string
	Scopes []string
}

// DemoTokenVerifier satisfies [Authenticator] using a static token → claims map.
type DemoTokenVerifier struct {
	tokens map[string]TokenClaims
}

// New creates a verifier from a token → claims map.
func NewDemoTokenVerifier(tokens map[string]TokenClaims) *DemoTokenVerifier {
	return &DemoTokenVerifier{tokens: tokens}
}

// Compile-time assertion: *DemoTokenVerifier must satisfy Authenticator.
var _ Authenticator = (*DemoTokenVerifier)(nil)

// VerifyAccessToken extracts the bearer token from the Authorization header,
// looks it up in the static map, and returns the verified identity.
func (v *DemoTokenVerifier) VerifyAccessToken(authorizationHeader string) (AuthenticatedUser, error) {
	token, ok := extractBearer(authorizationHeader)
	if !ok {
		return AuthenticatedUser{}, &AuthenticationError{
			Message: "missing or malformed Authorization header",
		}
	}

	claims, ok := v.tokens[token]
	if !ok {
		return AuthenticatedUser{}, &AuthenticationError{
			Message: fmt.Sprintf("invalid token: %s", token),
		}
	}

	return AuthenticatedUser{
		Subject: rebac.User(claims.Sub),
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
