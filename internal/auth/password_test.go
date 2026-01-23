package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	plain := "testpassword123"
	
	hash, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword() failed: %v", err)
	}
	
	if hash == "" {
		t.Error("Expected non-empty hash")
	}
	
	if hash == plain {
		t.Error("Hash should not equal plain text password")
	}
}

func TestComparePassword(t *testing.T) {
	plain := "testpassword123"
	
	hash, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword() failed: %v", err)
	}
	
	// Test correct password
	if err := ComparePassword(hash, plain); err != nil {
		t.Errorf("ComparePassword() failed for correct password: %v", err)
	}
	
	// Test wrong password
	if err := ComparePassword(hash, "wrongpassword"); err == nil {
		t.Error("ComparePassword() should fail for wrong password")
	}
}

func TestComparePassword_DifferentHashes(t *testing.T) {
	plain := "testpassword123"
	
	hash1, _ := HashPassword(plain)
	hash2, _ := HashPassword(plain)
	
	// Bcrypt should generate different hashes for the same password
	if hash1 == hash2 {
		t.Error("Expected different hashes for same password (bcrypt salt)")
	}
	
	// But both should validate correctly
	if err := ComparePassword(hash1, plain); err != nil {
		t.Error("First hash should validate")
	}
	
	if err := ComparePassword(hash2, plain); err != nil {
		t.Error("Second hash should validate")
	}
}
