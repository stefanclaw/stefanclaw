package provider

import "context"

// Provider defines the interface for LLM providers.
type Provider interface {
	Name() string
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamDelta, error)
	ListModels(ctx context.Context) ([]ModelInfo, error)
	IsAvailable(ctx context.Context) error
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the input for a chat completion.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	NumCtx   int       `json:"-"` // Ollama-specific context size, not serialized generically
}

// ChatResponse is the output of a non-streaming chat completion.
type ChatResponse struct {
	Message Message `json:"message"`
	Model   string  `json:"model"`
	Usage   Usage   `json:"usage"`
}

// StreamDelta represents a single streaming chunk.
type StreamDelta struct {
	Content string
	Done    bool
	Usage   *Usage
	Err     error
}

// ModelInfo describes an available model.
type ModelInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// Usage contains token usage statistics.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
