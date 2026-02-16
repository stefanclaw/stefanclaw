package tui

import "strings"

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

// HelpText returns the help message for all slash commands.
func HelpText() string {
	return `Available commands:
  /help                Show this help message
  /quit, /bye, /exit   Exit stefanclaw
  /models              List available models
  /model <name>        Switch to a different model
  /session new         Start a new session
  /session list        List all sessions
  /clear               Clear the current conversation display
  /memory              Show current memory entries
  /remember <fact>     Add a fact to memory
  /forget <keyword>    Remove matching memory entries
  /language [<name>]   Show or change response language
  /heartbeat [on|off|<interval>]  Manage heartbeat check-ins
  /fetch <url>         Fetch a web page and display as markdown
  /search <query>      Search the web and display results
  /personality edit    Open personality files in $EDITOR`
}
