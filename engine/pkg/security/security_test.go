package security

import (
	"testing"
	"time"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !CheckPassword("secret123", hash) {
		t.Error("password should match")
	}
	if CheckPassword("wrong", hash) {
		t.Error("wrong password should not match")
	}
}

func TestHashPassword_TooShort(t *testing.T) {
	_, err := HashPassword("abc")
	if err == nil {
		t.Fatal("expected error for short password")
	}
}

func TestJWT_GenerateAndValidate(t *testing.T) {
	cfg := JWTConfig{Secret: "test-secret-key-32chars-minimum!", Expiration: time.Hour}

	token, err := GenerateToken(cfg, "user-1", "admin", []string{"admin"}, []string{"base.admin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	claims, err := ValidateToken(cfg, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", claims.UserID)
	}
	if claims.Username != "admin" {
		t.Errorf("expected admin, got %s", claims.Username)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "admin" {
		t.Errorf("expected [admin], got %v", claims.Roles)
	}
	if len(claims.Groups) != 1 || claims.Groups[0] != "base.admin" {
		t.Errorf("expected [base.admin], got %v", claims.Groups)
	}
}

func TestJWT_InvalidToken(t *testing.T) {
	cfg := JWTConfig{Secret: "test-secret-key-32chars-minimum!"}
	_, err := ValidateToken(cfg, "invalid.token.here")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestJWT_WrongSecret(t *testing.T) {
	cfg1 := JWTConfig{Secret: "secret-one", Expiration: time.Hour}
	cfg2 := JWTConfig{Secret: "secret-two"}

	token, _ := GenerateToken(cfg1, "user-1", "admin", nil, nil)
	_, err := ValidateToken(cfg2, token)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}
