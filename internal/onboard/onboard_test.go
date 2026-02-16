package onboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stefanclaw/stefanclaw/internal/config"
)

func setupTestEnv(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("STEFANCLAW_CONFIG_DIR", tmp)
	return tmp
}

func TestIsFirstRun_NoConfig(t *testing.T) {
	setupTestEnv(t)
	if !config.IsFirstRun() {
		t.Error("should be first run with no config")
	}
}

func TestSetup_CreatesConfigDir(t *testing.T) {
	tmp := setupTestEnv(t)

	srv := newMockOllama(t, []string{"qwen3-next"})
	defer srv.Close()

	r := &Runner{
		Stdin:   strings.NewReader("\n"),
		Stdout:  &bytes.Buffer{},
		BaseURL: srv.URL,
	}

	result, err := r.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Check config dir exists
	if _, err := os.Stat(filepath.Join(tmp, "personality")); err != nil {
		t.Errorf("personality dir not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "sessions")); err != nil {
		t.Errorf("sessions dir not created: %v", err)
	}
	if result.Model != "qwen3-next" {
		t.Errorf("model = %q, want qwen3-next", result.Model)
	}
}

func TestSetup_CopiesPersonalityFiles(t *testing.T) {
	tmp := setupTestEnv(t)

	srv := newMockOllama(t, []string{"llama3"})
	defer srv.Close()

	r := &Runner{
		Stdin:   strings.NewReader("\n"),
		Stdout:  &bytes.Buffer{},
		BaseURL: srv.URL,
	}

	_, err := r.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	files := []string{"IDENTITY.md", "SOUL.md", "USER.md", "MEMORY.md", "BOOT.md", "BOOTSTRAP.md"}
	for _, f := range files {
		path := filepath.Join(tmp, "personality", f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("personality file %s not copied: %v", f, err)
		}
	}

	// Check USER.md has language
	data, _ := os.ReadFile(filepath.Join(tmp, "personality", "USER.md"))
	if !strings.Contains(string(data), "Language:") {
		t.Error("USER.md should contain language")
	}
}

func TestSetup_DetectsOllama(t *testing.T) {
	setupTestEnv(t)

	srv := newMockOllama(t, []string{"qwen3-next", "llama3"})
	defer srv.Close()

	out := &bytes.Buffer{}
	r := &Runner{
		Stdin:   strings.NewReader("\n"),
		Stdout:  out,
		BaseURL: srv.URL,
	}

	_, err := r.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if !strings.Contains(out.String(), "found!") {
		t.Error("output should say Ollama was found")
	}
}

func TestSetup_OllamaNotFound(t *testing.T) {
	setupTestEnv(t)

	out := &bytes.Buffer{}
	r := &Runner{
		Stdin:   strings.NewReader("User\n"),
		Stdout:  out,
		BaseURL: "http://127.0.0.1:1",
	}

	_, err := r.Run()
	if err == nil {
		t.Fatal("Run() should error when Ollama is not running")
	}

	if !strings.Contains(out.String(), "not found") || !strings.Contains(out.String(), "ollama serve") {
		t.Error("output should suggest running ollama serve")
	}
}

func TestSetup_NoModels(t *testing.T) {
	setupTestEnv(t)

	srv := newMockOllama(t, nil) // no models
	defer srv.Close()

	out := &bytes.Buffer{}
	r := &Runner{
		Stdin:   strings.NewReader("User\n"),
		Stdout:  out,
		BaseURL: srv.URL,
	}

	_, err := r.Run()
	if err == nil {
		t.Fatal("Run() should error when no models are found")
	}

	if !strings.Contains(out.String(), "ollama pull") {
		t.Error("output should suggest pulling a model")
	}
}

func TestSetup_LanguageDetectedAndStored(t *testing.T) {
	tmp := setupTestEnv(t)
	t.Setenv("LANG", "de_DE.UTF-8")
	t.Setenv("LC_ALL", "")
	t.Setenv("LANGUAGE", "")

	srv := newMockOllama(t, []string{"qwen3-next"})
	defer srv.Close()

	out := &bytes.Buffer{}
	r := &Runner{
		Stdin:   strings.NewReader("\n"), // empty line = accept detected language
		Stdout:  out,
		BaseURL: srv.URL,
	}

	result, err := r.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Config should have detected language
	if result.Config.Language != "Deutsch" {
		t.Errorf("config.Language = %q, want Deutsch", result.Config.Language)
	}

	// USER.md should contain language
	data, _ := os.ReadFile(filepath.Join(tmp, "personality", "USER.md"))
	if !strings.Contains(string(data), "Language: Deutsch") {
		t.Errorf("USER.md should contain 'Language: Deutsch', got:\n%s", string(data))
	}
}

func TestSetup_LanguageCustom(t *testing.T) {
	tmp := setupTestEnv(t)

	srv := newMockOllama(t, []string{"qwen3-next"})
	defer srv.Close()

	r := &Runner{
		Stdin:   strings.NewReader("Français\n"),
		Stdout:  &bytes.Buffer{},
		BaseURL: srv.URL,
	}

	result, err := r.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if result.Config.Language != "Français" {
		t.Errorf("config.Language = %q, want Français", result.Config.Language)
	}

	data, _ := os.ReadFile(filepath.Join(tmp, "personality", "USER.md"))
	if !strings.Contains(string(data), "Language: Français") {
		t.Errorf("USER.md should contain 'Language: Français', got:\n%s", string(data))
	}
}

// newMockOllama creates a test server mimicking Ollama's /api/tags endpoint.
func newMockOllama(t *testing.T, modelNames []string) *httptest.Server {
	t.Helper()

	type model struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	}
	type response struct {
		Models []model `json:"models"`
	}

	var models []model
	for _, name := range modelNames {
		models = append(models, model{Name: name, Size: 4000000000})
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			json.NewEncoder(w).Encode(response{Models: models})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}
