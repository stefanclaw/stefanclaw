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
	commands := []string{"/help", "/quit", "/models", "/model", "/session", "/clear", "/memory", "/remember", "/forget", "/personality"}
	for _, cmd := range commands {
		if !contains(help, cmd) {
			t.Errorf("help text missing command: %s", cmd)
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
