package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/stefanclaw/stefanclaw/internal/provider"
)

const compactPrompt = `Summarize this conversation concisely. Capture key topics discussed, decisions made, and important context. Write in third person, past tense. Keep it under 200 words.`

// EstimateTokens approximates token count as chars/4.
func EstimateTokens(messages []provider.Message) int {
	total := 0
	for _, m := range messages {
		total += len(m.Content) / 4
	}
	return total
}

// CompactResult holds the result of compaction.
type CompactResult struct {
	Summary         string
	OriginalCount   int
	RemainingCount  int
	CompactedTokens int
}

// Compact summarizes old messages when the conversation exceeds the token threshold.
// It keeps the most recent `keepRecent` messages and summarizes the rest.
// Returns nil if no compaction is needed.
func Compact(ctx context.Context, p provider.Provider, model string, messages []provider.Message, maxTokens int, keepRecent int) (*CompactResult, []provider.Message, error) {
	tokens := EstimateTokens(messages)
	threshold := int(float64(maxTokens) * 0.8)

	if tokens <= threshold || len(messages) <= keepRecent+1 {
		return nil, messages, nil
	}

	// Split: old messages to summarize, recent to keep
	splitIdx := len(messages) - keepRecent
	if splitIdx < 1 {
		return nil, messages, nil
	}

	oldMessages := messages[:splitIdx]
	recentMessages := messages[splitIdx:]

	// Build transcript of old messages
	var transcript strings.Builder
	for _, m := range oldMessages {
		if m.Role == "summary" {
			transcript.WriteString("[Previous summary]: " + m.Content + "\n")
		} else {
			transcript.WriteString(m.Role + ": " + m.Content + "\n")
		}
	}

	resp, err := p.Chat(ctx, provider.ChatRequest{
		Model: model,
		Messages: []provider.Message{
			{Role: "system", Content: compactPrompt},
			{Role: "user", Content: transcript.String()},
		},
	})
	if err != nil {
		return nil, messages, fmt.Errorf("compacting conversation: %w", err)
	}

	summary := resp.Message.Content

	// Build new message list: summary + recent
	compacted := make([]provider.Message, 0, 1+len(recentMessages))
	compacted = append(compacted, provider.Message{
		Role:    "summary",
		Content: summary,
	})
	compacted = append(compacted, recentMessages...)

	result := &CompactResult{
		Summary:         summary,
		OriginalCount:   len(messages),
		RemainingCount:  len(compacted),
		CompactedTokens: EstimateTokens(oldMessages),
	}

	return result, compacted, nil
}
