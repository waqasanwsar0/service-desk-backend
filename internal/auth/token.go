package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"time"
)

var (
	ErrTokenExpired = errors.New("token expired")
	ErrTokenInvalid = errors.New("token invalid")
)

// secret comes from env in production; falls back to a dev default so the
// module runs out of the box. ALWAYS set SERVICEDESK_AUTH_SECRET in any
// real deployment.
func secret() []byte {
	if s := os.Getenv("SERVICEDESK_AUTH_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("dev-only-insecure-secret-change-me")
}

type Claims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"exp"`
}

// GenerateToken produces a compact "<base64 payload>.<base64 signature>" token.
// It's deliberately simple (not a full JWT implementation) but follows the
// same sign-and-verify principle, using only the stdlib.
func GenerateToken(userID, email, role string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		ExpiresAt: time.Now().Add(ttl).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)

	mac := hmac.New(sha256.New, secret())
	mac.Write([]byte(encodedPayload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return encodedPayload + "." + sig, nil
}

func ParseToken(token string) (*Claims, error) {
	dot := indexByte(token, '.')
	if dot < 0 {
		return nil, ErrTokenInvalid
	}
	encodedPayload, sig := token[:dot], token[dot+1:]

	mac := hmac.New(sha256.New, secret())
	mac.Write([]byte(encodedPayload))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return nil, ErrTokenInvalid
	}

	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrTokenInvalid
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, ErrTokenExpired
	}
	return &claims, nil
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
