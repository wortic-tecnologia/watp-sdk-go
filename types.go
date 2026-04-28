package watp

import "fmt"

type MessageType string

const (
	MessageTypeHashInterest     MessageType = "HASH_INTEREST"
	MessageTypeHashProposal     MessageType = "HASH_PROPOSAL"
	MessageTypeHashCounter      MessageType = "HASH_COUNTER"
	MessageTypeHashConcluded    MessageType = "HASH_CONCLUDED"
	MessageTypeHashCancelled    MessageType = "HASH_CANCELLED"
	MessageTypeHashWaitingHuman MessageType = "HASH_WAITING_HUMAN"
)

func (m MessageType) String() string {
	return string(m)
}

func (m MessageType) IsValid() bool {
	switch m {
	case MessageTypeHashInterest,
		MessageTypeHashProposal,
		MessageTypeHashCounter,
		MessageTypeHashConcluded,
		MessageTypeHashCancelled,
		MessageTypeHashWaitingHuman:
		return true
	default:
		return false
	}
}

func ParseMessageType(value string) (MessageType, error) {
	messageType := MessageType(value)
	if !messageType.IsValid() {
		return "", fmt.Errorf("unsupported WATP message type %q", value)
	}

	return messageType, nil
}

type CanonicalHashInput struct {
	Type         MessageType
	Payload      Payload
	PreviousHash *string
	Timestamp    string
}

type Envelope struct {
	ID            string
	NegotiationID string
	Type          MessageType
	Payload       Payload
	Hash          string
	PreviousHash  *string
	Timestamp     string
	Sender        string
	Signature     *string
}

type BuildMessageInput struct {
	ID        string
	Timestamp string
	Sender    string
	Signature string
	Payload   Payload
}

type BuildHashInterestMessageInput struct {
	BuildMessageInput
	NegotiationID string
	PreviousHash  string
}

type BuildChainedMessageInput struct {
	BuildMessageInput
	NegotiationID string
	PreviousHash  string
}

type BuildHashCounterMessageInput = BuildChainedMessageInput
type BuildHashConcludedMessageInput = BuildChainedMessageInput
type BuildHashCancelledMessageInput = BuildChainedMessageInput
