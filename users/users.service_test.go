package users

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	hash, err := hashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "correct horse battery" {
		t.Fatal("password stored in plaintext")
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil || cost != bcryptCost {
		t.Fatalf("got cost %d (err %v), want %d", cost, err, bcryptCost)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte("correct horse battery")) != nil {
		t.Fatal("hash does not verify the original password")
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte("wrong password")) == nil {
		t.Fatal("hash verifies a wrong password")
	}
}

func TestHashPasswordRejectsBadLengths(t *testing.T) {
	for _, pw := range []string{"", "short", strings.Repeat("a", maxPasswordBytes+1)} {
		if _, err := hashPassword(pw); !errors.Is(err, ErrInvalidPassword) {
			t.Errorf("password of length %d: got %v, want ErrInvalidPassword", len(pw), err)
		}
	}
}
