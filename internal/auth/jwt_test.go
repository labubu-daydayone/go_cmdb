package auth

import (
	"testing"
	"time"
)

func TestGenerateAndParseToken(t *testing.T) {
	// Initialize JWT secret
	InitJWT("test-secret-key")
	
	uid := 1
	username := "testuser"
	role := "admin"
	expireAt := time.Now().Add(24 * time.Hour)
	issuer := "go_cmdb"
	
	// Generate token
	token, err := GenerateToken(uid, username, role, expireAt, issuer)
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}
	
	if token == "" {
		t.Error("Expected non-empty token")
	}
	
	// Parse token
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() failed: %v", err)
	}
	
	// Verify claims
	if claims.UID != uid {
		t.Errorf("Expected UID %d, got %d", uid, claims.UID)
	}
	
	if claims.Username != username {
		t.Errorf("Expected username %s, got %s", username, claims.Username)
	}
	
	if claims.Role != role {
		t.Errorf("Expected role %s, got %s", role, claims.Role)
	}
	
	if claims.Issuer != issuer {
		t.Errorf("Expected issuer %s, got %s", issuer, claims.Issuer)
	}
}

func TestParseToken_InvalidToken(t *testing.T) {
	InitJWT("test-secret-key")
	
	// Test with invalid token
	_, err := ParseToken("invalid.token.string")
	if err == nil {
		t.Error("ParseToken() should fail for invalid token")
	}
}

func TestParseToken_ExpiredToken(t *testing.T) {
	InitJWT("test-secret-key")
	
	// Generate token that's already expired
	expireAt := time.Now().Add(-1 * time.Hour)
	token, err := GenerateToken(1, "testuser", "admin", expireAt, "go_cmdb")
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}
	
	// Try to parse expired token
	_, err = ParseToken(token)
	if err == nil {
		t.Error("ParseToken() should fail for expired token")
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	InitJWT("secret-1")
	
	token, err := GenerateToken(1, "testuser", "admin", time.Now().Add(24*time.Hour), "go_cmdb")
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}
	
	// Change secret
	InitJWT("secret-2")
	
	// Try to parse with different secret
	_, err = ParseToken(token)
	if err == nil {
		t.Error("ParseToken() should fail when secret is different")
	}
}

func TestGenerateToken_UninitializedSecret(t *testing.T) {
	// Reset secret
	jwtSecret = nil
	
	_, err := GenerateToken(1, "testuser", "admin", time.Now().Add(24*time.Hour), "go_cmdb")
	if err == nil {
		t.Error("GenerateToken() should fail when secret is not initialized")
	}
	
	// Restore secret for other tests
	InitJWT("test-secret-key")
}
