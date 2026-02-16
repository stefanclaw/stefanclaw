package session

import (
	"path/filepath"
	"testing"

	"github.com/stefanclaw/stefanclaw/internal/provider"
)

func TestTranscript_WriteAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")

	msgs := []provider.Message{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi!"},
	}

	for _, msg := range msgs {
		if err := AppendMessage(path, msg); err != nil {
			t.Fatalf("AppendMessage() error: %v", err)
		}
	}

	loaded, err := ReadTranscript(path)
	if err != nil {
		t.Fatalf("ReadTranscript() error: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("got %d messages, want 3", len(loaded))
	}
	for i, msg := range msgs {
		if loaded[i].Role != msg.Role || loaded[i].Content != msg.Content {
			t.Errorf("message[%d] = {%s, %s}, want {%s, %s}", i,
				loaded[i].Role, loaded[i].Content, msg.Role, msg.Content)
		}
	}
}

func TestTranscript_AppendToExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")

	AppendMessage(path, provider.Message{Role: "user", Content: "First"})
	AppendMessage(path, provider.Message{Role: "assistant", Content: "Reply"})

	// Append more
	AppendMessage(path, provider.Message{Role: "user", Content: "Second"})

	loaded, err := ReadTranscript(path)
	if err != nil {
		t.Fatalf("ReadTranscript() error: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("got %d messages, want 3", len(loaded))
	}
	if loaded[2].Content != "Second" {
		t.Errorf("last message = %q, want Second", loaded[2].Content)
	}
}

func TestTranscript_ReadMissing(t *testing.T) {
	msgs, err := ReadTranscript("/nonexistent/transcript.jsonl")
	if err != nil {
		t.Fatalf("ReadTranscript() should not error for missing file, got: %v", err)
	}
	if msgs != nil {
		t.Errorf("expected nil messages, got %d", len(msgs))
	}
}
