package watp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func BuildNodeSignature(rawBody string, secret string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("secret is required")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(rawBody))

	return "sha256=" + hex.EncodeToString(mac.Sum(nil)), nil
}

func VerifyNodeSignature(rawBody string, signature string, secret string) bool {
	if signature == "" || secret == "" {
		return false
	}

	expected, err := BuildNodeSignature(rawBody, secret)
	if err != nil {
		return false
	}

	return hmac.Equal([]byte(expected), []byte(signature))
}
