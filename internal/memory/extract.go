package memory

import (
	"context"
	"strings"

	"github.com/stefanclaw/stefanclaw/internal/provider"
)

const extractPrompt = `Extract key facts, preferences, and decisions from this conversation as bullet points.
Only include information worth remembering for future conversations.
Return ONLY bullet points, one per line, starting with "- ".
If there's nothing worth remembering, return "NONE".`

// Extractor uses an LLM to extract memorable facts from conversations.
type Extractor struct {
	provider provider.Provider
	model    string
}

// NewExtractor creates a new fact extractor.
func NewExtractor(p provider.Provider, model string) *Extractor {
	return &Extractor{provider: p, model: model}
}

// Extract asks the LLM to extract key facts from a conversation transcript.
func (e *Extractor) Extract(ctx context.Context, messages []provider.Message) ([]string, error) {
	// Build a transcript summary for the LLM
	var transcript strings.Builder
	for _, msg := range messages {
		if msg.Role == "system" {
			continue
		}
		transcript.WriteString(msg.Role + ": " + msg.Content + "\n")
	}

	resp, err := e.provider.Chat(ctx, provider.ChatRequest{
		Model: e.model,
		Messages: []provider.Message{
			{Role: "system", Content: extractPrompt},
			{Role: "user", Content: transcript.String()},
		},
	})
	if err != nil {
		return nil, err
	}

	return parseFacts(resp.Message.Content), nil
}

func parseFacts(content string) []string {
	content = strings.TrimSpace(content)
	if content == "NONE" || content == "" {
		return nil
	}

	var facts []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			facts = append(facts, line)
		}
	}
	return facts
}
