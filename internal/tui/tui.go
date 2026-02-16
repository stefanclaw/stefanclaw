package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/stefanclaw/stefanclaw/internal/memory"
	"github.com/stefanclaw/stefanclaw/internal/prompt"
	"github.com/stefanclaw/stefanclaw/internal/provider"
	"github.com/stefanclaw/stefanclaw/internal/session"
)

// Options configures the TUI.
type Options struct {
	Provider       provider.Provider
	SessionStore   session.Store
	MemoryStore    *memory.Store
	PromptAsm      *prompt.Assembler
	SystemPrompt   string
	Model          string
	Session        *session.Session
	PersonalityDir string
}

// StreamDeltaMsg carries a streaming token.
type StreamDeltaMsg struct {
	Content string
}

// StreamDoneMsg signals the end of a streaming response.
type StreamDoneMsg struct {
	Usage *provider.Usage
}

// StreamErrMsg carries a streaming error.
type StreamErrMsg struct {
	Err error
}

// ModelListMsg carries the result of listing models.
type ModelListMsg struct {
	Models []provider.ModelInfo
	Err    error
}

// Model is the Bubble Tea model for the chat TUI.
type Model struct {
	options  Options
	viewport viewport.Model
	textarea textarea.Model
	messages []displayMessage
	width    int
	height   int

	streaming       bool
	streamContent   string
	streamCancelFn  context.CancelFunc
	streamCh        <-chan provider.StreamDelta

	mdRenderer *glamour.TermRenderer
	err        error
	ready      bool
	quitting   bool
}

type displayMessage struct {
	role    string
	content string
}

// New creates a new TUI model.
func New(opts Options) Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message... (or /help)"
	ta.Focus()
	ta.CharLimit = 4096
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.Prompt = inputPromptStyle.Render("> ")

	vp := viewport.New(80, 20)

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(76),
	)

	return Model{
		options:    opts,
		textarea:   ta,
		viewport:   vp,
		mdRenderer: renderer,
	}
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			if m.streaming && m.streamCancelFn != nil {
				m.streamCancelFn()
				m.streaming = false
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEnter:
			if m.streaming {
				return m, nil
			}
			return m.handleSubmit()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		statusH := 1
		inputH := 5
		viewH := m.height - statusH - inputH
		if viewH < 1 {
			viewH = 1
		}
		m.viewport.Width = m.width
		m.viewport.Height = viewH
		m.textarea.SetWidth(m.width)
		m.ready = true

	case StreamDeltaMsg:
		m.streamContent += msg.Content
		m.updateViewport()
		return m, m.waitForDelta()

	case StreamDoneMsg:
		m.streaming = false
		if m.streamContent != "" {
			m.messages = append(m.messages, displayMessage{
				role:    "assistant",
				content: m.streamContent,
			})
			// Save to transcript
			if m.options.Session != nil && m.options.SessionStore != nil {
				m.options.SessionStore.Append(m.options.Session.ID, provider.Message{
					Role:    "assistant",
					Content: m.streamContent,
				})
			}
		}
		m.streamContent = ""
		m.updateViewport()
		return m, nil

	case StreamErrMsg:
		m.streaming = false
		m.err = msg.Err
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Error: %v", msg.Err),
		})
		m.streamContent = ""
		m.updateViewport()
		return m, nil

	case ModelListMsg:
		if msg.Err != nil {
			m.messages = append(m.messages, displayMessage{
				role:    "system",
				content: fmt.Sprintf("Error listing models: %v", msg.Err),
			})
		} else {
			var lines []string
			lines = append(lines, "Available models:")
			for _, model := range msg.Models {
				marker := "  "
				if model.Name == m.options.Model {
					marker = "* "
				}
				lines = append(lines, fmt.Sprintf("%s%s", marker, model.Name))
			}
			m.messages = append(m.messages, displayMessage{
				role:    "system",
				content: strings.Join(lines, "\n"),
			})
		}
		m.updateViewport()
		return m, nil
	}

	// Update textarea
	if !m.streaming {
		var taCmd tea.Cmd
		m.textarea, taCmd = m.textarea.Update(msg)
		cmds = append(cmds, taCmd)
	}

	// Update viewport
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}
	if !m.ready {
		return "Initializing..."
	}

	status := StatusBar(m.options.Model, m.options.Provider.Name(), m.width)
	separator := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Width(m.width).
		Render(strings.Repeat("─", m.width))

	return fmt.Sprintf("%s\n%s\n%s\n%s",
		status,
		m.viewport.View(),
		separator,
		m.textarea.View(),
	)
}

func (m *Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textarea.Value())
	if input == "" {
		return m, nil
	}
	m.textarea.Reset()

	// Check for slash command
	if cmd := ParseCommand(input); cmd != nil {
		return m.handleCommand(cmd)
	}

	// Add user message
	m.messages = append(m.messages, displayMessage{role: "user", content: input})

	// Save to transcript
	if m.options.Session != nil && m.options.SessionStore != nil {
		m.options.SessionStore.Append(m.options.Session.ID, provider.Message{
			Role:    "user",
			Content: input,
		})
	}

	m.updateViewport()

	// Start streaming
	m.streaming = true
	m.streamContent = ""

	ctx, cancel := context.WithCancel(context.Background())
	m.streamCancelFn = cancel

	return m, m.startStream(ctx, input)
}

func (m *Model) handleCommand(cmd *Command) (tea.Model, tea.Cmd) {
	switch cmd.Name {
	case "quit", "q":
		m.quitting = true
		return m, tea.Quit

	case "help", "h":
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: HelpText(),
		})
		m.updateViewport()
		return m, nil

	case "clear":
		m.messages = nil
		m.updateViewport()
		return m, nil

	case "models":
		return m, m.listModels()

	case "model":
		if cmd.Args == "" {
			m.messages = append(m.messages, displayMessage{
				role:    "system",
				content: fmt.Sprintf("Current model: %s\nUsage: /model <name>", m.options.Model),
			})
		} else {
			m.options.Model = cmd.Args
			m.messages = append(m.messages, displayMessage{
				role:    "system",
				content: fmt.Sprintf("Switched to model: %s", cmd.Args),
			})
		}
		m.updateViewport()
		return m, nil

	case "session":
		return m.handleSessionCommand(cmd.Args)

	case "memory":
		return m.handleMemoryCommand()

	case "remember":
		return m.handleRememberCommand(cmd.Args)

	case "forget":
		return m.handleForgetCommand(cmd.Args)

	case "personality":
		if cmd.Args == "edit" {
			m.messages = append(m.messages, displayMessage{
				role:    "system",
				content: fmt.Sprintf("Open your personality files at:\n  %s", m.options.PersonalityDir),
			})
		} else {
			m.messages = append(m.messages, displayMessage{
				role:    "system",
				content: "Usage: /personality edit",
			})
		}
		m.updateViewport()
		return m, nil

	default:
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Unknown command: /%s. Type /help for available commands.", cmd.Name),
		})
		m.updateViewport()
		return m, nil
	}
}

func (m *Model) handleSessionCommand(args string) (tea.Model, tea.Cmd) {
	switch args {
	case "new":
		if m.options.SessionStore != nil {
			s, err := m.options.SessionStore.Create("New Chat", m.options.Model)
			if err != nil {
				m.messages = append(m.messages, displayMessage{
					role:    "system",
					content: fmt.Sprintf("Error creating session: %v", err),
				})
			} else {
				m.options.Session = s
				m.options.SessionStore.SetCurrent(s.ID)
				m.messages = nil
				m.messages = append(m.messages, displayMessage{
					role:    "system",
					content: fmt.Sprintf("New session: %s", s.ID),
				})
			}
		}
	case "list":
		if m.options.SessionStore != nil {
			sessions, err := m.options.SessionStore.List()
			if err != nil {
				m.messages = append(m.messages, displayMessage{
					role:    "system",
					content: fmt.Sprintf("Error listing sessions: %v", err),
				})
			} else if len(sessions) == 0 {
				m.messages = append(m.messages, displayMessage{
					role:    "system",
					content: "No sessions found.",
				})
			} else {
				var lines []string
				lines = append(lines, "Sessions:")
				for _, s := range sessions {
					marker := "  "
					if m.options.Session != nil && s.ID == m.options.Session.ID {
						marker = "* "
					}
					lines = append(lines, fmt.Sprintf("%s%s - %s (%s)",
						marker, s.ID, s.Title, s.Model))
				}
				m.messages = append(m.messages, displayMessage{
					role:    "system",
					content: strings.Join(lines, "\n"),
				})
			}
		}
	default:
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: "Usage: /session new | /session list",
		})
	}
	m.updateViewport()
	return m, nil
}

func (m *Model) buildMessages(userInput string) []provider.Message {
	var msgs []provider.Message

	if m.options.SystemPrompt != "" {
		msgs = append(msgs, provider.Message{
			Role:    "system",
			Content: m.options.SystemPrompt,
		})
	}

	// Add conversation history
	for _, dm := range m.messages {
		if dm.role == "user" || dm.role == "assistant" {
			msgs = append(msgs, provider.Message{
				Role:    dm.role,
				Content: dm.content,
			})
		}
	}

	return msgs
}

func (m *Model) startStream(ctx context.Context, userInput string) tea.Cmd {
	return func() tea.Msg {
		msgs := m.buildMessages(userInput)
		ch, err := m.options.Provider.StreamChat(ctx, provider.ChatRequest{
			Model:    m.options.Model,
			Messages: msgs,
		})
		if err != nil {
			return StreamErrMsg{Err: err}
		}

		m.streamCh = ch
		return readFromChannel(ch)
	}
}

func (m *Model) waitForDelta() tea.Cmd {
	ch := m.streamCh
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		return readFromChannel(ch)
	}
}

func readFromChannel(ch <-chan provider.StreamDelta) tea.Msg {
	delta, ok := <-ch
	if !ok {
		return StreamDoneMsg{}
	}
	if delta.Err != nil {
		return StreamErrMsg{Err: delta.Err}
	}
	if delta.Done {
		return StreamDoneMsg{Usage: delta.Usage}
	}
	return StreamDeltaMsg{Content: delta.Content}
}

func (m *Model) renderMarkdown(content string) string {
	if m.mdRenderer == nil {
		return content
	}
	rendered, err := m.mdRenderer.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimSpace(rendered)
}

func (m *Model) updateViewport() {
	var lines []string
	for _, msg := range m.messages {
		switch msg.role {
		case "user":
			label := userLabelStyle.Render("You: ")
			lines = append(lines, label+msg.content)
		case "assistant":
			label := assistantLabelStyle.Render("Assistant: ")
			rendered := m.renderMarkdown(msg.content)
			lines = append(lines, label+rendered)
		case "system":
			lines = append(lines, systemMsgStyle.Render(msg.content))
		}
		lines = append(lines, "")
	}

	// Show streaming content (no markdown rendering during streaming for speed)
	if m.streaming && m.streamContent != "" {
		label := assistantLabelStyle.Render("Assistant: ")
		lines = append(lines, label+m.streamContent+"▌")
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m *Model) listModels() tea.Cmd {
	return func() tea.Msg {
		models, err := m.options.Provider.ListModels(context.Background())
		return ModelListMsg{Models: models, Err: err}
	}
}

func (m *Model) handleMemoryCommand() (tea.Model, tea.Cmd) {
	if m.options.MemoryStore == nil {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: "Memory system not configured.",
		})
		m.updateViewport()
		return m, nil
	}

	entries, err := m.options.MemoryStore.Entries()
	if err != nil {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Error reading memory: %v", err),
		})
	} else if len(entries) == 0 {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: "No memory entries yet.",
		})
	} else {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: "Memory:\n" + strings.Join(entries, "\n"),
		})
	}
	m.updateViewport()
	return m, nil
}

func (m *Model) handleRememberCommand(fact string) (tea.Model, tea.Cmd) {
	if fact == "" {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: "Usage: /remember <fact>",
		})
		m.updateViewport()
		return m, nil
	}

	if m.options.MemoryStore == nil {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: "Memory system not configured.",
		})
		m.updateViewport()
		return m, nil
	}

	if err := m.options.MemoryStore.Append([]string{fact}); err != nil {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Error saving memory: %v", err),
		})
	} else {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Remembered: %s", fact),
		})
	}
	m.updateViewport()
	return m, nil
}

func (m *Model) handleForgetCommand(keyword string) (tea.Model, tea.Cmd) {
	if keyword == "" {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: "Usage: /forget <keyword>",
		})
		m.updateViewport()
		return m, nil
	}

	if m.options.MemoryStore == nil {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: "Memory system not configured.",
		})
		m.updateViewport()
		return m, nil
	}

	removed, err := m.options.MemoryStore.Forget(keyword)
	if err != nil {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Error: %v", err),
		})
	} else if removed == 0 {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("No memory entries matching %q found.", keyword),
		})
	} else {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Forgot %d entries matching %q.", removed, keyword),
		})
	}
	m.updateViewport()
	return m, nil
}
