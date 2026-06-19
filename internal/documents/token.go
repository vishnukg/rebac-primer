package documents

import (
	"strings"

	"rebac-primer/internal/rebac"
)

// TokenClaims holds the raw claims extracted from a demo token.
type TokenClaims struct {
	Sub    string
	Scopes []string
}

// DemoTokenVerifier verifies bearer tokens using a static token-to-claims map.
type DemoTokenVerifier struct {
	tokens map[string]TokenClaims
}

// NewDemoTokenVerifier creates a verifier from a token → claims map.
func NewDemoTokenVerifier(tokens map[string]TokenClaims) *DemoTokenVerifier {
	copied := make(map[string]TokenClaims, len(tokens))
	for token, claims := range tokens {
		copied[token] = TokenClaims{
			Sub:    claims.Sub,
			Scopes: append([]string(nil), claims.Scopes...),
		}
	}
	return &DemoTokenVerifier{tokens: copied}
}

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
			Message: "invalid token",
		}
	}
	if strings.TrimSpace(claims.Sub) == "" {
		return AuthenticatedUser{}, &AuthenticationError{Message: "invalid token claims"}
	}

	return AuthenticatedUser{
		Subject: rebac.User(claims.Sub),
		Scopes:  append([]string(nil), claims.Scopes...),
	}, nil
}

// extractBearer parses an Authorization header of shape "Bearer <token>".
func extractBearer(header string) (string, bool) {
	fields := strings.Fields(header)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return "", false
	}
	return fields[1], true
}
