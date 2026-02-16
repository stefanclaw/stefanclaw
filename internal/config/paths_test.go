package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDir_Default(t *testing.T) {
	// Unset override env var
	os.Unsetenv("STEFANCLAW_CONFIG_DIR")

	dir := Dir()
	if !strings.HasSuffix(dir, filepath.Join(".config", "stefanclaw")) {
		t.Errorf("Dir() = %q, want suffix .config/stefanclaw", dir)
	}
}

func TestDir_EnvOverride(t *testing.T) {
	t.Setenv("STEFANCLAW_CONFIG_DIR", "/tmp/test-stefanclaw")
	defer os.Unsetenv("STEFANCLAW_CONFIG_DIR")

	dir := Dir()
	if dir != "/tmp/test-stefanclaw" {
		t.Errorf("Dir() = %q, want /tmp/test-stefanclaw", dir)
	}
}

func TestPersonalityDir(t *testing.T) {
	os.Unsetenv("STEFANCLAW_CONFIG_DIR")
	dir := PersonalityDir()
	if !strings.HasSuffix(dir, filepath.Join("stefanclaw", "personality")) {
		t.Errorf("PersonalityDir() = %q, want suffix stefanclaw/personality", dir)
	}
}

func TestSessionsDir(t *testing.T) {
	os.Unsetenv("STEFANCLAW_CONFIG_DIR")
	dir := SessionsDir()
	if !strings.HasSuffix(dir, filepath.Join("stefanclaw", "sessions")) {
		t.Errorf("SessionsDir() = %q, want suffix stefanclaw/sessions", dir)
	}
}

func TestConfigFile(t *testing.T) {
	os.Unsetenv("STEFANCLAW_CONFIG_DIR")
	f := ConfigFile()
	if !strings.HasSuffix(f, filepath.Join("stefanclaw", "config.yaml")) {
		t.Errorf("ConfigFile() = %q, want suffix stefanclaw/config.yaml", f)
	}
}
