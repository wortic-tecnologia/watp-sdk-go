package watp

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func (e Envelope) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("id is required")
	}

	if e.Sender == "" {
		return fmt.Errorf("sender is required")
	}

	if !e.Type.IsValid() {
		return fmt.Errorf("unsupported WATP message type %q", e.Type)
	}

	if !IsHexSHA256(e.Hash) {
		return fmt.Errorf("hash must be a lowercase SHA-256 hex string")
	}

	if e.Timestamp == "" {
		return fmt.Errorf("timestamp is required")
	}

	if e.Type != MessageTypeHashInterest && e.NegotiationID == "" {
		return fmt.Errorf("%s requires negotiation_id", e.Type)
	}

	if e.PreviousHash != nil && !IsHexSHA256(*e.PreviousHash) {
		return fmt.Errorf("previousHash must be a lowercase SHA-256 hex string")
	}

	if e.Type != MessageTypeHashInterest && e.PreviousHash == nil {
		return fmt.Errorf("%s requires previousHash", e.Type)
	}

	if e.Type == MessageTypeHashCancelled {
		return nil
	}

	if !payloadHasIdentity(e.Payload) {
		return fmt.Errorf("payload must include at least one of sku, item, or product_name")
	}

	qty, ok := payloadNumericValue(e.Payload, "qty", "quantity")
	if !ok || qty <= 0 {
		return fmt.Errorf("payload must include qty or quantity greater than zero")
	}

	unitPrice, ok := payloadNumericValue(e.Payload, "unit_price", "price")
	if !ok || unitPrice < 0 {
		return fmt.Errorf("payload must include unit_price or price greater than or equal to zero")
	}

	if e.Type == MessageTypeHashInterest && !payloadPublicHTTPSURL(e.Payload, "buyer_url", false) {
		return fmt.Errorf("buyer_url is required for HASH_INTEREST and must be a public HTTPS URL")
	}

	if rawPaymentURL, ok := e.Payload.Get("payment_url"); ok {
		if text, ok := rawPaymentURL.(string); ok && text != "" && !isPublicHTTPSURL(text, true) {
			return fmt.Errorf("payment_url must be a public HTTPS URL")
		}
	}

	return nil
}

func (e Envelope) CanonicalHashInput() CanonicalHashInput {
	return CanonicalHashInput{
		Type:         e.Type,
		Payload:      e.Payload.Clone(),
		PreviousHash: cloneStringPointer(e.PreviousHash),
		Timestamp:    e.Timestamp,
	}
}

func (e Envelope) CanonicalHashJSON() (string, error) {
	return CreateCanonicalMessageBody(e.CanonicalHashInput())
}

func (e Envelope) MarshalJSON() ([]byte, error) {
	fields := make([]Field, 0, 9)
	if e.ID != "" {
		fields = append(fields, KV("id", e.ID))
	}

	if e.NegotiationID != "" {
		fields = append(fields, KV("negotiation_id", e.NegotiationID))
	}

	fields = append(fields,
		KV("type", e.Type.String()),
		KV("payload", e.Payload),
		KV("hash", e.Hash),
	)

	if e.PreviousHash != nil {
		fields = append(fields, KV("previousHash", *e.PreviousHash))
	}

	fields = append(fields, KV("timestamp", e.Timestamp))

	if e.Sender != "" {
		fields = append(fields, KV("sender", e.Sender))
	}

	if e.Signature != nil {
		fields = append(fields, KV("signature", *e.Signature))
	}

	return marshalOrderedObject(fields)
}

func (e *Envelope) UnmarshalJSON(data []byte) error {
	value, err := unmarshalOrderedValue(data)
	if err != nil {
		return err
	}

	object, ok := value.(Payload)
	if !ok {
		return fmt.Errorf("WATP envelope must be a JSON object")
	}

	var envelope Envelope

	if rawID, ok := object.Get("id"); ok {
		id, err := requiredStringFromValue(rawID, "id")
		if err != nil {
			return err
		}

		envelope.ID = id
	}

	if rawNegotiationID, ok := object.Get("negotiation_id"); ok {
		negotiationID, err := requiredStringFromValue(rawNegotiationID, "negotiation_id")
		if err != nil {
			return err
		}

		envelope.NegotiationID = negotiationID
	}

	rawType, ok := object.Get("type")
	if !ok {
		return fmt.Errorf("WATP envelope requires type")
	}

	typeString, err := requiredStringFromValue(rawType, "type")
	if err != nil {
		return err
	}

	envelope.Type, err = ParseMessageType(typeString)
	if err != nil {
		return err
	}

	rawPayload, ok := object.Get("payload")
	if !ok {
		return fmt.Errorf("WATP envelope requires payload")
	}

	payload, ok := rawPayload.(Payload)
	if !ok {
		return fmt.Errorf("payload must be a JSON object")
	}

	envelope.Payload = payload

	rawHash, ok := object.Get("hash")
	if !ok {
		return fmt.Errorf("WATP envelope requires hash")
	}

	envelope.Hash, err = requiredStringFromValue(rawHash, "hash")
	if err != nil {
		return err
	}

	if rawPreviousHash, ok := object.Get("previousHash"); ok {
		envelope.PreviousHash, err = optionalStringPointerFromValue(rawPreviousHash, "previousHash")
		if err != nil {
			return err
		}
	} else if rawPreviousHash, ok := object.Get("prev_hash"); ok {
		envelope.PreviousHash, err = optionalStringPointerFromValue(rawPreviousHash, "prev_hash")
		if err != nil {
			return err
		}
	}

	rawTimestamp, ok := object.Get("timestamp")
	if !ok {
		return fmt.Errorf("WATP envelope requires timestamp")
	}

	envelope.Timestamp, err = requiredStringFromValue(rawTimestamp, "timestamp")
	if err != nil {
		return err
	}

	if rawSender, ok := object.Get("sender"); ok {
		envelope.Sender, err = requiredStringFromValue(rawSender, "sender")
		if err != nil {
			return err
		}
	}

	if rawSignature, ok := object.Get("signature"); ok {
		envelope.Signature, err = optionalStringPointerFromValue(rawSignature, "signature")
		if err != nil {
			return err
		}
	}

	if err := envelope.Validate(); err != nil {
		return err
	}

	*e = envelope
	return nil
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func requiredStringFromValue(value any, fieldName string) (string, error) {
	text, ok := value.(string)
	if !ok || text == "" {
		return "", fmt.Errorf("%s must be a non-empty string", fieldName)
	}

	return text, nil
}

func optionalStringPointerFromValue(value any, fieldName string) (*string, error) {
	if value == nil {
		return nil, nil
	}

	text, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("%s must be a string or null", fieldName)
	}

	return &text, nil
}

func payloadHasIdentity(payload Payload) bool {
	for _, key := range []string{"sku", "item", "product_name"} {
		rawValue, ok := payload.Get(key)
		if !ok {
			continue
		}

		text, ok := rawValue.(string)
		if ok && text != "" {
			return true
		}
	}

	return false
}

func payloadNumericValue(payload Payload, canonicalKey string, aliasKey string) (float64, bool) {
	if rawValue, ok := payload.Get(canonicalKey); ok {
		return numericValue(rawValue)
	}

	if rawValue, ok := payload.Get(aliasKey); ok {
		return numericValue(rawValue)
	}

	return 0, false
}

func payloadPublicHTTPSURL(payload Payload, key string, allowQuery bool) bool {
	rawValue, ok := payload.Get(key)
	if !ok {
		return false
	}

	text, ok := rawValue.(string)
	if !ok {
		return false
	}

	return isPublicHTTPSURL(text, allowQuery)
}

func isPublicHTTPSURL(value string, allowQuery bool) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
		return false
	}

	if parsed.User != nil || parsed.Fragment != "" || (!allowQuery && parsed.RawQuery != "") {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return false
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return true
	}

	return !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}

func numericValue(value any) (float64, bool) {
	switch value := value.(type) {
	case int:
		return float64(value), true
	case int8:
		return float64(value), true
	case int16:
		return float64(value), true
	case int32:
		return float64(value), true
	case int64:
		return float64(value), true
	case uint:
		return float64(value), true
	case uint8:
		return float64(value), true
	case uint16:
		return float64(value), true
	case uint32:
		return float64(value), true
	case uint64:
		return float64(value), true
	case float32:
		return float64(value), true
	case float64:
		return value, true
	case jsonNumber:
		number, err := value.Float64()
		if err != nil {
			return 0, false
		}

		return number, true
	default:
		return 0, false
	}
}

type jsonNumber interface {
	Float64() (float64, error)
}
