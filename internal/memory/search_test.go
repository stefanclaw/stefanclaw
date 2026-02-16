package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearch_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MEMORY.md")
	os.WriteFile(path, []byte("# Memory\n\n- User likes GO programming\n- User prefers Neovim\n"), 0o644)

	store := NewStore(path)
	matches, err := store.Search("go")
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(matches) != 1 {
		t.Errorf("got %d matches, want 1", len(matches))
	}
}

func TestSearch_EmptyMemory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MEMORY.md")

	store := NewStore(path) // file doesn't exist
	matches, err := store.Search("anything")
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("got %d matches, want 0", len(matches))
	}
}
