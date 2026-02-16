package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stefanclaw/stefanclaw/internal/provider"
)

func TestListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ollamaModelsResponse{
			Models: []ollamaModel{
				{Name: "qwen3-next", Size: 4000000000},
				{Name: "llama3", Size: 8000000000},
			},
		})
	}))
	defer srv.Close()

	p := New(srv.URL)
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("got %d models, want 2", len(models))
	}
	if models[0].Name != "qwen3-next" {
		t.Errorf("models[0].Name = %q, want qwen3-next", models[0].Name)
	}
}

func TestIsAvailable_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"models":[]}`))
	}))
	defer srv.Close()

	p := New(srv.URL)
	if err := p.IsAvailable(context.Background()); err != nil {
		t.Errorf("IsAvailable() error: %v", err)
	}
}

func TestIsAvailable_Failure(t *testing.T) {
	p := New("http://127.0.0.1:1")
	if err := p.IsAvailable(context.Background()); err == nil {
		t.Error("IsAvailable() should return error for unreachable server")
	}
}

func TestChat_SimpleResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Stream {
			t.Error("Chat should set stream=false")
		}

		json.NewEncoder(w).Encode(ollamaChatResponse{
			Model:   "qwen3-next",
			Message: provider.Message{Role: "assistant", Content: "Hello!"},
			Done:    true,
			PromptEvalCount: 10,
			EvalCount:       5,
		})
	}))
	defer srv.Close()

	p := New(srv.URL)
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "qwen3-next",
		Messages: []provider.Message{{Role: "user", Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Message.Content != "Hello!" {
		t.Errorf("response content = %q, want Hello!", resp.Message.Content)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("total tokens = %d, want 15", resp.Usage.TotalTokens)
	}
}

func TestChat_WithSystemPrompt(t *testing.T) {
	var receivedMessages []provider.Message
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaChatRequest
		json.NewDecoder(r.Body).Decode(&req)
		receivedMessages = req.Messages

		json.NewEncoder(w).Encode(ollamaChatResponse{
			Model:   "qwen3-next",
			Message: provider.Message{Role: "assistant", Content: "I understand"},
			Done:    true,
		})
	}))
	defer srv.Close()

	p := New(srv.URL)
	p.Chat(context.Background(), provider.ChatRequest{
		Model: "qwen3-next",
		Messages: []provider.Message{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "Hello"},
		},
	})

	if len(receivedMessages) != 2 {
		t.Fatalf("sent %d messages, want 2", len(receivedMessages))
	}
	if receivedMessages[0].Role != "system" {
		t.Errorf("first message role = %q, want system", receivedMessages[0].Role)
	}
}

func TestChat_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"model not found"}`))
	}))
	defer srv.Close()

	p := New(srv.URL)
	_, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "nonexistent",
		Messages: []provider.Message{{Role: "user", Content: "Hi"}},
	})
	if err == nil {
		t.Error("Chat() should return error for 400 response")
	}
}

func TestChat_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := New(srv.URL)
	_, err := p.Chat(ctx, provider.ChatRequest{
		Model:    "qwen3-next",
		Messages: []provider.Message{{Role: "user", Content: "Hi"}},
	})
	if err == nil {
		t.Error("Chat() should return error on context cancellation")
	}
}

func TestStreamChat_TokenByToken(t *testing.T) {
	tokens := []string{"Hello", " ", "world", "!"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("server does not support flushing")
		}

		for _, tok := range tokens {
			chunk := ollamaChatResponse{
				Model:   "qwen3-next",
				Message: provider.Message{Role: "assistant", Content: tok},
				Done:    false,
			}
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "%s\n", data)
			flusher.Flush()
		}

		// Final chunk
		final := ollamaChatResponse{
			Model:           "qwen3-next",
			Done:            true,
			PromptEvalCount: 10,
			EvalCount:       4,
		}
		data, _ := json.Marshal(final)
		fmt.Fprintf(w, "%s\n", data)
		flusher.Flush()
	}))
	defer srv.Close()

	p := New(srv.URL)
	ch, err := p.StreamChat(context.Background(), provider.ChatRequest{
		Model:    "qwen3-next",
		Messages: []provider.Message{{Role: "user", Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("StreamChat() error: %v", err)
	}

	var collected []string
	var finalDelta provider.StreamDelta
	for delta := range ch {
		if delta.Err != nil {
			t.Fatalf("stream error: %v", delta.Err)
		}
		if delta.Done {
			finalDelta = delta
			break
		}
		collected = append(collected, delta.Content)
	}

	if len(collected) != len(tokens) {
		t.Errorf("got %d tokens, want %d", len(collected), len(tokens))
	}
	for i, tok := range tokens {
		if i < len(collected) && collected[i] != tok {
			t.Errorf("token[%d] = %q, want %q", i, collected[i], tok)
		}
	}
	if !finalDelta.Done {
		t.Error("final delta should have Done=true")
	}
	if finalDelta.Usage == nil || finalDelta.Usage.TotalTokens != 14 {
		t.Errorf("final usage total = %v, want 14", finalDelta.Usage)
	}
}

func TestStreamChat_FinalChunk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher := w.(http.Flusher)

		chunk := ollamaChatResponse{
			Model:   "qwen3-next",
			Message: provider.Message{Role: "assistant", Content: "Hi"},
			Done:    false,
		}
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "%s\n", data)
		flusher.Flush()

		final := ollamaChatResponse{Done: true, EvalCount: 1}
		data, _ = json.Marshal(final)
		fmt.Fprintf(w, "%s\n", data)
		flusher.Flush()
	}))
	defer srv.Close()

	p := New(srv.URL)
	ch, err := p.StreamChat(context.Background(), provider.ChatRequest{
		Model:    "qwen3-next",
		Messages: []provider.Message{{Role: "user", Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("StreamChat() error: %v", err)
	}

	var gotDone bool
	for delta := range ch {
		if delta.Done {
			gotDone = true
		}
	}
	if !gotDone {
		t.Error("never received Done delta")
	}
}

func TestStreamChat_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher := w.(http.Flusher)
		fmt.Fprintf(w, "not valid json\n")
		flusher.Flush()
	}))
	defer srv.Close()

	p := New(srv.URL)
	ch, err := p.StreamChat(context.Background(), provider.ChatRequest{
		Model:    "qwen3-next",
		Messages: []provider.Message{{Role: "user", Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("StreamChat() error: %v", err)
	}

	var gotErr bool
	for delta := range ch {
		if delta.Err != nil {
			gotErr = true
		}
	}
	if !gotErr {
		t.Error("should have received error for malformed JSON")
	}
}

func TestStreamChat_ConnectionError(t *testing.T) {
	p := New("http://127.0.0.1:1")
	_, err := p.StreamChat(context.Background(), provider.ChatRequest{
		Model:    "qwen3-next",
		Messages: []provider.Message{{Role: "user", Content: "Hi"}},
	})
	if err == nil {
		t.Error("StreamChat() should return error for unreachable server")
	}
}

func TestStreamChat_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return nothing, just close
	}))
	defer srv.Close()

	p := New(srv.URL)
	ch, err := p.StreamChat(context.Background(), provider.ChatRequest{
		Model:    "qwen3-next",
		Messages: []provider.Message{{Role: "user", Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("StreamChat() error: %v", err)
	}

	count := 0
	for range ch {
		count++
	}
	// Channel should close with no deltas (or scanner just finishes)
	if count > 0 {
		t.Logf("got %d deltas from empty response (acceptable if 0)", count)
	}
}
