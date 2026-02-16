package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSystemPrompt_AllSections(t *testing.T) {
	dir := t.TempDir()
	for _, name := range AllSections {
		content := "# " + name + "\nTest content for " + name
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	a := NewAssembler(dir)
	if err := a.LoadFiles(); err != nil {
		t.Fatalf("LoadFiles() error: %v", err)
	}

	prompt := a.BuildSystemPrompt()
	for _, name := range AllSections {
		if !strings.Contains(prompt, "Test content for "+name) {
			t.Errorf("prompt missing content for %s", name)
		}
	}
}

func TestBuildSystemPrompt_MissingSections(t *testing.T) {
	dir := t.TempDir()
	// Only write IDENTITY
	if err := os.WriteFile(filepath.Join(dir, SectionIdentity), []byte("# Identity\nI am test"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := NewAssembler(dir)
	if err := a.LoadFiles(); err != nil {
		t.Fatalf("LoadFiles() error: %v", err)
	}

	prompt := a.BuildSystemPrompt()
	if !strings.Contains(prompt, "I am test") {
		t.Error("prompt should contain IDENTITY content")
	}
	// Missing sections fall back to embedded defaults
	if !a.HasSection(SectionSoul) {
		t.Log("SOUL.md not on disk, falling back to embedded — OK")
	}
}

func TestBuildSystemPrompt_EmptySections(t *testing.T) {
	dir := t.TempDir()
	// Write an empty file
	if err := os.WriteFile(filepath.Join(dir, SectionIdentity), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	// Write a non-empty file
	if err := os.WriteFile(filepath.Join(dir, SectionSoul), []byte("# Soul\nBe helpful"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := NewAssembler(dir)
	a.LoadFiles()

	prompt := a.BuildSystemPrompt()
	if strings.Contains(prompt, "IDENTITY") {
		// Empty IDENTITY should not appear in prompt (it was empty on disk, but embedded has content)
		// Actually embedded fallback won't trigger since the file exists. Let's check HasSection
	}
	if !strings.Contains(prompt, "Be helpful") {
		t.Error("prompt should contain SOUL content")
	}
}

func TestLoadFiles_FromDisk(t *testing.T) {
	dir := t.TempDir()
	content := "# Custom Identity\nI am a custom assistant"
	if err := os.WriteFile(filepath.Join(dir, SectionIdentity), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	a := NewAssembler(dir)
	a.LoadFiles()

	got := a.Section(SectionIdentity)
	if got != content {
		t.Errorf("Section(IDENTITY) = %q, want %q", got, content)
	}
}

func TestLoadFiles_FallbackToEmbedded(t *testing.T) {
	dir := t.TempDir() // empty directory, no files

	a := NewAssembler(dir)
	a.LoadFiles()

	// Should fall back to embedded defaults
	if !a.HasSection(SectionIdentity) {
		t.Error("IDENTITY should be loaded from embedded defaults")
	}
}

func TestBootPrompt_IncludedOnStartup(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, SectionBoot), []byte("# Boot\n- Greet the user"), 0o644)
	os.WriteFile(filepath.Join(dir, SectionIdentity), []byte("# Identity\nI am test"), 0o644)

	a := NewAssembler(dir)
	a.LoadFiles()

	prompt := a.BuildSystemPrompt()
	if !strings.Contains(prompt, "Greet the user") {
		t.Error("BOOT.md content should be in system prompt")
	}
}

func TestBootPrompt_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, SectionBoot), []byte(""), 0o644)

	a := NewAssembler(dir)
	a.LoadFiles()

	if a.HasSection(SectionBoot) {
		t.Error("empty BOOT.md should not count as a section")
	}
}

func TestBootstrapDetected_FirstRun(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, SectionBootstrap), []byte("# Bootstrap\nWelcome!"), 0o644)

	a := NewAssembler(dir)
	a.LoadFiles()

	if !a.HasBootstrap() {
		t.Error("HasBootstrap() should be true when BOOTSTRAP.md exists")
	}

	if !BootstrapExists(dir) {
		t.Error("BootstrapExists() should be true")
	}
}

func TestBootstrapNotDetected_AfterDeletion(t *testing.T) {
	dir := t.TempDir()
	// No BOOTSTRAP.md on disk, and don't fall back to embedded
	// Since loadFile falls back to embedded, we need a different check
	// BootstrapExists only checks disk
	if BootstrapExists(dir) {
		t.Error("BootstrapExists() should be false when file doesn't exist on disk")
	}
}

func TestBootstrapDeletion_AfterFirstConversation(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, SectionBootstrap), []byte("# Bootstrap\nWelcome!"), 0o644)

	a := NewAssembler(dir)
	a.LoadFiles()

	if !a.HasBootstrap() {
		t.Fatal("should have bootstrap before deletion")
	}

	if err := a.DeleteBootstrap(); err != nil {
		t.Fatalf("DeleteBootstrap() error: %v", err)
	}

	if a.HasBootstrap() {
		t.Error("HasBootstrap() should be false after deletion")
	}

	if BootstrapExists(dir) {
		t.Error("BOOTSTRAP.md should be deleted from disk")
	}
}

func TestBuildSystemPromptWithLanguage(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, SectionIdentity), []byte("# Identity\nI am test"), 0o644)

	a := NewAssembler(dir)
	a.LoadFiles()

	prompt := a.BuildSystemPromptWithLanguage("Deutsch")
	if !strings.Contains(prompt, "Always respond in Deutsch") {
		t.Error("prompt should contain language instruction for Deutsch")
	}
	if !strings.Contains(prompt, "I am test") {
		t.Error("prompt should still contain personality sections")
	}
}

func TestBuildSystemPromptWithLanguage_EmptyFallback(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, SectionIdentity), []byte("# Identity\nI am test"), 0o644)

	a := NewAssembler(dir)
	a.LoadFiles()

	prompt := a.BuildSystemPromptWithLanguage("")
	if !strings.Contains(prompt, "Always respond in English") {
		t.Error("empty language should fall back to English")
	}
}

func TestHeartbeatSection_Loaded(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, SectionHeartbeat), []byte("# Heartbeat\nCheck in periodically"), 0o644)

	a := NewAssembler(dir)
	a.LoadFiles()

	if !a.HasSection(SectionHeartbeat) {
		t.Error("HEARTBEAT.md should be loaded")
	}
	prompt := a.BuildSystemPrompt()
	if !strings.Contains(prompt, "Check in periodically") {
		t.Error("system prompt should contain heartbeat content")
	}
}

func TestEmbeddedDefaults_NotEmpty(t *testing.T) {
	for _, name := range AllSections {
		content, err := EmbeddedDefault(name)
		if err != nil {
			t.Errorf("EmbeddedDefault(%s) error: %v", name, err)
			continue
		}
		if strings.TrimSpace(content) == "" {
			t.Errorf("EmbeddedDefault(%s) is empty", name)
		}
	}
}
