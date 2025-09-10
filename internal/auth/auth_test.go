package auth

import (
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	service := New("test-secret")
	password := "testpassword123"

	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash should not be empty")
	}

	if !service.VerifyPassword(password, hash) {
		t.Fatal("Password verification failed")
	}

	if service.VerifyPassword("wrongpassword", hash) {
		t.Fatal("Wrong password should not verify")
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	service := New("test-secret")
	
	token, err := service.GenerateToken(1, "testuser", false)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Fatal("Token should not be empty")
	}

	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.UserID != 1 {
		t.Fatalf("Expected UserID 1, got %d", claims.UserID)
	}

	if claims.Username != "testuser" {
		t.Fatalf("Expected username 'testuser', got %s", claims.Username)
	}

	if claims.IsAdmin != false {
		t.Fatalf("Expected IsAdmin false, got %t", claims.IsAdmin)
	}
}