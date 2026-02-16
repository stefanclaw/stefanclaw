package session

import (
	"context"
	"strings"
	"testing"

	"github.com/stefanclaw/stefanclaw/internal/provider"
)

type mockProvider struct {
	chatResp *provider.ChatResponse
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Chat(_ context.Context, _ provider.ChatRequest) (*provider.ChatResponse, error) {
	return m.chatResp, nil
}
func (m *mockProvider) StreamChat(_ context.Context, _ provider.ChatRequest) (<-chan provider.StreamDelta, error) {
	return nil, nil
}
func (m *mockProvider) ListModels(_ context.Context) ([]provider.ModelInfo, error) {
	return nil, nil
}
func (m *mockProvider) IsAvailable(_ context.Context) error { return nil }

func TestCompact_ShortConversation_NoChange(t *testing.T) {
	messages := []provider.Message{
		{Role: "user", Content: "Hi"},
		{Role: "assistant", Content: "Hello!"},
	}

	mp := &mockProvider{}
	result, compacted, err := Compact(context.Background(), mp, "test", messages, 10000, 4)
	if err != nil {
		t.Fatalf("Compact() error: %v", err)
	}
	if result != nil {
		t.Error("should not compact short conversation")
	}
	if len(compacted) != 2 {
		t.Errorf("got %d messages, want 2", len(compacted))
	}
}

func TestCompact_LongConversation_Summarized(t *testing.T) {
	// Create a long conversation that exceeds threshold
	var messages []provider.Message
	for i := 0; i < 20; i++ {
		messages = append(messages,
			provider.Message{Role: "user", Content: strings.Repeat("question about topic ", 20)},
			provider.Message{Role: "assistant", Content: strings.Repeat("detailed answer about topic ", 20)},
		)
	}

	mp := &mockProvider{
		chatResp: &provider.ChatResponse{
			Message: provider.Message{
				Role:    "assistant",
				Content: "The user discussed various topics with the assistant.",
			},
		},
	}

	// Set a low maxTokens to force compaction
	result, _, err := Compact(context.Background(), mp, "test", messages, 500, 4)
	if err != nil {
		t.Fatalf("Compact() error: %v", err)
	}
	if result == nil {
		t.Fatal("should have compacted long conversation")
	}
	if result.OriginalCount != 40 {
		t.Errorf("OriginalCount = %d, want 40", result.OriginalCount)
	}
}

func TestCompact_PreservesRecentMessages(t *testing.T) {
	var messages []provider.Message
	for i := 0; i < 20; i++ {
		messages = append(messages,
			provider.Message{Role: "user", Content: strings.Repeat("long question ", 50)},
			provider.Message{Role: "assistant", Content: strings.Repeat("long answer ", 50)},
		)
	}
	// Mark the last user message
	messages[len(messages)-2].Content = "FINAL_USER_MESSAGE"
	messages[len(messages)-1].Content = "FINAL_ASSISTANT_MESSAGE"

	mp := &mockProvider{
		chatResp: &provider.ChatResponse{
			Message: provider.Message{
				Role:    "assistant",
				Content: "Summary of conversation.",
			},
		},
	}

	_, compacted, err := Compact(context.Background(), mp, "test", messages, 500, 4)
	if err != nil {
		t.Fatalf("Compact() error: %v", err)
	}

	// Should have summary + 4 recent messages
	if len(compacted) != 5 {
		t.Fatalf("got %d messages, want 5 (1 summary + 4 recent)", len(compacted))
	}
	if compacted[0].Role != "summary" {
		t.Errorf("first message role = %q, want summary", compacted[0].Role)
	}

	// Check that recent messages are preserved
	found := false
	for _, m := range compacted {
		if m.Content == "FINAL_USER_MESSAGE" || m.Content == "FINAL_ASSISTANT_MESSAGE" {
			found = true
		}
	}
	if !found {
		t.Error("recent messages should be preserved")
	}
}

func TestCompact_SummaryFormat(t *testing.T) {
	var messages []provider.Message
	for i := 0; i < 20; i++ {
		messages = append(messages,
			provider.Message{Role: "user", Content: strings.Repeat("x", 200)},
			provider.Message{Role: "assistant", Content: strings.Repeat("y", 200)},
		)
	}

	mp := &mockProvider{
		chatResp: &provider.ChatResponse{
			Message: provider.Message{
				Role:    "assistant",
				Content: "The conversation covered Go programming and project setup.",
			},
		},
	}

	result, _, err := Compact(context.Background(), mp, "test", messages, 500, 4)
	if err != nil {
		t.Fatalf("Compact() error: %v", err)
	}
	if result == nil {
		t.Fatal("expected compaction")
	}
	if result.Summary == "" {
		t.Error("summary should not be empty")
	}
}

func TestTokenEstimate(t *testing.T) {
	messages := []provider.Message{
		{Role: "user", Content: "Hello world"},       // 11 chars = ~2 tokens
		{Role: "assistant", Content: "Hi there man"},  // 12 chars = ~3 tokens
	}
	tokens := EstimateTokens(messages)
	if tokens < 2 || tokens > 10 {
		t.Errorf("EstimateTokens = %d, expected roughly 5", tokens)
	}
}
