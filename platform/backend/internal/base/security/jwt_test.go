package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"wfagent/internal/base/config"
	"wfagent/internal/base/dto"
)

func TestSignAndParse(t *testing.T) {
	setupJWTConfig()

	user := dto.UserInfoResponse{
		UserID:      "42",
		Username:    "alice",
		Roles:       []string{"SYS_ADMIN"},
		Permissions: []string{"*"},
	}

	token, expiresIn, err := SignAccessToken(user)
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}
	if expiresIn != int(config.C.JWT.AccessTTL.Seconds()) {
		t.Fatalf("expires_in = %d, want %d", expiresIn, int(config.C.JWT.AccessTTL.Seconds()))
	}

	claims, err := ParseAccessToken(token)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if claims.UserID != user.UserID || claims.Username != user.Username || claims.TokenType != "access" {
		t.Fatalf("claims mismatch: %#v", claims)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "SYS_ADMIN" {
		t.Fatalf("roles mismatch: %#v", claims.Roles)
	}
	if len(claims.Permissions) != 1 || claims.Permissions[0] != "*" {
		t.Fatalf("permissions mismatch: %#v", claims.Permissions)
	}
}

func TestExpiredToken(t *testing.T) {
	setupJWTConfig()

	claims := Claims{
		UserID:    "42",
		Username:  "alice",
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Minute)),
			Issuer:    config.C.JWT.Issuer,
			Subject:   "42",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(config.C.JWT.AccessSecret))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	if _, err := ParseAccessToken(token); err == nil {
		t.Fatal("expected expired token parse error")
	}
}

func TestParseAccessTokenRejectsUnexpectedIssuerAndAlgorithm(t *testing.T) {
	setupJWTConfig()

	claims := Claims{
		UserID:    "42",
		Username:  "alice",
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "untrusted-issuer",
			Subject:   "42",
		},
	}
	wrongIssuer, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(config.C.JWT.AccessSecret))
	if err != nil {
		t.Fatalf("sign token with unexpected issuer: %v", err)
	}
	if _, err := ParseAccessToken(wrongIssuer); err == nil {
		t.Fatal("expected unexpected issuer to be rejected")
	}

	claims.Issuer = config.C.JWT.Issuer
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign unsigned token: %v", err)
	}
	if _, err := ParseAccessToken(unsigned); err == nil {
		t.Fatal("expected non-HS256 token to be rejected")
	}
}

func setupJWTConfig() {
	config.C = &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test-access-secret",
			RefreshSecret: "test-refresh-secret",
			AccessTTL:     2 * time.Hour,
			RefreshTTL:    7 * 24 * time.Hour,
			Issuer:        "wfagent-test",
		},
	}
}
