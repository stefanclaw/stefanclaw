package tui

import "testing"

func TestParseSlashCommand(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantArgs string
	}{
		{"/help", "help", ""},
		{"/quit", "quit", ""},
		{"/model llama3", "model", "llama3"},
		{"/session new", "session", "new"},
		{"/remember user likes Go", "remember", "user likes Go"},
		{"/search latest Go news", "search", "latest Go news"},
		{"  /help  ", "help", ""},
	}

	for _, tt := range tests {
		cmd := ParseCommand(tt.input)
		if cmd == nil {
			t.Errorf("ParseCommand(%q) = nil, want command", tt.input)
			continue
		}
		if cmd.Name != tt.wantName {
			t.Errorf("ParseCommand(%q).Name = %q, want %q", tt.input, cmd.Name, tt.wantName)
		}
		if cmd.Args != tt.wantArgs {
			t.Errorf("ParseCommand(%q).Args = %q, want %q", tt.input, cmd.Args, tt.wantArgs)
		}
	}
}

func TestParseSlashCommand_NotACommand(t *testing.T) {
	tests := []string{
		"hello",
		"not a command",
		"",
		"  ",
	}

	for _, input := range tests {
		cmd := ParseCommand(input)
		if cmd != nil {
			t.Errorf("ParseCommand(%q) = %+v, want nil", input, cmd)
		}
	}
}

func TestHelpText(t *testing.T) {
	help := HelpText()
	commands := []string{"/help", "/quit", "/bye", "/exit", "/models", "/model", "/session", "/clear", "/memory", "/remember", "/forget", "/language", "/heartbeat", "/fetch", "/search", "/personality"}
	for _, cmd := range commands {
		if !contains(help, cmd) {
			t.Errorf("help text missing command: %s", cmd)
		}
	}
}

func TestParseExitAliases(t *testing.T) {
	for _, input := range []string{"/quit", "/bye", "/exit", "/q"} {
		cmd := ParseCommand(input)
		if cmd == nil {
			t.Errorf("ParseCommand(%q) = nil, want command", input)
			continue
		}
		// All should parse as valid commands that the TUI handles as exit
		switch cmd.Name {
		case "quit", "q", "bye", "exit":
			// ok
		default:
			t.Errorf("ParseCommand(%q).Name = %q, want quit/q/bye/exit", input, cmd.Name)
		}
	}
}

func TestParseLanguageCommand(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantArgs string
	}{
		{"/language", "language", ""},
		{"/language Deutsch", "language", "Deutsch"},
		{"/language English", "language", "English"},
	}
	for _, tt := range tests {
		cmd := ParseCommand(tt.input)
		if cmd == nil {
			t.Errorf("ParseCommand(%q) = nil", tt.input)
			continue
		}
		if cmd.Name != tt.wantName {
			t.Errorf("ParseCommand(%q).Name = %q, want %q", tt.input, cmd.Name, tt.wantName)
		}
		if cmd.Args != tt.wantArgs {
			t.Errorf("ParseCommand(%q).Args = %q, want %q", tt.input, cmd.Args, tt.wantArgs)
		}
	}
}

func TestParseFetchCommand(t *testing.T) {
	tests := []struct {
		input    string
		wantArgs string
	}{
		{"/fetch https://example.com", "https://example.com"},
		{"/fetch http://example.com/page", "http://example.com/page"},
		{"/fetch", ""},
	}
	for _, tt := range tests {
		cmd := ParseCommand(tt.input)
		if cmd == nil {
			t.Errorf("ParseCommand(%q) = nil", tt.input)
			continue
		}
		if cmd.Name != "fetch" {
			t.Errorf("ParseCommand(%q).Name = %q, want fetch", tt.input, cmd.Name)
		}
		if cmd.Args != tt.wantArgs {
			t.Errorf("ParseCommand(%q).Args = %q, want %q", tt.input, cmd.Args, tt.wantArgs)
		}
	}
}

func TestParseSearchCommand(t *testing.T) {
	tests := []struct {
		input    string
		wantArgs string
	}{
		{"/search latest Go news", "latest Go news"},
		{"/search capital of france", "capital of france"},
		{"/search", ""},
	}
	for _, tt := range tests {
		cmd := ParseCommand(tt.input)
		if cmd == nil {
			t.Errorf("ParseCommand(%q) = nil", tt.input)
			continue
		}
		if cmd.Name != "search" {
			t.Errorf("ParseCommand(%q).Name = %q, want search", tt.input, cmd.Name)
		}
		if cmd.Args != tt.wantArgs {
			t.Errorf("ParseCommand(%q).Args = %q, want %q", tt.input, cmd.Args, tt.wantArgs)
		}
	}
}

func TestParseHeartbeatCommand(t *testing.T) {
	tests := []struct {
		input    string
		wantArgs string
	}{
		{"/heartbeat", ""},
		{"/heartbeat on", "on"},
		{"/heartbeat off", "off"},
		{"/heartbeat 2h", "2h"},
	}
	for _, tt := range tests {
		cmd := ParseCommand(tt.input)
		if cmd == nil {
			t.Errorf("ParseCommand(%q) = nil", tt.input)
			continue
		}
		if cmd.Name != "heartbeat" {
			t.Errorf("ParseCommand(%q).Name = %q, want heartbeat", tt.input, cmd.Name)
		}
		if cmd.Args != tt.wantArgs {
			t.Errorf("ParseCommand(%q).Args = %q, want %q", tt.input, cmd.Args, tt.wantArgs)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
