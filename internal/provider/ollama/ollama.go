package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/stefanclaw/stefanclaw/internal/provider"
)

// OllamaProvider implements the Provider interface for Ollama.
type OllamaProvider struct {
	baseURL string
	client  *http.Client
}

// New creates a new OllamaProvider.
func New(baseURL string) *OllamaProvider {
	return &OllamaProvider{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (o *OllamaProvider) Name() string {
	return "ollama"
}

// ollamaChatRequest is the Ollama API chat request format.
type ollamaChatRequest struct {
	Model    string             `json:"model"`
	Messages []provider.Message `json:"messages"`
	Stream   bool               `json:"stream"`
}

// ollamaChatResponse is a single response/chunk from Ollama's /api/chat.
type ollamaChatResponse struct {
	Model              string           `json:"model"`
	Message            provider.Message `json:"message"`
	Done               bool             `json:"done"`
	TotalDuration      int64            `json:"total_duration"`
	PromptEvalCount    int              `json:"prompt_eval_count"`
	EvalCount          int              `json:"eval_count"`
}

// ollamaModelsResponse is the response from /api/tags.
type ollamaModelsResponse struct {
	Models []ollamaModel `json:"models"`
}

type ollamaModel struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// Chat sends a non-streaming chat request.
func (o *OllamaProvider) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	body := ollamaChatRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   false,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/chat", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &provider.ChatResponse{
		Message: ollamaResp.Message,
		Model:   ollamaResp.Model,
		Usage: provider.Usage{
			PromptTokens:     ollamaResp.PromptEvalCount,
			CompletionTokens: ollamaResp.EvalCount,
			TotalTokens:      ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		},
	}, nil
}

// StreamChat sends a streaming chat request and returns a channel of deltas.
func (o *OllamaProvider) StreamChat(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamDelta, error) {
	body := ollamaChatRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   true,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/chat", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan provider.StreamDelta)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var chunk ollamaChatResponse
			if err := json.Unmarshal(line, &chunk); err != nil {
				ch <- provider.StreamDelta{Err: fmt.Errorf("decoding chunk: %w", err)}
				return
			}

			if chunk.Done {
				ch <- provider.StreamDelta{
					Done: true,
					Usage: &provider.Usage{
						PromptTokens:     chunk.PromptEvalCount,
						CompletionTokens: chunk.EvalCount,
						TotalTokens:      chunk.PromptEvalCount + chunk.EvalCount,
					},
				}
				return
			}

			ch <- provider.StreamDelta{
				Content: chunk.Message.Content,
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case <-ctx.Done():
				// Context cancelled, don't send error
			default:
				ch <- provider.StreamDelta{Err: fmt.Errorf("reading stream: %w", err)}
			}
		}
	}()

	return ch, nil
}

// ListModels returns available models from Ollama.
func (o *OllamaProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("listing models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var modelsResp ollamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("decoding models: %w", err)
	}

	models := make([]provider.ModelInfo, len(modelsResp.Models))
	for i, m := range modelsResp.Models {
		models[i] = provider.ModelInfo{
			Name: m.Name,
			Size: m.Size,
		}
	}
	return models, nil
}

// IsAvailable checks if Ollama is running and reachable.
func (o *OllamaProvider) IsAvailable(ctx context.Context) error {
	return Detect(ctx, o.baseURL)
}
