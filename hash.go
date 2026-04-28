package watp

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"
)

var sha256HexPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func UTCTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
}

func SHA256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func IsHexSHA256(value string) bool {
	return sha256HexPattern.MatchString(value)
}

func CreateCanonicalMessageBody(input CanonicalHashInput) (string, error) {
	if !input.Type.IsValid() {
		return "", fmt.Errorf("unsupported WATP message type %q", input.Type)
	}

	if input.Timestamp == "" {
		return "", fmt.Errorf("timestamp is required")
	}

	body, err := marshalOrderedObject([]Field{
		{Key: "type", Value: input.Type.String()},
		{Key: "payload", Value: input.Payload},
		{Key: "previousHash", Value: nullableStringValue(input.PreviousHash)},
		{Key: "timestamp", Value: input.Timestamp},
	})
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func ComputeMessageHash(input CanonicalHashInput) (string, error) {
	body, err := CreateCanonicalMessageBody(input)
	if err != nil {
		return "", err
	}

	return SHA256Hex(body), nil
}

func nullableStringValue(value *string) any {
	if value == nil {
		return nil
	}

	return *value
}
