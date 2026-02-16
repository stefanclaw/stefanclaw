package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Section names for personality files.
const (
	SectionIdentity  = "IDENTITY.md"
	SectionSoul      = "SOUL.md"
	SectionUser      = "USER.md"
	SectionMemory    = "MEMORY.md"
	SectionBoot      = "BOOT.md"
	SectionHeartbeat = "HEARTBEAT.md"
	SectionBootstrap = "BOOTSTRAP.md"
)

// AllSections lists the personality files in prompt assembly order.
var AllSections = []string{
	SectionIdentity,
	SectionSoul,
	SectionUser,
	SectionMemory,
	SectionBoot,
	SectionHeartbeat,
	SectionBootstrap,
}

// Assembler loads personality files and builds a system prompt.
type Assembler struct {
	personalityDir string
	sections       map[string]string
}

// NewAssembler creates an Assembler that reads from the given personality directory.
func NewAssembler(personalityDir string) *Assembler {
	return &Assembler{
		personalityDir: personalityDir,
		sections:       make(map[string]string),
	}
}

// LoadFiles reads personality files from disk, falling back to embedded defaults.
func (a *Assembler) LoadFiles() error {
	for _, name := range AllSections {
		content, err := a.loadFile(name)
		if err != nil {
			continue // skip missing sections
		}
		a.sections[name] = content
	}
	return nil
}

func (a *Assembler) loadFile(name string) (string, error) {
	// Try disk first
	diskPath := filepath.Join(a.personalityDir, name)
	data, err := os.ReadFile(diskPath)
	if err == nil {
		return string(data), nil
	}

	// Fall back to embedded
	data, err = embeddedFS.ReadFile(filepath.Join(defaultsDir, name))
	if err != nil {
		return "", fmt.Errorf("section %s not found: %w", name, err)
	}
	return string(data), nil
}

// BuildSystemPrompt assembles all loaded sections into a single system prompt.
func (a *Assembler) BuildSystemPrompt() string {
	var parts []string
	for _, name := range AllSections {
		content, ok := a.sections[name]
		if !ok || strings.TrimSpace(content) == "" {
			continue
		}
		parts = append(parts, strings.TrimSpace(content))
	}
	return strings.Join(parts, "\n\n---\n\n")
}

// BuildSystemPromptWithLanguage assembles the system prompt and prepends a
// language instruction so the LLM responds in the user's preferred language.
func (a *Assembler) BuildSystemPromptWithLanguage(language string) string {
	base := a.BuildSystemPrompt()
	if language == "" {
		language = "English"
	}
	instruction := fmt.Sprintf("IMPORTANT: Always respond in %s. All your messages, questions, and responses must be in %s.", language, language)
	return instruction + "\n\n---\n\n" + base
}

// HasSection returns true if the named section was loaded and is non-empty.
func (a *Assembler) HasSection(name string) bool {
	content, ok := a.sections[name]
	return ok && strings.TrimSpace(content) != ""
}

// Section returns the content of a named section.
func (a *Assembler) Section(name string) string {
	return a.sections[name]
}

// HasBootstrap returns true if BOOTSTRAP.md exists on disk (first-run ritual pending).
func (a *Assembler) HasBootstrap() bool {
	return BootstrapExists(a.personalityDir)
}

// DeleteBootstrap removes BOOTSTRAP.md from disk (after first conversation).
func (a *Assembler) DeleteBootstrap() error {
	path := filepath.Join(a.personalityDir, SectionBootstrap)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	delete(a.sections, SectionBootstrap)
	return nil
}

// BootstrapExists checks if BOOTSTRAP.md exists on disk.
func BootstrapExists(personalityDir string) bool {
	_, err := os.Stat(filepath.Join(personalityDir, SectionBootstrap))
	return err == nil
}

// EmbeddedDefault returns the embedded default content for a section.
func EmbeddedDefault(name string) (string, error) {
	data, err := embeddedFS.ReadFile(filepath.Join(defaultsDir, name))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
