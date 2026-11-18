package auth

import "testing"

func TestPasswordHashAndCompare(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("password must not be stored as plain text")
	}
	if err := ComparePassword(hash, "correct horse battery staple"); err != nil {
		t.Fatalf("compare password: %v", err)
	}
	if err := ComparePassword(hash, "wrong password"); err == nil {
		t.Fatal("expected wrong password to be rejected")
	}
}

func TestHashPasswordRejectsEmptyPassword(t *testing.T) {
	if _, err := HashPassword(""); err == nil {
		t.Fatal("expected empty password to be rejected")
	}
}

func TestRefreshTokenIsRandomAndHashable(t *testing.T) {
	first, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate first token: %v", err)
	}
	second, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate second token: %v", err)
	}
	if first == second {
		t.Fatal("refresh tokens must be random")
	}
	if HashRefreshToken(first) == first || len(HashRefreshToken(first)) != 64 {
		t.Fatal("refresh token hash has an unexpected format")
	}
}
