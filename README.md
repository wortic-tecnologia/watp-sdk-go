# WATP SDK for Go

Initial Go scaffold for the current real WATP contract.

This package is aligned with the contract documented in `protocol/` and with the current TypeScript and Python scaffolds. It covers the frozen minimum surface for the current runtime:

- envelope types for the real message contract
- SHA-256 hash helper compatible with the current runtime
- builders for `HASH_INTEREST`, `HASH_COUNTER`, `HASH_CONCLUDED`, and `HASH_CANCELLED`
- an executable example in `examples/minimal`

## Scope

The scaffold follows the current practical rules from the protocol snapshot:

- chained messages require `negotiation_id`
- chained messages require `previousHash`
- new builders emit canonical field names `qty`, `unit_price`, and `previousHash`
- envelope parsing accepts `prev_hash`, `quantity`, and `price`
- `payment_url` is optional and additive on `HASH_CONCLUDED`
- `signature` is treated as opaque

## Ordered payloads and hash compatibility

The current WATP runtime hashes the exact JSON object:

```json
{
  "type": "HASH_INTEREST",
  "payload": {
    "sku": "ABC-1",
    "qty": 10,
    "unit_price": 12.5
  },
  "previousHash": null,
  "timestamp": "2026-04-23T00:00:00.000000Z"
}
```

To stay compatible, this SDK keeps payload field order through `watp.Payload` instead of using a plain Go map for hashing and JSON emission.

Use `watp.NewPayload(...)` and, for nested JSON objects that must stay ordered, use another `watp.Payload` as the field value.

## Local validation

```bash
cd /Applications/XAMPP/xamppfiles/htdocs/watp/sdk/go
go test ./...
```

## Minimal example

```bash
cd /Applications/XAMPP/xamppfiles/htdocs/watp/sdk/go
go run ./examples/minimal
```

## Quick usage

```go
package main

import (
    "fmt"

    "github.com/watp/sdk-go"
)

func main() {
    interest, err := watp.BuildHashInterestMessage(watp.BuildHashInterestMessageInput{
        BuildMessageInput: watp.BuildMessageInput{
            Sender: "site-b",
            Payload: watp.NewPayload(
                watp.KV("sku", "TC-PC6-150"),
                watp.KV("qty", 10),
                watp.KV("unit_price", 12.9),
                watp.KV("currency", "BRL"),
                watp.KV("buyer_url", "https://site-b.wortic.com"),
            ),
        },
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(interest.Hash)
}
```

## Exposed API

- `MessageType`
- `Envelope`
- `Payload`
- `Field`
- `KV()`
- `UTCTimestamp()`
- `SHA256Hex()`
- `IsHexSHA256()`
- `CreateCanonicalMessageBody()`
- `ComputeMessageHash()`
- `BuildHashInterestMessage()`
- `BuildHashCounterMessage()`
- `BuildHashConcludedMessage()`
- `BuildHashCancelledMessage()`

## Deliberate omissions

This initial scaffold does not include:

- HTTP client helpers
- Bearer or HMAC transport helpers
- cryptographic verification of `signature`
- registry or discovery support
- builders for `HASH_PROPOSAL` or `HASH_WAITING_HUMAN`
