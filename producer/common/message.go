package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

// Message is the owned, lossless envelope used between producer adapters.
type Message struct {
	Backend string
	Topic   string
	Key     []byte
	Value   []byte
}

func ValidateMessage(m Message) error {
	if m.Topic == "" || len(m.Topic) > 249 || m.Topic == "." || m.Topic == ".." {
		return errors.New("producer: invalid topic")
	}
	for _, c := range m.Topic {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("._-", c)) {
			return errors.New("producer: invalid topic")
		}
	}
	return nil
}
func Encode(m Message) ([]byte, error) {
	if err := ValidateMessage(m); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

// Decode accepts historical envelopes for Input callers; new callers use JSON or Submit.
func Decode(b []byte) (Message, error) {
	var m Message
	if bytes.HasPrefix(bytes.TrimSpace(b), []byte("{")) {
		if err := json.Unmarshal(b, &m); err != nil {
			return m, err
		}
	} else {
		parts := bytes.SplitN(b, []byte(" "), 5)
		if len(parts) != 5 || len(parts[1]) == 0 {
			return m, errors.New("producer: malformed legacy envelope")
		}
		m = Message{Backend: string(parts[1]), Topic: string(parts[2]), Key: parts[3], Value: parts[4]}
	}
	if m.Backend == "" {
		m.Backend = "kafka"
	}
	return m, ValidateMessage(m)
}
