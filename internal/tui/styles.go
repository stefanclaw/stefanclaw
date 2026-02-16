package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#6B7280")
	successColor   = lipgloss.Color("#10B981")

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	// Messages
	userLabelStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	assistantLabelStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	systemMsgStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Italic(true)

	// Input area
	inputPromptStyle = lipgloss.NewStyle().
				Foreground(primaryColor)
)
