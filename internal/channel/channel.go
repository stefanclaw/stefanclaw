package channel

// Channel defines the interface for message delivery channels (TUI, Telegram, etc).
// This is a placeholder for future multi-channel support.
type Channel interface {
	Name() string
	Start() error
	Stop() error
}
