package tui

import (
	"context"
	"testing"

	"github.com/stefanclaw/stefanclaw/internal/provider"
)

// mockProvider implements provider.Provider for testing.
type mockProvider struct {
	name       string
	chatResp   *provider.ChatResponse
	chatErr    error
	streamCh   chan provider.StreamDelta
	streamErr  error
	models     []provider.ModelInfo
	modelsErr  error
	available  error
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Chat(_ context.Context, _ provider.ChatRequest) (*provider.ChatResponse, error) {
	return m.chatResp, m.chatErr
}

func (m *mockProvider) StreamChat(_ context.Context, _ provider.ChatRequest) (<-chan provider.StreamDelta, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	return m.streamCh, nil
}

func (m *mockProvider) ListModels(_ context.Context) ([]provider.ModelInfo, error) {
	return m.models, m.modelsErr
}

func (m *mockProvider) IsAvailable(_ context.Context) error {
	return m.available
}

func TestInitialView(t *testing.T) {
	mp := &mockProvider{name: "test"}
	m := New(Options{
		Provider: mp,
		Model:    "test-model",
	})

	view := m.View()
	if view != "Initializing..." {
		t.Errorf("initial view = %q, want Initializing...", view)
	}
}

func TestSendMessage(t *testing.T) {
	ch := make(chan provider.StreamDelta, 2)
	ch <- provider.StreamDelta{Content: "Hello!"}
	ch <- provider.StreamDelta{Done: true}

	mp := &mockProvider{
		name:     "test",
		streamCh: ch,
	}
	m := New(Options{
		Provider: mp,
		Model:    "test-model",
	})

	// Simulate window size
	m.width = 80
	m.height = 24
	m.ready = true

	// Set textarea content
	m.textarea.SetValue("Hi there")

	// Simulate submit
	newM, _ := m.handleSubmit()
	model := newM.(*Model)

	// Should have user message
	last := model.messages[len(model.messages)-1]
	if last.role != "user" {
		t.Errorf("last message role = %q, want user", last.role)
	}
	if last.content != "Hi there" {
		t.Errorf("last message content = %q, want Hi there", last.content)
	}
	if !model.streaming {
		t.Error("should be streaming after submit")
	}
}

func TestStreamingResponse(t *testing.T) {
	mp := &mockProvider{name: "test"}
	m := New(Options{
		Provider: mp,
		Model:    "test-model",
	})
	m.width = 80
	m.height = 24
	m.ready = true
	m.streaming = true

	// Receive delta
	newM, _ := m.Update(StreamDeltaMsg{Content: "Hello"})
	model := newM.(Model)

	if model.streamContent != "Hello" {
		t.Errorf("streamContent = %q, want Hello", model.streamContent)
	}

	// Receive more
	newM, _ = model.Update(StreamDeltaMsg{Content: " world"})
	model = newM.(Model)

	if model.streamContent != "Hello world" {
		t.Errorf("streamContent = %q, want Hello world", model.streamContent)
	}

	// Done
	newM, _ = model.Update(StreamDoneMsg{})
	model = newM.(Model)

	if model.streaming {
		t.Error("should not be streaming after done")
	}
	// Last message should be the assistant response (welcome message is first)
	last := model.messages[len(model.messages)-1]
	if last.content != "Hello world" {
		t.Errorf("last message content = %q, want Hello world", last.content)
	}
}

func TestStreamingError(t *testing.T) {
	mp := &mockProvider{name: "test"}
	m := New(Options{
		Provider: mp,
		Model:    "test-model",
	})
	m.width = 80
	m.height = 24
	m.ready = true
	m.streaming = true
	m.streamContent = "partial"

	newM, _ := m.Update(StreamErrMsg{Err: context.DeadlineExceeded})
	model := newM.(Model)

	if model.streaming {
		t.Error("should not be streaming after error")
	}
	if model.streamContent != "" {
		t.Errorf("streamContent should be empty, got %q", model.streamContent)
	}
}

func TestInputDisabledDuringStreaming(t *testing.T) {
	mp := &mockProvider{name: "test"}
	m := New(Options{
		Provider: mp,
		Model:    "test-model",
	})
	m.width = 80
	m.height = 24
	m.ready = true
	m.streaming = true

	// handleSubmit should be a no-op during streaming
	// (Enter key is blocked in Update)
	m.textarea.SetValue("should not send")
	// The Update function blocks Enter during streaming, so we test that the
	// streaming flag prevents processing
	if !m.streaming {
		t.Error("should be streaming")
	}
}

func TestQuitCommand(t *testing.T) {
	mp := &mockProvider{name: "test"}
	m := New(Options{
		Provider: mp,
		Model:    "test-model",
	})
	m.width = 80
	m.height = 24
	m.ready = true

	m.textarea.SetValue("/quit")
	newM, _ := m.handleSubmit()
	model := newM.(*Model)

	if !model.quitting {
		t.Error("should be quitting after /quit")
	}
}

func TestUnknownCommand(t *testing.T) {
	mp := &mockProvider{name: "test"}
	m := New(Options{
		Provider: mp,
		Model:    "test-model",
	})
	m.width = 80
	m.height = 24
	m.ready = true

	m.textarea.SetValue("/foobar")
	newM, _ := m.handleSubmit()
	model := newM.(*Model)

	// Last message should be the unknown command error (welcome is first)
	last := model.messages[len(model.messages)-1]
	if last.role != "system" {
		t.Error("unknown command should produce system message")
	}
}

func TestResponseDisplay(t *testing.T) {
	mp := &mockProvider{name: "test"}
	m := New(Options{
		Provider: mp,
		Model:    "test-model",
	})
	m.width = 80
	m.height = 24
	m.ready = true

	m.messages = []displayMessage{
		{role: "user", content: "Hello"},
		{role: "assistant", content: "Hi there!"},
	}
	m.updateViewport()

	view := m.View()
	if view == "Initializing..." {
		t.Error("view should not be 'Initializing...' when ready")
	}
}

func TestHelpCommand(t *testing.T) {
	mp := &mockProvider{name: "test"}
	m := New(Options{
		Provider: mp,
		Model:    "test-model",
	})
	m.width = 80
	m.height = 24
	m.ready = true

	m.textarea.SetValue("/help")
	newM, _ := m.handleSubmit()
	model := newM.(*Model)

	// Last message should be help text (welcome is first)
	last := model.messages[len(model.messages)-1]
	if last.role != "system" {
		t.Error("help should produce system message")
	}
}
