package documents_test

import (
	"testing"

	"rebac-primer/internal/documents"
	"rebac-primer/internal/rebac"
)

// These tests cover the demo token verifier. Its only collaborator is a static
// token → claims map, which is plain test data (a fixture), not a stub or mock:
// there is no behaviour to fake and no interaction to verify.

func newVerifier() *documents.DemoTokenVerifier {
	return documents.NewDemoTokenVerifier(map[string]documents.TokenClaims{
		"demo-token-alice": {Sub: "alice", Scopes: []string{"documents:read", "documents:write"}},
	})
}

func TestVerifier_GivenValidBearerToken_WhenVerified_ThenReturnsAuthenticatedUser(t *testing.T) {
	// Arrange
	verifier := newVerifier()

	// Act
	user, err := verifier.VerifyAccessToken("Bearer demo-token-alice")

	// Assert
	if err != nil {
		t.Fatalf("VerifyAccessToken returned unexpected error: %v", err)
	}
	if user.Subject != rebac.User("alice") {
		t.Errorf("Subject = %q, want %q", user.Subject, rebac.User("alice"))
	}
	if len(user.Scopes) != 2 {
		t.Errorf("Scopes = %v, want 2 scopes", user.Scopes)
	}
}

func TestVerifier_GivenMissingHeader_WhenVerified_ThenReturnsAuthenticationError(t *testing.T) {
	// Arrange
	verifier := newVerifier()

	// Act
	_, err := verifier.VerifyAccessToken("")

	// Assert
	if !documents.IsAuthenticationError(err) {
		t.Errorf("error = %v, want an AuthenticationError", err)
	}
}

func TestVerifier_GivenHeaderWithoutBearerPrefix_WhenVerified_ThenReturnsAuthenticationError(t *testing.T) {
	// Arrange
	verifier := newVerifier()

	// Act
	_, err := verifier.VerifyAccessToken("demo-token-alice")

	// Assert
	if !documents.IsAuthenticationError(err) {
		t.Errorf("error = %v, want an AuthenticationError", err)
	}
}

func TestVerifier_GivenUnknownToken_WhenVerified_ThenReturnsAuthenticationError(t *testing.T) {
	// Arrange
	verifier := newVerifier()

	// Act
	_, err := verifier.VerifyAccessToken("Bearer not-a-real-token")

	// Assert
	if !documents.IsAuthenticationError(err) {
		t.Errorf("error = %v, want an AuthenticationError", err)
	}
}
