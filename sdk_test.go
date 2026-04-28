package watp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComputeMessageHashMatchesProtocolFixtures(t *testing.T) {
	t.Parallel()

	concludedPreviousHash := strings.Repeat("a", 64)

	testCases := []struct {
		name         string
		messageType  MessageType
		payload      Payload
		previousHash *string
		timestamp    string
		wantHash     string
	}{
		{
			name:        "hash interest",
			messageType: MessageTypeHashInterest,
			payload: NewPayload(
				KV("item", "Patch Cord Cat6 1.5m"),
				KV("sku", "TC-PC6-150"),
				KV("qty", 10),
				KV("unit_price", 12.9),
				KV("currency", "BRL"),
				KV("buyer", "site-b"),
				KV("buyer_name", "Site B"),
				KV("buyer_url", "https://site-b.wortic.com"),
			),
			timestamp: "2026-04-23T05:40:00.000000Z",
			wantHash:  "4c5ee5b2dc6a546afe29b11a0e7d20dd41d7dd61bcca4b119a1310c8c3b54475",
		},
		{
			name:        "hash concluded with payment url",
			messageType: MessageTypeHashConcluded,
			payload: NewPayload(
				KV("item", "Patch Cord Cat6 1.5m"),
				KV("sku", "TC-PC6-150"),
				KV("qty", 10),
				KV("unit_price", 12.9),
				KV("currency", "BRL"),
				KV("accepted_by", "supplier"),
				KV("payment_url", "https://produtos.williamschons.com.br/checkout/order-pay/1234/?pay_for_order=true&key=wc_order_abc123"),
			),
			previousHash: &concludedPreviousHash,
			timestamp:    "2026-04-23T05:41:00.000000Z",
			wantHash:     "dd289952f18573901289e88371061875b550675ce68e8c1bd9ee1ef1c0122b15",
		},
		{
			name:        "php compatible whole-number float",
			messageType: MessageTypeHashInterest,
			payload: NewPayload(
				KV("sku", "SKU-100"),
				KV("qty", 1),
				KV("unit_price", 100.0),
				KV("currency", "BRL"),
			),
			timestamp: "2026-04-28T00:00:00.000000Z",
			wantHash:  "351a76ec467a22fcdde02586af196e866f0fd07e3609b3b145be568b4e02d56a",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			hash, err := ComputeMessageHash(CanonicalHashInput{
				Type:         testCase.messageType,
				Payload:      testCase.payload,
				PreviousHash: testCase.previousHash,
				Timestamp:    testCase.timestamp,
			})
			if err != nil {
				t.Fatalf("ComputeMessageHash() error = %v", err)
			}

			if hash != testCase.wantHash {
				t.Fatalf("ComputeMessageHash() = %s, want %s", hash, testCase.wantHash)
			}
		})
	}
}

func TestProtocolFixturesConformance(t *testing.T) {
	t.Parallel()

	testCases := map[string]bool{
		"valid-hash-interest.json":                         true,
		"valid-hash-concluded-payment-url.json":            true,
		"invalid-hash-interest-missing-buyer-url.json":     false,
		"invalid-hash-counter-missing-negotiation-id.json": false,
		"invalid-hash-concluded-invalid-payment-url.json":  false,
	}

	for fileName, shouldPass := range testCases {
		fileName := fileName
		shouldPass := shouldPass
		t.Run(fileName, func(t *testing.T) {
			t.Parallel()

			raw := readProtocolFixture(t, fileName)

			var envelope Envelope
			err := json.Unmarshal(raw, &envelope)
			if shouldPass && err != nil {
				t.Fatalf("expected valid fixture, got error = %v", err)
			}

			if !shouldPass {
				if err == nil {
					t.Fatalf("expected invalid fixture to fail validation")
				}
				return
			}

			hash, err := ComputeMessageHash(envelope.CanonicalHashInput())
			if err != nil {
				t.Fatalf("ComputeMessageHash() error = %v", err)
			}

			if hash != envelope.Hash {
				t.Fatalf("ComputeMessageHash() = %s, want %s", hash, envelope.Hash)
			}
		})
	}
}

func readProtocolFixture(t *testing.T, fileName string) []byte {
	t.Helper()

	candidates := []string{
		filepath.Join("..", "..", "protocol", "fixtures", fileName),
		filepath.Join("protocol", "fixtures", fileName),
	}

	for _, candidate := range candidates {
		raw, err := os.ReadFile(candidate)
		if err == nil {
			return raw
		}
	}

	t.Fatalf("protocol fixture %s not found in %v", fileName, candidates)
	return nil
}

func TestNodeSignatureMatchesSharedVector(t *testing.T) {
	t.Parallel()

	const expected = "sha256=e94d4c3acf8e7f838218bb8aa3ab6e98f6a78b0755421a81efd1d96497e19411"

	signature, err := BuildNodeSignature("{}", "secret-a")
	if err != nil {
		t.Fatalf("BuildNodeSignature() error = %v", err)
	}

	if signature != expected {
		t.Fatalf("BuildNodeSignature() = %s, want %s", signature, expected)
	}

	if !VerifyNodeSignature("{}", signature, "secret-a") {
		t.Fatalf("VerifyNodeSignature() returned false for valid signature")
	}

	if VerifyNodeSignature("{}", signature, "wrong-secret") {
		t.Fatalf("VerifyNodeSignature() returned true for wrong secret")
	}
}

func TestBuildHashInterestMessageCanonicalizesAliases(t *testing.T) {
	t.Parallel()

	message, err := BuildHashInterestMessage(BuildHashInterestMessageInput{
		BuildMessageInput: BuildMessageInput{
			ID:        "interest-1",
			Timestamp: "2026-04-23T00:00:00.000000Z",
			Sender:    "site-b",
			Payload: NewPayload(
				KV("sku", "ABC-1"),
				KV("quantity", 10),
				KV("price", 12.5),
				KV("currency", "BRL"),
				KV("buyer_url", "https://buyer.example.com"),
			),
		},
	})
	if err != nil {
		t.Fatalf("BuildHashInterestMessage() error = %v", err)
	}

	if _, ok := message.Payload.Get("qty"); !ok {
		t.Fatalf("expected canonical qty field")
	}

	if _, ok := message.Payload.Get("unit_price"); !ok {
		t.Fatalf("expected canonical unit_price field")
	}

	if _, ok := message.Payload.Get("quantity"); ok {
		t.Fatalf("did not expect quantity alias in built payload")
	}

	if _, ok := message.Payload.Get("price"); ok {
		t.Fatalf("did not expect price alias in built payload")
	}

	gotJSON, err := message.CanonicalHashJSON()
	if err != nil {
		t.Fatalf("CanonicalHashJSON() error = %v", err)
	}

	wantJSON := `{"type":"HASH_INTEREST","payload":{"sku":"ABC-1","qty":10,"unit_price":12.5,"currency":"BRL","buyer_url":"https://buyer.example.com"},"previousHash":null,"timestamp":"2026-04-23T00:00:00.000000Z"}`
	if gotJSON != wantJSON {
		t.Fatalf("CanonicalHashJSON() = %s, want %s", gotJSON, wantJSON)
	}
}

func TestEnvelopeUnmarshalAcceptsAliases(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"id":"msg-1","negotiation_id":"neg-1","type":"HASH_COUNTER","payload":{"sku":"ABC-1","quantity":10,"price":12.5},"hash":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","prev_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","timestamp":"2026-04-23T00:00:00.000000Z","sender":"site-b"}`)

	var envelope Envelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if envelope.PreviousHash == nil || *envelope.PreviousHash != strings.Repeat("a", 64) {
		t.Fatalf("unexpected previous hash: %#v", envelope.PreviousHash)
	}

	if qty, ok := payloadNumericValue(envelope.Payload, "qty", "quantity"); !ok || qty != 10 {
		t.Fatalf("unexpected quantity parsed from aliases")
	}

	if unitPrice, ok := payloadNumericValue(envelope.Payload, "unit_price", "price"); !ok || unitPrice != 12.5 {
		t.Fatalf("unexpected price parsed from aliases")
	}
}

func TestEnvelopeMarshalJSONDoesNotEscapePaymentURL(t *testing.T) {
	t.Parallel()

	previousHash := strings.Repeat("a", 64)

	envelope, err := BuildHashConcludedMessage(BuildHashConcludedMessageInput{
		BuildMessageInput: BuildMessageInput{
			ID:        "msg-1",
			Timestamp: "2026-04-23T05:41:00.000000Z",
			Sender:    "produtos",
			Payload: NewPayload(
				KV("item", "Patch Cord Cat6 1.5m"),
				KV("sku", "TC-PC6-150"),
				KV("qty", 10),
				KV("unit_price", 12.9),
				KV("currency", "BRL"),
				KV("accepted_by", "supplier"),
				KV("payment_url", "https://produtos.example.com/checkout/order-pay/1234/?pay_for_order=true&key=wc_order_abc123"),
			),
		},
		NegotiationID: "neg-payment-123",
		PreviousHash:  previousHash,
	})
	if err != nil {
		t.Fatalf("BuildHashConcludedMessage() error = %v", err)
	}

	encoded, err := envelope.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	if strings.Contains(string(encoded), `\u0026`) {
		t.Fatalf("MarshalJSON() escaped payment_url: %s", encoded)
	}
}
