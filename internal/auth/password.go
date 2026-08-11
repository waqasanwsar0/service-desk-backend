package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

// NOTE: This is a stdlib-only salted+iterated SHA-256 hash so the module
// has zero external dependencies. It works, but bcrypt or argon2id
// (golang.org/x/crypto/bcrypt) is the production-grade choice — swap this
// out once the dev machine has normal internet access for `go get`.
const iterations = 100_000

func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := deriveKey(password, salt)
	return hex.EncodeToString(salt) + "$" + hex.EncodeToString(hash), nil
}

func VerifyPassword(password, stored string) bool {
	parts := strings.SplitN(stored, "$", 2)
	if len(parts) != 2 {
		return false
	}
	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	wantHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	gotHash := deriveKey(password, salt)
	return hmac.Equal(gotHash, wantHash)
}

func deriveKey(password string, salt []byte) []byte {
	h := sha256.Sum256(append(salt, []byte(password)...))
	for i := 0; i < iterations; i++ {
		h = sha256.Sum256(h[:])
	}
	return h[:]
}

var ErrInvalidCredentials = errors.New("invalid email or password")
