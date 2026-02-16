package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Command represents a parsed slash command.
type Command struct {
	Name string
	Args string
}

// ParseCommand parses a slash command from input.
// Returns nil if the input is not a slash command.
func ParseCommand(input string) *Command {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return nil
	}

	input = input[1:] // strip leading /
	parts := strings.SplitN(input, " ", 2)
	cmd := &Command{Name: strings.ToLower(parts[0])}
	if len(parts) > 1 {
		cmd.Args = strings.TrimSpace(parts[1])
	}
	return cmd
}

// CommandHandler is the function signature for command handlers.
type CommandHandler func(m *Model, args string) (tea.Model, tea.Cmd)

// CommandDef describes a single slash command.
type CommandDef struct {
	Name        string
	Aliases     []string
	Description string
	Usage       string
	Handler     CommandHandler
}

// registry holds all registered commands.
var registry []CommandDef

func init() {
	registry = []CommandDef{
		{
			Name:        "help",
			Aliases:     []string{"h"},
			Description: "Show this help message",
			Usage:       "/help",
			Handler:     handleHelp,
		},
		{
			Name:        "quit",
			Aliases:     []string{"q", "bye", "exit"},
			Description: "Exit stefanclaw",
			Usage:       "/quit",
			Handler:     handleQuit,
		},
		{
			Name:        "models",
			Description: "List available models",
			Usage:       "/models",
			Handler:     handleModels,
		},
		{
			Name:        "model",
			Description: "Switch to a different model",
			Usage:       "/model <name>",
			Handler:     handleModel,
		},
		{
			Name:        "session",
			Description: "Start a new session or list sessions",
			Usage:       "/session new|list",
			Handler:     handleSession,
		},
		{
			Name:        "clear",
			Description: "Clear the current conversation display",
			Usage:       "/clear",
			Handler:     handleClear,
		},
		{
			Name:        "memory",
			Description: "Show current memory entries",
			Usage:       "/memory",
			Handler:     handleMemory,
		},
		{
			Name:        "remember",
			Description: "Add a fact to memory",
			Usage:       "/remember <fact>",
			Handler:     handleRemember,
		},
		{
			Name:        "forget",
			Description: "Remove matching memory entries",
			Usage:       "/forget <keyword>",
			Handler:     handleForget,
		},
		{
			Name:        "language",
			Description: "Show or change response language",
			Usage:       "/language [<name>]",
			Handler:     handleLanguage,
		},
		{
			Name:        "heartbeat",
			Description: "Manage heartbeat check-ins",
			Usage:       "/heartbeat [on|off|<interval>]",
			Handler:     handleHeartbeat,
		},
		{
			Name:        "fetch",
			Description: "Fetch a web page and display as markdown",
			Usage:       "/fetch <url>",
			Handler:     handleFetch,
		},
		{
			Name:        "search",
			Description: "Search the web and display results",
			Usage:       "/search <query>",
			Handler:     handleSearch,
		},
		{
			Name:        "personality",
			Description: "Open personality files in $EDITOR",
			Usage:       "/personality edit",
			Handler:     handlePersonality,
		},
		{
			Name:        "update",
			Aliases:     []string{"upgrade"},
			Description: "Check for updates and upgrade stefanclaw",
			Usage:       "/update",
			Handler:     handleUpdate,
		},
	}
}

// HelpText returns the help message for all slash commands.
func HelpText() string {
	var b strings.Builder
	b.WriteString("Available commands:\n")
	for _, def := range registry {
		line := fmt.Sprintf("  %-38s %s", def.Usage, def.Description)
		if len(def.Aliases) > 0 {
			aliases := make([]string, len(def.Aliases))
			for i, a := range def.Aliases {
				aliases[i] = "/" + a
			}
			line = fmt.Sprintf("  %-38s %s (aliases: %s)", def.Usage, def.Description, strings.Join(aliases, ", "))
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// handleCommand dispatches a parsed command to the matching handler.
func (m *Model) handleCommand(cmd *Command) (tea.Model, tea.Cmd) {
	for _, def := range registry {
		if cmd.Name == def.Name {
			return def.Handler(m, cmd.Args)
		}
		for _, alias := range def.Aliases {
			if cmd.Name == alias {
				return def.Handler(m, cmd.Args)
			}
		}
	}

	m.messages = append(m.messages, displayMessage{
		role:    "system",
		content: fmt.Sprintf("Unknown command: /%s. Type /help for available commands.", cmd.Name),
	})
	m.updateViewport()
	return m, nil
}
