package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func handleQuit(m *Model, args string) (tea.Model, tea.Cmd) {
	m.quitting = true
	return m, tea.Quit
}

func handleHelp(m *Model, args string) (tea.Model, tea.Cmd) {
	m.messages = append(m.messages, displayMessage{
		role:    "system",
		content: HelpText(),
	})
	m.updateViewport()
	return m, nil
}

func handleClear(m *Model, args string) (tea.Model, tea.Cmd) {
	m.messages = nil
	m.updateViewport()
	return m, nil
}

func handleModels(m *Model, args string) (tea.Model, tea.Cmd) {
	return m, m.listModels()
}

func handleModel(m *Model, args string) (tea.Model, tea.Cmd) {
	if args == "" {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Current model: %s\nUsage: /model <name>", m.options.Model),
		})
	} else {
		m.options.Model = args
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Switched to model: %s", args),
		})
	}
	m.updateViewport()
	return m, nil
}

func handleSession(m *Model, args string) (tea.Model, tea.Cmd) {
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

func handleMemory(m *Model, args string) (tea.Model, tea.Cmd) {
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

func handleRemember(m *Model, args string) (tea.Model, tea.Cmd) {
	if args == "" {
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

	if err := m.options.MemoryStore.Append([]string{args}); err != nil {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Error saving memory: %v", err),
		})
	} else {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Remembered: %s", args),
		})
	}
	m.updateViewport()
	return m, nil
}

func handleForget(m *Model, args string) (tea.Model, tea.Cmd) {
	if args == "" {
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

	removed, err := m.options.MemoryStore.Forget(args)
	if err != nil {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Error: %v", err),
		})
	} else if removed == 0 {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("No memory entries matching %q found.", args),
		})
	} else {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Forgot %d entries matching %q.", removed, args),
		})
	}
	m.updateViewport()
	return m, nil
}

func handleLanguage(m *Model, args string) (tea.Model, tea.Cmd) {
	if args == "" {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Current language: %s\nUsage: /language <name>", m.options.Language),
		})
	} else {
		m.options.Language = args
		if m.options.PromptAsm != nil {
			m.options.SystemPrompt = m.options.PromptAsm.BuildSystemPromptWithLanguage(args)
		}
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Language changed to: %s", args),
		})
	}
	m.updateViewport()
	return m, nil
}

func handleHeartbeat(m *Model, args string) (tea.Model, tea.Cmd) {
	switch args {
	case "":
		status := "disabled"
		if m.heartbeatEnabled {
			status = "enabled"
		}
		m.messages = append(m.messages, displayMessage{
			role: "system",
			content: fmt.Sprintf("Heartbeat: %s\nInterval: %s",
				status, m.heartbeatInterval),
		})
	case "on":
		m.heartbeatEnabled = true
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: fmt.Sprintf("Heartbeat enabled (every %s)", m.heartbeatInterval),
		})
		m.updateViewport()
		return m, m.scheduleHeartbeat()
	case "off":
		m.heartbeatEnabled = false
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: "Heartbeat disabled.",
		})
	default:
		dur, err := time.ParseDuration(args)
		if err != nil {
			m.messages = append(m.messages, displayMessage{
				role:    "system",
				content: fmt.Sprintf("Invalid interval: %s\nUsage: /heartbeat [on|off|<duration>]", args),
			})
		} else {
			m.heartbeatInterval = dur
			m.messages = append(m.messages, displayMessage{
				role:    "system",
				content: fmt.Sprintf("Heartbeat interval set to %s", dur),
			})
			if m.heartbeatEnabled {
				m.updateViewport()
				return m, m.scheduleHeartbeat()
			}
		}
	}
	m.updateViewport()
	return m, nil
}

func handleFetch(m *Model, args string) (tea.Model, tea.Cmd) {
	if args == "" {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: "Usage: /fetch <url>",
		})
		m.updateViewport()
		return m, nil
	}

	m.messages = append(m.messages, displayMessage{
		role:    "system",
		content: fmt.Sprintf("Fetching %s...", args),
	})
	m.updateViewport()

	client := m.fetchClient
	return m, func() tea.Msg {
		content, err := client.Fetch(context.Background(), args)
		if err != nil {
			return FetchErrMsg{Err: err}
		}
		return FetchDoneMsg{URL: args, Content: content}
	}
}

func handleSearch(m *Model, args string) (tea.Model, tea.Cmd) {
	if args == "" {
		m.messages = append(m.messages, displayMessage{
			role:    "system",
			content: "Usage: /search <query>",
		})
		m.updateViewport()
		return m, nil
	}

	m.messages = append(m.messages, displayMessage{
		role:    "system",
		content: fmt.Sprintf("Searching for %q...", args),
	})
	m.updateViewport()

	client := m.fetchClient
	return m, func() tea.Msg {
		content, err := client.Search(context.Background(), args)
		if err != nil {
			return SearchErrMsg{Err: err}
		}
		return SearchDoneMsg{Query: args, Content: content}
	}
}

func handlePersonality(m *Model, args string) (tea.Model, tea.Cmd) {
	if args == "edit" {
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
}

