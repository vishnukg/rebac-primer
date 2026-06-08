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

func TestVerifier_GivenLowercaseBearerScheme_WhenVerified_ThenReturnsAuthenticatedUser(t *testing.T) {
	// Arrange
	verifier := newVerifier()

	// Act
	user, err := verifier.VerifyAccessToken("bearer demo-token-alice")

	// Assert
	if err != nil {
		t.Fatalf("VerifyAccessToken returned unexpected error: %v", err)
	}
	if user.Subject != rebac.User("alice") {
		t.Errorf("Subject = %q, want %q", user.Subject, rebac.User("alice"))
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

func TestVerifier_GivenBearerHeaderWithExtraParts_WhenVerified_ThenReturnsAuthenticationError(t *testing.T) {
	// Arrange
	verifier := newVerifier()

	// Act
	_, err := verifier.VerifyAccessToken("Bearer demo-token-alice extra")

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

func TestVerifier_GivenTokenWithEmptySubject_WhenVerified_ThenReturnsAuthenticationError(t *testing.T) {
	// Arrange
	verifier := documents.NewDemoTokenVerifier(map[string]documents.TokenClaims{
		"bad-token": {Sub: "   ", Scopes: []string{"documents:read"}},
	})

	// Act
	_, err := verifier.VerifyAccessToken("Bearer bad-token")

	// Assert
	if !documents.IsAuthenticationError(err) {
		t.Errorf("error = %v, want an AuthenticationError", err)
	}
}

func TestVerifier_GivenCallerMutatesSeedClaims_WhenVerified_ThenUsesSnapshot(t *testing.T) {
	// Arrange
	scopes := []string{"documents:read"}
	tokens := map[string]documents.TokenClaims{
		"demo-token-alice": {Sub: "alice", Scopes: scopes},
	}
	verifier := documents.NewDemoTokenVerifier(tokens)
	tokens["demo-token-alice"] = documents.TokenClaims{Sub: "mallory"}
	scopes[0] = "documents:admin"

	// Act
	user, err := verifier.VerifyAccessToken("Bearer demo-token-alice")

	// Assert
	if err != nil {
		t.Fatalf("VerifyAccessToken returned unexpected error: %v", err)
	}
	if user.Subject != rebac.User("alice") {
		t.Errorf("Subject = %q, want %q", user.Subject, rebac.User("alice"))
	}
	if got := user.Scopes[0]; got != "documents:read" {
		t.Errorf("Scopes[0] = %q, want %q", got, "documents:read")
	}
}

func TestVerifier_GivenCallerMutatesReturnedScopes_WhenVerifiedAgain_ThenUsesSnapshot(t *testing.T) {
	// Arrange
	verifier := newVerifier()
	user, err := verifier.VerifyAccessToken("Bearer demo-token-alice")
	if err != nil {
		t.Fatalf("VerifyAccessToken returned unexpected error: %v", err)
	}

	// Act
	user.Scopes[0] = "documents:admin"
	again, err := verifier.VerifyAccessToken("Bearer demo-token-alice")

	// Assert
	if err != nil {
		t.Fatalf("VerifyAccessToken returned unexpected error: %v", err)
	}
	if got := again.Scopes[0]; got != "documents:read" {
		t.Errorf("Scopes[0] = %q, want %q", got, "documents:read")
	}
}
