package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"

	"github.com/watp/sdk-go"
)

func main() {
	interest, err := watp.BuildHashInterestMessage(watp.BuildHashInterestMessageInput{
		BuildMessageInput: watp.BuildMessageInput{
			Sender: "site-b",
			Payload: watp.NewPayload(
				watp.KV("item", "Patch Cord Cat6 1.5m"),
				watp.KV("sku", "TC-PC6-150"),
				watp.KV("qty", 10),
				watp.KV("unit_price", 12.9),
				watp.KV("currency", "BRL"),
				watp.KV("buyer", "site-b"),
				watp.KV("buyer_url", "https://site-b.wortic.com"),
			),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	concluded, err := watp.BuildHashConcludedMessage(watp.BuildHashConcludedMessageInput{
		BuildMessageInput: watp.BuildMessageInput{
			Sender: "produtos",
			Payload: watp.NewPayload(
				watp.KV("item", "Patch Cord Cat6 1.5m"),
				watp.KV("sku", "TC-PC6-150"),
				watp.KV("qty", 10),
				watp.KV("unit_price", 12.9),
				watp.KV("currency", "BRL"),
				watp.KV("accepted_by", "supplier"),
				watp.KV("payment_url", "https://produtos.example.com/checkout/order-pay/1234/?pay_for_order=true&key=wc_order_abc123"),
			),
		},
		NegotiationID: "neg-payment-123",
		PreviousHash:  interest.Hash,
	})
	if err != nil {
		log.Fatal(err)
	}

	interestJSON, err := prettyJSON(interest)
	if err != nil {
		log.Fatal(err)
	}

	concludedJSON, err := prettyJSON(concluded)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(interestJSON))
	fmt.Println(string(concludedJSON))
}

func prettyJSON(envelope watp.Envelope) ([]byte, error) {
	raw, err := envelope.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var formatted bytes.Buffer
	if err := json.Indent(&formatted, raw, "", "  "); err != nil {
		return nil, err
	}

	return formatted.Bytes(), nil
}
