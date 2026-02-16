package onboard

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/stefanclaw/stefanclaw/internal/config"
	"github.com/stefanclaw/stefanclaw/internal/prompt"
	"github.com/stefanclaw/stefanclaw/internal/provider/ollama"
)

// Result holds the outcome of the onboarding flow.
type Result struct {
	Config config.Config
	Model  string
}

// Runner encapsulates onboarding dependencies for testability.
type Runner struct {
	Stdin   io.Reader
	Stdout  io.Writer
	BaseURL string
}

// NewRunner creates a Runner with default stdin/stdout.
func NewRunner() *Runner {
	return &Runner{
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		BaseURL: "http://127.0.0.1:11434",
	}
}

// Run executes the first-run onboarding flow.
func (r *Runner) Run() (*Result, error) {
	w := r.Stdout

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  Welcome to stefanclaw!")
	fmt.Fprintln(w, "  Your personal AI assistant.")
	fmt.Fprintln(w, "")

	// Step 1: Check Ollama
	fmt.Fprint(w, "  Checking for Ollama... ")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ollama.Detect(ctx, r.BaseURL); err != nil {
		fmt.Fprintln(w, "not found.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "  Ollama is not running. Please install and start it:")
		fmt.Fprintln(w, "    1. Install from https://ollama.ai")
		fmt.Fprintln(w, "    2. Run: ollama serve")
		fmt.Fprintln(w, "    3. Then re-run stefanclaw")
		return nil, fmt.Errorf("ollama not running at %s", r.BaseURL)
	}
	fmt.Fprintln(w, "found!")

	// Step 2: List models
	provider := ollama.New(r.BaseURL)
	models, err := provider.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing models: %w", err)
	}

	if len(models) == 0 {
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "  No models found. Pull one with:")
		fmt.Fprintln(w, "    ollama pull qwen3-next")
		return nil, fmt.Errorf("no models found")
	}

	// Pick default model
	selectedModel := models[0].Name
	for _, m := range models {
		if strings.Contains(m.Name, "qwen3") {
			selectedModel = m.Name
			break
		}
	}
	fmt.Fprintf(w, "  Found %d model(s). Using: %s\n", len(models), selectedModel)

	// Step 3: Create config directory
	fmt.Fprint(w, "  Creating config directory... ")
	configDir := config.Dir()
	if err := os.MkdirAll(config.PersonalityDir(), 0o755); err != nil {
		fmt.Fprintln(w, "failed.")
		return nil, fmt.Errorf("creating config directory: %w", err)
	}
	if err := os.MkdirAll(config.SessionsDir(), 0o755); err != nil {
		return nil, fmt.Errorf("creating sessions directory: %w", err)
	}
	fmt.Fprintln(w, "done.")
	fmt.Fprintf(w, "  Config: %s\n", configDir)

	// Step 4: Copy personality files
	fmt.Fprint(w, "  Copying personality templates... ")
	for _, name := range prompt.AllSections {
		content, err := prompt.EmbeddedDefault(name)
		if err != nil {
			continue
		}
		path := config.PersonalityDir() + "/" + name
		os.WriteFile(path, []byte(content), 0o644)
	}
	fmt.Fprintln(w, "done.")

	// Step 5: Ask user's name
	fmt.Fprintln(w, "")
	fmt.Fprint(w, "  What's your name? ")
	scanner := bufio.NewScanner(r.Stdin)
	var userName string
	if scanner.Scan() {
		userName = strings.TrimSpace(scanner.Text())
	}
	if userName == "" {
		userName = "friend"
	}

	// Update USER.md
	userContent := fmt.Sprintf("# User\n\n- Name: %s\n", userName)
	os.WriteFile(config.PersonalityDir()+"/USER.md", []byte(userContent), 0o644)

	// Step 6: Save config
	cfg := config.Defaults()
	cfg.Provider.Ollama.BaseURL = r.BaseURL
	cfg.Model.Default = selectedModel

	if err := config.Save(cfg); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}

	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "  Setup complete! Hello, %s.\n", userName)
	fmt.Fprintln(w, "  Starting stefanclaw...")

	return &Result{
		Config: cfg,
		Model:  selectedModel,
	}, nil
}
