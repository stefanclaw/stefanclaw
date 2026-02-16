package config

import (
	"os"
	"path/filepath"
)

// Dir returns the configuration directory path (~/.config/stefanclaw).
// It can be overridden with the STEFANCLAW_CONFIG_DIR environment variable.
func Dir() string {
	if d := os.Getenv("STEFANCLAW_CONFIG_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".config", "stefanclaw")
	}
	return filepath.Join(home, ".config", "stefanclaw")
}

// PersonalityDir returns the path to the personality files directory.
func PersonalityDir() string {
	return filepath.Join(Dir(), "personality")
}

// SessionsDir returns the path to the sessions directory.
func SessionsDir() string {
	return filepath.Join(Dir(), "sessions")
}

// ConfigFile returns the path to the config.yaml file.
func ConfigFile() string {
	return filepath.Join(Dir(), "config.yaml")
}
