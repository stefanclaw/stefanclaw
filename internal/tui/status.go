package tui

import "fmt"

// StatusBar renders the top status bar.
func StatusBar(model, providerName string, width int) string {
	text := fmt.Sprintf("  stefanclaw - %s via %s  ", model, providerName)
	return statusBarStyle.Width(width).Render(text)
}
