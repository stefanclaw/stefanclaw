package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	Provider    ProviderConfig    `yaml:"provider"`
	Model       ModelConfig       `yaml:"model"`
	Personality PersonalityConfig `yaml:"personality"`
	Session     SessionConfig     `yaml:"session"`
	Memory      MemoryConfig      `yaml:"memory"`
	TUI         TUIConfig         `yaml:"tui"`
}

// ProviderConfig holds provider settings.
type ProviderConfig struct {
	Default string       `yaml:"default"`
	Ollama  OllamaConfig `yaml:"ollama"`
}

// OllamaConfig holds Ollama-specific settings.
type OllamaConfig struct {
	BaseURL string `yaml:"base_url"`
}

// ModelConfig holds model settings.
type ModelConfig struct {
	Default string `yaml:"default"`
}

// PersonalityConfig holds personality directory settings.
type PersonalityConfig struct {
	Dir string `yaml:"dir"`
}

// SessionConfig holds session directory settings.
type SessionConfig struct {
	Dir string `yaml:"dir"`
}

// MemoryConfig holds memory settings.
type MemoryConfig struct {
	Enabled        bool `yaml:"enabled"`
	MaxPromptTokens int  `yaml:"max_prompt_tokens"`
}

// TUIConfig holds TUI settings.
type TUIConfig struct {
	Theme string `yaml:"theme"`
}

// Defaults returns a Config with sensible defaults.
func Defaults() Config {
	return Config{
		Provider: ProviderConfig{
			Default: "ollama",
			Ollama: OllamaConfig{
				BaseURL: "http://127.0.0.1:11434",
			},
		},
		Model: ModelConfig{
			Default: "qwen3-next",
		},
		Personality: PersonalityConfig{
			Dir: "personality",
		},
		Session: SessionConfig{
			Dir: "sessions",
		},
		Memory: MemoryConfig{
			Enabled:        true,
			MaxPromptTokens: 2000,
		},
		TUI: TUIConfig{
			Theme: "auto",
		},
	}
}

// Load reads the config from disk. If the file doesn't exist, returns defaults.
func Load() (Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(ConfigFile())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Defaults(), err
	}

	return cfg, nil
}

// Save writes the config to disk.
func Save(cfg Config) error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigFile(), data, 0o644)
}

// IsFirstRun returns true if the config directory does not exist.
func IsFirstRun() bool {
	_, err := os.Stat(ConfigFile())
	return os.IsNotExist(err)
}
