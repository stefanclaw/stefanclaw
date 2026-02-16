package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/stefanclaw/stefanclaw/internal/provider"
)

// AppendMessage appends a single message as a JSONL line.
func AppendMessage(path string, msg provider.Message) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("opening transcript: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}

	return nil
}

// ReadTranscript reads all messages from a JSONL transcript file.
func ReadTranscript(path string) ([]provider.Message, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening transcript: %w", err)
	}
	defer f.Close()

	var messages []provider.Message
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg provider.Message
		if err := json.Unmarshal(line, &msg); err != nil {
			return nil, fmt.Errorf("decoding message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, scanner.Err()
}
