package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendMemory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MEMORY.md")
	os.WriteFile(path, []byte("# Memory\n"), 0o644)

	store := NewStore(path)
	err := store.Append([]string{"User prefers Go", "User uses Neovim"})
	if err != nil {
		t.Fatalf("Append() error: %v", err)
	}

	content, _ := store.Read()
	if !strings.Contains(content, "- User prefers Go") {
		t.Error("memory should contain 'User prefers Go'")
	}
	if !strings.Contains(content, "- User uses Neovim") {
		t.Error("memory should contain 'User uses Neovim'")
	}
}

func TestReadMemory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MEMORY.md")

	// Test reading non-existent file
	store := NewStore(path)
	content, err := store.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if content != "# Memory\n" {
		t.Errorf("Read() = %q, want default header", content)
	}
}

func TestSearchMemory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MEMORY.md")
	os.WriteFile(path, []byte("# Memory\n\n## 2026-02-16\n- User prefers Go\n- User likes coffee\n- User uses Neovim\n"), 0o644)

	store := NewStore(path)

	matches, err := store.Search("user")
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(matches) != 3 {
		t.Errorf("got %d matches, want 3", len(matches))
	}

	matches, err = store.Search("neovim")
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(matches) != 1 {
		t.Errorf("got %d matches for 'neovim', want 1", len(matches))
	}

	matches, err = store.Search("python")
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("got %d matches for 'python', want 0", len(matches))
	}
}

func TestMemoryInPrompt_WithinBudget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MEMORY.md")
	os.WriteFile(path, []byte("# Memory\n\n- Fact one\n- Fact two\n- Fact three\n"), 0o644)

	store := NewStore(path)
	content, err := store.ForPrompt(1000) // plenty of budget
	if err != nil {
		t.Fatalf("ForPrompt() error: %v", err)
	}
	if !strings.Contains(content, "Fact one") || !strings.Contains(content, "Fact three") {
		t.Error("should include all facts when within budget")
	}
}

func TestMemoryInPrompt_ExceedsBudget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MEMORY.md")

	// Create a lot of entries
	var builder strings.Builder
	builder.WriteString("# Memory\n\n")
	for i := 0; i < 100; i++ {
		builder.WriteString("- This is a somewhat long memory entry that takes up space in the prompt context window\n")
	}
	os.WriteFile(path, []byte(builder.String()), 0o644)

	store := NewStore(path)
	content, err := store.ForPrompt(50) // very small budget (200 chars)
	if err != nil {
		t.Fatalf("ForPrompt() error: %v", err)
	}

	// Should be truncated
	if len(content) > 250 { // 50 tokens * 4 chars + some header slack
		t.Errorf("content too long: %d chars, budget was 50 tokens (200 chars)", len(content))
	}
}

func TestForget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MEMORY.md")
	os.WriteFile(path, []byte("# Memory\n\n## 2026-02-16\n- User prefers dark mode\n- User likes Go\n- User uses dark theme\n"), 0o644)

	store := NewStore(path)
	removed, err := store.Forget("dark")
	if err != nil {
		t.Fatalf("Forget() error: %v", err)
	}
	if removed != 2 {
		t.Errorf("removed %d entries, want 2", removed)
	}

	entries, _ := store.Entries()
	if len(entries) != 1 {
		t.Errorf("got %d entries, want 1", len(entries))
	}
	if entries[0] != "- User likes Go" {
		t.Errorf("remaining entry = %q, want '- User likes Go'", entries[0])
	}
}

func TestManualRemember(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MEMORY.md")
	os.WriteFile(path, []byte("# Memory\n"), 0o644)

	store := NewStore(path)
	err := store.Append([]string{"prefers dark mode"})
	if err != nil {
		t.Fatalf("Append() error: %v", err)
	}

	entries, _ := store.Entries()
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if !strings.Contains(entries[0], "prefers dark mode") {
		t.Errorf("entry = %q, want 'prefers dark mode'", entries[0])
	}
}
