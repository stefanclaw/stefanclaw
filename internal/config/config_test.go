package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()

	if cfg.Provider.Default != "ollama" {
		t.Errorf("default provider = %q, want ollama", cfg.Provider.Default)
	}
	if cfg.Provider.Ollama.BaseURL != "http://127.0.0.1:11434" {
		t.Errorf("ollama base_url = %q, want http://127.0.0.1:11434", cfg.Provider.Ollama.BaseURL)
	}
	if cfg.Provider.Ollama.MaxNumCtx != 32768 {
		t.Errorf("ollama max_num_ctx = %d, want 32768", cfg.Provider.Ollama.MaxNumCtx)
	}
	if cfg.Model.Default != "qwen3:8b" {
		t.Errorf("default model = %q, want qwen3:8b", cfg.Model.Default)
	}
	if !cfg.Memory.Enabled {
		t.Error("memory should be enabled by default")
	}
	if cfg.Memory.MaxPromptTokens != 2000 {
		t.Errorf("max_prompt_tokens = %d, want 2000", cfg.Memory.MaxPromptTokens)
	}
	if cfg.Language == "" {
		t.Error("language should have a default value")
	}
	if cfg.Heartbeat.Enabled {
		t.Error("heartbeat should be disabled by default")
	}
	if cfg.Heartbeat.Interval != "4h" {
		t.Errorf("heartbeat interval = %q, want 4h", cfg.Heartbeat.Interval)
	}
}

func TestLoadMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("STEFANCLAW_CONFIG_DIR", tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Should return defaults when config file doesn't exist
	if cfg.Provider.Default != "ollama" {
		t.Errorf("provider.default = %q, want ollama", cfg.Provider.Default)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("STEFANCLAW_CONFIG_DIR", tmp)

	cfg := Defaults()
	cfg.Model.Default = "llama3"

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(tmp, "config.yaml")); err != nil {
		t.Fatalf("config.yaml not created: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.Model.Default != "llama3" {
		t.Errorf("loaded model = %q, want llama3", loaded.Model.Default)
	}
}

func TestIsFirstRun(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("STEFANCLAW_CONFIG_DIR", tmp)

	if !IsFirstRun() {
		t.Error("IsFirstRun() = false, want true (no config.yaml)")
	}

	if err := Save(Defaults()); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if IsFirstRun() {
		t.Error("IsFirstRun() = true, want false (config.yaml exists)")
	}
}
