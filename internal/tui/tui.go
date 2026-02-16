package tui

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/stefanclaw/stefanclaw/internal/config"
	"github.com/stefanclaw/stefanclaw/internal/fetch"
	"github.com/stefanclaw/stefanclaw/internal/memory"
	"github.com/stefanclaw/stefanclaw/internal/prompt"
	"github.com/stefanclaw/stefanclaw/internal/provider"
	"github.com/stefanclaw/stefanclaw/internal/session"
	"github.com/stefanclaw/stefanclaw/internal/update"
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
	Language       string
	Heartbeat      config.HeartbeatConfig
	MaxNumCtx      int
	Version        string
}

// ctxTiers defines the adaptive context size tiers.
var ctxTiers = []int{4096, 8192, 16384, 32768}

// StreamStartedMsg carries the channel after the stream connection is established.
type StreamStartedMsg struct {
	Ch <-chan provider.StreamDelta
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

// HeartbeatTickMsg signals a heartbeat check-in is due.
type HeartbeatTickMsg struct{}

// FetchDoneMsg carries the result of a web fetch.
type FetchDoneMsg struct {
	URL     string
	Content string
}

// FetchErrMsg carries a web fetch error.
type FetchErrMsg struct {
	Err error
}

// SearchDoneMsg carries the result of a web search.
type SearchDoneMsg struct {
	Query   string
	Content string
}

// SearchErrMsg carries a web search error.
type SearchErrMsg struct {
	Err error
}

// UpdateCheckMsg carries the result of a background update check.
type UpdateCheckMsg struct {
	Result *update.Result
	Err    error
}

// UpdateApplyMsg carries the result of an update apply.
type UpdateApplyMsg struct {
	Result *update.Result
	Err    error
}

// Model is the Bubble Tea model for the chat TUI.
type Model struct {
	options  Options
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model
	messages []displayMessage
	width    int
	height   int

	streaming       bool
	streamContent   string
	streamCancelFn  context.CancelFunc
	streamCh        <-chan provider.StreamDelta
	waiting         bool // true while waiting for first token

	mdRenderer  *glamour.TermRenderer
	err         error
	ready       bool
	quitting    bool
	autoGreet       bool // trigger LLM greeting on first window size
	bootstrapStream bool // true when current stream is the first-run greeting

	heartbeatInterval time.Duration
	heartbeatEnabled  bool
	heartbeatStream   bool // true when current stream is a heartbeat check-in

	currentNumCtx int // Current adaptive context size
	maxNumCtx     int // Upper limit from config

	fetchClient *fetch.Client
}

type displayMessage struct {
	role    string
	content string
}

// New creates a new TUI model.
func New(opts Options) Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message... (Alt+Enter for newline)"
	ta.Focus()
	ta.CharLimit = 4096
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.Prompt = inputPromptStyle.Render("> ")

	vp := viewport.New(80, 20)

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(76),
	)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = assistantLabelStyle

	isFirstRun := opts.PromptAsm != nil && opts.PromptAsm.HasBootstrap()

	heartbeatInterval, _ := time.ParseDuration(opts.Heartbeat.Interval)
	if heartbeatInterval <= 0 {
		heartbeatInterval = 4 * time.Hour
	}

	maxCtx := opts.MaxNumCtx
	if maxCtx <= 0 {
		maxCtx = 32768
	}

	return Model{
		options:           opts,
		textarea:          ta,
		viewport:          vp,
		spinner:           sp,
		mdRenderer:        renderer,
		autoGreet:         isFirstRun,
		heartbeatEnabled:  opts.Heartbeat.Enabled,
		heartbeatInterval: heartbeatInterval,
		currentNumCtx:     ctxTiers[0],
		maxNumCtx:         maxCtx,
		fetchClient:       fetch.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
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
			if msg.Alt {
				m.textarea.InsertString("\n")
				return m, nil
			}
			return m.handleSubmit()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		statusH := 1
		inputH := 3
		viewH := m.height - statusH - inputH
		if viewH < 1 {
			viewH = 1
		}
		m.viewport.Width = m.width
		m.viewport.Height = viewH
		m.textarea.SetWidth(m.width)

		if !m.ready {
			m.ready = true
			var initCmds []tea.Cmd
			if m.autoGreet {
				m.autoGreet = false
				m.messages = append(m.messages, displayMessage{
					role:    "system",
					content: "Starting up... waiting for model to respond.",
				})
				initCmds = append(initCmds, m.triggerAutoGreet())
			}
			if m.heartbeatEnabled {
				initCmds = append(initCmds, m.scheduleHeartbeat())
			}
			// Background update check (only for release builds)
			if v := m.options.Version; v != "" && v != "dev" {
				initCmds = append(initCmds, m.checkForUpdate())
			}
			if len(initCmds) > 0 {
				m.updateViewport()
				initCmds = append(initCmds, m.spinner.Tick)
				return m, tea.Batch(initCmds...)
			}
		}
		m.updateViewport()

	case StreamStartedMsg:
		m.streamCh = msg.Ch
		m.waiting = true
		m.updateViewport()
		return m, tea.Batch(waitForDelta(m.streamCh), m.spinner.Tick)

	case StreamDeltaMsg:
		m.waiting = false
		m.streamContent += msg.Content
		m.updateViewport()
		return m, waitForDelta(m.streamCh)

	case StreamDoneMsg:
		m.streaming = false
		m.waiting = false
		wasHeartbeat := m.heartbeatStream
		m.heartbeatStream = false

		// Delete BOOTSTRAP.md after first greeting so auto-greet doesn't fire again
		if m.bootstrapStream {
			m.bootstrapStream = false
			if m.options.PromptAsm != nil {
				m.options.PromptAsm.DeleteBootstrap()
			}
		}

		// Adaptive context scaling: check if we need to grow
		if msg.Usage != nil && msg.Usage.PromptTokens > 0 {
			threshold := int(float64(m.currentNumCtx) * 0.6)
			if msg.Usage.PromptTokens > threshold {
				for _, tier := range ctxTiers {
					if tier > m.currentNumCtx && tier <= m.maxNumCtx {
						m.currentNumCtx = tier
						m.messages = append(m.messages, displayMessage{
							role: "system",
							content: fmt.Sprintf("Context expanded to %d tokens (conversation is growing). The next response may take a moment while the model reloads.", tier),
						})
						break
					}
				}
			}
		}

		if m.streamContent != "" {
			// Heartbeat skip: discard silently
			if wasHeartbeat && strings.Contains(m.streamContent, "HEARTBEAT_SKIP") {
				m.streamContent = ""
				m.updateViewport()
				if m.heartbeatEnabled {
					return m, m.scheduleHeartbeat()
				}
				return m, nil
			}

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

		// Reschedule heartbeat after a response completes
		if wasHeartbeat && m.heartbeatEnabled {
			return m, m.scheduleHeartbeat()
		}
		return m, nil

	case StreamErrMsg:
		m.streaming = false
		m.waiting = false
		m.err = msg.Err
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Error: %v", msg.Err),
		})
		m.streamContent = ""
		m.updateViewport()
		return m, nil

	case HeartbeatTickMsg:
		if m.streaming || !m.heartbeatEnabled {
			return m, nil
		}
		return m, m.triggerHeartbeat()

	case FetchDoneMsg:
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Fetched %s:\n\n%s", msg.URL, msg.Content),
		})
		m.updateViewport()
		return m, nil

	case FetchErrMsg:
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Fetch error: %v", msg.Err),
		})
		m.updateViewport()
		return m, nil

	case SearchDoneMsg:
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Search results for %q:\n\n%s", msg.Query, msg.Content),
		})
		m.updateViewport()
		return m, nil

	case SearchErrMsg:
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Search error: %v", msg.Err),
		})
		m.updateViewport()
		return m, nil

	case UpdateCheckMsg:
		if msg.Err == nil && msg.Result != nil && msg.Result.UpdateAvailable {
			m.messages = append(m.messages, displayMessage{
				role: "system",
				content: fmt.Sprintf("Update available: v%s → v%s. Run /update to upgrade.", msg.Result.CurrentVersion, msg.Result.LatestVersion),
			})
			m.updateViewport()
		}
		return m, nil

	case UpdateApplyMsg:
		if msg.Err != nil {
			m.messages = append(m.messages, displayMessage{
				role:    "system",
				content: fmt.Sprintf("Update failed: %v", msg.Err),
			})
		} else if msg.Result.Applied {
			m.messages = append(m.messages, displayMessage{
				role:    "system",
				content: fmt.Sprintf("Updated to v%s. Restart stefanclaw to use the new version.", msg.Result.LatestVersion),
			})
		} else {
			m.messages = append(m.messages, displayMessage{
				role:    "system",
				content: "Already running the latest version.",
			})
		}
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

	// Update spinner when streaming with no content yet
	if m.streaming && m.streamContent == "" {
		var spCmd tea.Cmd
		m.spinner, spCmd = m.spinner.Update(msg)
		cmds = append(cmds, spCmd)
		m.updateViewport()
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

	// Start streaming
	m.streaming = true
	m.waiting = true
	m.streamContent = ""

	// Update viewport after setting waiting=true so the spinner renders immediately
	m.updateViewport()

	ctx, cancel := context.WithCancel(context.Background())
	m.streamCancelFn = cancel

	var cmds []tea.Cmd
	cmds = append(cmds, m.startStream(ctx, input), m.spinner.Tick)

	// Reset heartbeat timer on user activity
	if m.heartbeatEnabled {
		cmds = append(cmds, m.scheduleHeartbeat())
	}

	return m, tea.Batch(cmds...)
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

// urlPattern matches http and https URLs in user messages.
var urlPattern = regexp.MustCompile(`https?://[^\s)<>]+`)

func (m *Model) startStream(ctx context.Context, userInput string) tea.Cmd {
	// Capture what we need — the closure must not rely on m fields surviving
	sysProm := m.options.SystemPrompt
	model := m.options.Model
	prov := m.options.Provider
	msgs := m.buildMessages(userInput)
	numCtx := m.currentNumCtx
	fetchClient := m.fetchClient

	return func() tea.Msg {
		_ = sysProm // already included via buildMessages

		// Auto-fetch URLs found in the user's message
		if urls := urlPattern.FindAllString(userInput, 3); len(urls) > 0 {
			var fetched []string
			for _, u := range urls {
				content, err := fetchClient.Fetch(ctx, u)
				if err == nil && content != "" {
					fetched = append(fetched, fmt.Sprintf("[Web content from %s]\n%s\n[End web content]", u, content))
				}
			}
			if len(fetched) > 0 {
				// Append fetched content to the last user message
				last := &msgs[len(msgs)-1]
				last.Content += "\n\n" + strings.Join(fetched, "\n\n")
			}
		}

		ch, err := prov.StreamChat(ctx, provider.ChatRequest{
			Model:    model,
			Messages: msgs,
			NumCtx:   numCtx,
		})
		if err != nil {
			return StreamErrMsg{Err: err}
		}
		return StreamStartedMsg{Ch: ch}
	}
}

func (m *Model) triggerAutoGreet() tea.Cmd {
	m.streaming = true
	m.waiting = true
	m.streamContent = ""
	m.bootstrapStream = true

	ctx, cancel := context.WithCancel(context.Background())
	m.streamCancelFn = cancel

	// Capture values for the closure
	sysProm := m.options.SystemPrompt
	model := m.options.Model
	prov := m.options.Provider
	lang := m.options.Language
	numCtx := m.currentNumCtx

	return func() tea.Msg {
		var msgs []provider.Message
		if sysProm != "" {
			msgs = append(msgs, provider.Message{
				Role:    "system",
				Content: sysProm,
			})
		}

		greetMsg := "Hello! Please greet me briefly and let me know you're ready to chat."
		if lang != "" && lang != "English" {
			greetMsg += " Respond in " + lang + "."
		}

		msgs = append(msgs, provider.Message{
			Role:    "user",
			Content: greetMsg,
		})

		ch, err := prov.StreamChat(ctx, provider.ChatRequest{
			Model:    model,
			Messages: msgs,
			NumCtx:   numCtx,
		})
		if err != nil {
			return StreamErrMsg{Err: err}
		}
		return StreamStartedMsg{Ch: ch}
	}
}

// waitForDelta reads the next item from a stream channel.
func waitForDelta(ch <-chan provider.StreamDelta) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
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
			lines = append(lines, lipgloss.NewStyle().Width(m.width).Render(label+msg.content))
		case "assistant":
			label := assistantLabelStyle.Render("Assistant: ")
			rendered := m.renderMarkdown(msg.content)
			lines = append(lines, label+rendered)
		case "system":
			lines = append(lines, systemMsgStyle.Render(msg.content))
		}
		lines = append(lines, "")
	}

	// Show spinner while waiting for LLM response
	if m.streaming && m.streamContent == "" {
		lines = append(lines, m.spinner.View()+" Thinking...")
		lines = append(lines, "")
	}

	// Show streaming content (no markdown rendering during streaming for speed)
	if m.streaming && m.streamContent != "" {
		label := assistantLabelStyle.Render("Assistant: ")
		lines = append(lines, lipgloss.NewStyle().Width(m.width).Render(label+m.streamContent+"▌"))
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

func (m *Model) scheduleHeartbeat() tea.Cmd {
	d := m.heartbeatInterval
	return tea.Tick(d, func(time.Time) tea.Msg {
		return HeartbeatTickMsg{}
	})
}

func (m *Model) checkForUpdate() tea.Cmd {
	version := m.options.Version
	return func() tea.Msg {
		res, err := update.Check(context.Background(), version)
		return UpdateCheckMsg{Result: res, Err: err}
	}
}

func (m *Model) triggerHeartbeat() tea.Cmd {
	m.streaming = true
	m.streamContent = ""
	m.heartbeatStream = true

	ctx, cancel := context.WithCancel(context.Background())
	m.streamCancelFn = cancel

	sysProm := m.options.SystemPrompt
	model := m.options.Model
	prov := m.options.Provider
	lang := m.options.Language
	numCtx := m.currentNumCtx

	return func() tea.Msg {
		var msgs []provider.Message
		if sysProm != "" {
			msgs = append(msgs, provider.Message{
				Role:    "system",
				Content: sysProm,
			})
		}

		// Include conversation context
		heartbeatPrompt := "[Heartbeat check-in] Review the user's memory and conversation context. If there's something relevant to say, say it briefly. If not, respond with exactly 'HEARTBEAT_SKIP'."
		if lang != "" && lang != "English" {
			heartbeatPrompt += " Respond in " + lang + "."
		}

		msgs = append(msgs, provider.Message{
			Role:    "user",
			Content: heartbeatPrompt,
		})

		ch, err := prov.StreamChat(ctx, provider.ChatRequest{
			Model:    model,
			Messages: msgs,
			NumCtx:   numCtx,
		})
		if err != nil {
			return StreamErrMsg{Err: err}
		}
		return StreamStartedMsg{Ch: ch}
	}
}

