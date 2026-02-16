package memory

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Store manages the MEMORY.md file.
type Store struct {
	path string
}

// NewStore creates a new memory store for the given MEMORY.md path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Read returns the full content of MEMORY.md.
func (s *Store) Read() (string, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return "# Memory\n", nil
		}
		return "", err
	}
	return string(data), nil
}

// Append adds facts to MEMORY.md under a dated section.
func (s *Store) Append(facts []string) error {
	if len(facts) == 0 {
		return nil
	}

	existing, err := s.Read()
	if err != nil {
		return err
	}

	dateHeader := fmt.Sprintf("\n## %s\n", time.Now().Format("2006-01-02"))

	var builder strings.Builder
	builder.WriteString(existing)

	// Check if today's section already exists
	if !strings.Contains(existing, dateHeader) {
		builder.WriteString(dateHeader)
	}

	for _, fact := range facts {
		fact = strings.TrimSpace(fact)
		if fact == "" {
			continue
		}
		// Ensure bullet point format
		if !strings.HasPrefix(fact, "- ") {
			fact = "- " + fact
		}
		builder.WriteString(fact + "\n")
	}

	return os.WriteFile(s.path, []byte(builder.String()), 0o644)
}

// Forget removes all lines containing the keyword.
func (s *Store) Forget(keyword string) (int, error) {
	content, err := s.Read()
	if err != nil {
		return 0, err
	}

	keyword = strings.ToLower(keyword)
	lines := strings.Split(content, "\n")
	var kept []string
	removed := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "- ") && strings.Contains(strings.ToLower(line), keyword) {
			removed++
			continue
		}
		kept = append(kept, line)
	}

	if removed == 0 {
		return 0, nil
	}

	return removed, os.WriteFile(s.path, []byte(strings.Join(kept, "\n")), 0o644)
}

// Entries returns all memory entries (bullet points).
func (s *Store) Entries() ([]string, error) {
	content, err := s.Read()
	if err != nil {
		return nil, err
	}

	var entries []string
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "- ") {
			entries = append(entries, line)
		}
	}
	return entries, nil
}

// ForPrompt returns memory content trimmed to fit within the token budget.
// Approximates tokens as chars/4.
func (s *Store) ForPrompt(maxTokens int) (string, error) {
	entries, err := s.Entries()
	if err != nil {
		return "", err
	}

	if len(entries) == 0 {
		return "", nil
	}

	maxChars := maxTokens * 4
	var result strings.Builder
	result.WriteString("# Memory\n\n")

	for _, entry := range entries {
		if result.Len()+len(entry)+1 > maxChars {
			break
		}
		result.WriteString(entry + "\n")
	}

	return result.String(), nil
}
