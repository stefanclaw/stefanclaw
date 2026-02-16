package session

import (
	"testing"

	"github.com/stefanclaw/stefanclaw/internal/provider"
)

func TestCreate(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	s, err := store.Create("Test Chat", "qwen3-next")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if s.ID == "" {
		t.Error("session ID should not be empty")
	}
	if s.Title != "Test Chat" {
		t.Errorf("title = %q, want Test Chat", s.Title)
	}
	if s.Model != "qwen3-next" {
		t.Errorf("model = %q, want qwen3-next", s.Model)
	}
}

func TestAppendAndGet(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	s, err := store.Create("Test", "qwen3-next")
	if err != nil {
		t.Fatal(err)
	}

	msgs := []provider.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	for _, msg := range msgs {
		if err := store.Append(s.ID, msg); err != nil {
			t.Fatalf("Append() error: %v", err)
		}
	}

	loaded, err := store.LoadTranscript(s.ID)
	if err != nil {
		t.Fatalf("LoadTranscript() error: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("got %d messages, want 2", len(loaded))
	}
	if loaded[0].Content != "Hello" {
		t.Errorf("msg[0].Content = %q, want Hello", loaded[0].Content)
	}
	if loaded[1].Content != "Hi there!" {
		t.Errorf("msg[1].Content = %q, want Hi there!", loaded[1].Content)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	store.Create("First", "qwen3-next")
	store.Create("Second", "llama3")
	store.Create("Third", "qwen3-next")

	sessions, err := store.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(sessions) != 3 {
		t.Fatalf("got %d sessions, want 3", len(sessions))
	}

	// Should be sorted newest first
	if sessions[0].Title != "Third" {
		t.Errorf("first session = %q, want Third (newest)", sessions[0].Title)
	}
}

func TestCurrentSession(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	// No current session initially
	cur, err := store.Current()
	if err != nil {
		t.Fatalf("Current() error: %v", err)
	}
	if cur != nil {
		t.Error("Current() should be nil initially")
	}

	s, _ := store.Create("Test", "qwen3-next")
	if err := store.SetCurrent(s.ID); err != nil {
		t.Fatalf("SetCurrent() error: %v", err)
	}

	cur, err = store.Current()
	if err != nil {
		t.Fatalf("Current() error: %v", err)
	}
	if cur == nil || cur.ID != s.ID {
		t.Errorf("Current() = %v, want session %s", cur, s.ID)
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	s, _ := store.Create("To Delete", "qwen3-next")
	store.Append(s.ID, provider.Message{Role: "user", Content: "test"})

	if err := store.Delete(s.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := store.Get(s.ID)
	if err == nil {
		t.Error("Get() should fail after delete")
	}
}
