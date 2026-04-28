package watp

import (
	"crypto/rand"
	"fmt"
	"math"
	"reflect"
)

type BuilderError struct {
	message string
}

func (e *BuilderError) Error() string {
	return e.message
}

func BuildHashInterestMessage(input BuildHashInterestMessageInput) (Envelope, error) {
	return buildMessage(
		MessageTypeHashInterest,
		input.BuildMessageInput,
		input.NegotiationID,
		optionalStringPointer(input.PreviousHash),
	)
}

func BuildHashCounterMessage(input BuildHashCounterMessageInput) (Envelope, error) {
	previousHash := input.PreviousHash
	return buildMessage(
		MessageTypeHashCounter,
		input.BuildMessageInput,
		input.NegotiationID,
		&previousHash,
	)
}

func BuildHashConcludedMessage(input BuildHashConcludedMessageInput) (Envelope, error) {
	previousHash := input.PreviousHash
	return buildMessage(
		MessageTypeHashConcluded,
		input.BuildMessageInput,
		input.NegotiationID,
		&previousHash,
	)
}

func BuildHashCancelledMessage(input BuildHashCancelledMessageInput) (Envelope, error) {
	previousHash := input.PreviousHash
	return buildMessage(
		MessageTypeHashCancelled,
		input.BuildMessageInput,
		input.NegotiationID,
		&previousHash,
	)
}

func buildMessage(
	messageType MessageType,
	base BuildMessageInput,
	negotiationID string,
	previousHash *string,
) (Envelope, error) {
	timestamp := base.Timestamp
	if timestamp == "" {
		timestamp = UTCTimestamp()
	}

	messageID := base.ID
	if messageID == "" {
		var err error
		messageID, err = newUUID()
		if err != nil {
			return Envelope{}, err
		}
	}

	if base.Sender == "" {
		return Envelope{}, newBuilderError("sender is required")
	}

	payload := base.Payload.Clone()
	if messageType != MessageTypeHashCancelled {
		var err error
		payload, err = normalizeNegotiationPayload(payload)
		if err != nil {
			return Envelope{}, err
		}

		if err := validateNegotiationPayload(payload); err != nil {
			return Envelope{}, err
		}
	}

	if messageType != MessageTypeHashInterest {
		if negotiationID == "" {
			return Envelope{}, newBuilderError("negotiationID is required for chained messages")
		}

		if previousHash == nil || !IsHexSHA256(*previousHash) {
			return Envelope{}, newBuilderError("previousHash must be a lowercase SHA-256 hex string")
		}
	} else if previousHash != nil && !IsHexSHA256(*previousHash) {
		return Envelope{}, newBuilderError("previousHash must be a lowercase SHA-256 hex string")
	}

	hash, err := ComputeMessageHash(CanonicalHashInput{
		Type:         messageType,
		Payload:      payload,
		PreviousHash: previousHash,
		Timestamp:    timestamp,
	})
	if err != nil {
		return Envelope{}, err
	}

	envelope := Envelope{
		ID:            messageID,
		NegotiationID: negotiationID,
		Type:          messageType,
		Payload:       payload,
		Hash:          hash,
		PreviousHash:  cloneStringPointer(previousHash),
		Timestamp:     timestamp,
		Sender:        base.Sender,
	}

	if base.Signature != "" {
		envelope.Signature = &base.Signature
	}

	if err := envelope.Validate(); err != nil {
		return Envelope{}, err
	}

	return envelope, nil
}

func validateNegotiationPayload(payload Payload) error {
	if !payloadHasIdentity(payload) {
		return newBuilderError("payload must include at least one of sku, item, or product_name")
	}

	qty, ok := payloadNumericValue(payload, "qty", "quantity")
	if !ok || !isFinite(qty) || qty <= 0 {
		return newBuilderError("payload qty or quantity must be a finite number greater than zero")
	}

	unitPrice, ok := payloadNumericValue(payload, "unit_price", "price")
	if !ok || !isFinite(unitPrice) || unitPrice < 0 {
		return newBuilderError("payload unit_price or price must be a finite number greater than or equal to zero")
	}

	return nil
}

func normalizeNegotiationPayload(payload Payload) (Payload, error) {
	normalized := payload.Clone()

	var err error
	normalized, err = normalizeAlias(normalized, "qty", "quantity")
	if err != nil {
		return Payload{}, err
	}

	normalized, err = normalizeAlias(normalized, "unit_price", "price")
	if err != nil {
		return Payload{}, err
	}

	return normalized, nil
}

func normalizeAlias(payload Payload, canonicalKey string, aliasKey string) (Payload, error) {
	canonicalIndex := fieldIndex(payload.fields, canonicalKey)
	aliasIndex := fieldIndex(payload.fields, aliasKey)

	switch {
	case aliasIndex == -1:
		return payload, nil
	case canonicalIndex == -1:
		payload.fields[aliasIndex].Key = canonicalKey
		return payload, nil
	default:
		if !valuesEquivalent(payload.fields[canonicalIndex].Value, payload.fields[aliasIndex].Value) {
			return Payload{}, newBuilderError(
				"payload cannot contain both %s and %s with different values",
				canonicalKey,
				aliasKey,
			)
		}

		payload.fields = append(payload.fields[:aliasIndex], payload.fields[aliasIndex+1:]...)
		return payload, nil
	}
}

func fieldIndex(fields []Field, key string) int {
	for index, field := range fields {
		if field.Key == key {
			return index
		}
	}

	return -1
}

func valuesEquivalent(left any, right any) bool {
	leftNumber, leftIsNumber := numericValue(left)
	rightNumber, rightIsNumber := numericValue(right)
	if leftIsNumber && rightIsNumber {
		return leftNumber == rightNumber
	}

	return reflect.DeepEqual(left, right)
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func optionalStringPointer(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func newUUID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate UUID: %w", err)
	}

	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		raw[0:4],
		raw[4:6],
		raw[6:8],
		raw[8:10],
		raw[10:16],
	), nil
}

func newBuilderError(format string, args ...any) error {
	return &BuilderError{message: fmt.Sprintf(format, args...)}
}
