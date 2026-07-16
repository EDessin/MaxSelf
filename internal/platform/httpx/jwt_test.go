package httpx

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestIssueAndParseToken(t *testing.T) {
	token, err := IssueToken("secret", "user-1", "demo@example.com", "Demo", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	claims, err := ParseToken("secret", token)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}
	if claims.UserID != "user-1" || claims.Subject != "user-1" || claims.Email != "demo@example.com" || claims.DisplayName != "Demo" {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	if _, err := ParseToken("wrong", token); err == nil {
		t.Fatal("expected invalid token error for wrong secret")
	}
}

func TestParseTokenRejectsUnexpectedSigningMethod(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UserID: "user-1"})
	tokenValue, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none token: %v", err)
	}

	_, err = ParseToken("secret", tokenValue)
	if err == nil || !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("expected invalid token error, got %v", err)
	}
}
