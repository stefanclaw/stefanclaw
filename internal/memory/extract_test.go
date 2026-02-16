package memory

import (
	"context"
	"testing"

	"github.com/stefanclaw/stefanclaw/internal/provider"
)

type mockProvider struct {
	resp *provider.ChatResponse
	err  error
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Chat(_ context.Context, _ provider.ChatRequest) (*provider.ChatResponse, error) {
	return m.resp, m.err
}
func (m *mockProvider) StreamChat(_ context.Context, _ provider.ChatRequest) (<-chan provider.StreamDelta, error) {
	return nil, nil
}
func (m *mockProvider) ListModels(_ context.Context) ([]provider.ModelInfo, error) {
	return nil, nil
}
func (m *mockProvider) IsAvailable(_ context.Context) error { return nil }

func TestExtractFacts_MockLLM(t *testing.T) {
	mp := &mockProvider{
		resp: &provider.ChatResponse{
			Message: provider.Message{
				Role: "assistant",
				Content: `- User prefers concise responses
- User works with Go
- User's name is Stefan`,
			},
		},
	}

	extractor := NewExtractor(mp, "test-model")
	facts, err := extractor.Extract(context.Background(), []provider.Message{
		{Role: "user", Content: "I'm Stefan and I work with Go"},
		{Role: "assistant", Content: "Nice to meet you!"},
	})
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}

	if len(facts) != 3 {
		t.Fatalf("got %d facts, want 3", len(facts))
	}
	if facts[0] != "- User prefers concise responses" {
		t.Errorf("fact[0] = %q", facts[0])
	}
}

func TestExtractFacts_NoneWorthRemembering(t *testing.T) {
	mp := &mockProvider{
		resp: &provider.ChatResponse{
			Message: provider.Message{
				Role:    "assistant",
				Content: "NONE",
			},
		},
	}

	extractor := NewExtractor(mp, "test-model")
	facts, err := extractor.Extract(context.Background(), []provider.Message{
		{Role: "user", Content: "What time is it?"},
		{Role: "assistant", Content: "I don't have access to the current time."},
	})
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}
	if len(facts) != 0 {
		t.Errorf("got %d facts, want 0", len(facts))
	}
}
