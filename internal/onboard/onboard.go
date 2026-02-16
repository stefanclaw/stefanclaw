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
		fmt.Fprintln(w, "    ollama pull qwen3:8b")
		return nil, fmt.Errorf("no models found")
	}

	// Filter qwen3 models
	var qwen3Models []string
	for _, m := range models {
		if strings.Contains(m.Name, "qwen3") {
			qwen3Models = append(qwen3Models, m.Name)
		}
	}

	scanner := bufio.NewScanner(r.Stdin)
	var selectedModel string

	if len(qwen3Models) > 0 {
		fmt.Fprintf(w, "  Found %d qwen3 model(s):\n", len(qwen3Models))
		fmt.Fprintln(w, "")

		// Determine default: prefer qwen3:8b, otherwise first qwen3 model
		defaultModel := qwen3Models[0]
		for _, name := range qwen3Models {
			if name == "qwen3:8b" {
				defaultModel = name
				break
			}
		}

		for i, name := range qwen3Models {
			marker := "  "
			if name == defaultModel {
				marker = "* "
			}
			fmt.Fprintf(w, "  %s%d) %s\n", marker, i+1, name)
		}
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "  Tip: Smaller models (e.g. 1b, 4b) are faster but less capable.")
		fmt.Fprintln(w, "       Larger models (e.g. 8b, 14b) are slower but produce better results.")
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "  Select a model [%s]: ", defaultModel)
		var choice string
		if scanner.Scan() {
			choice = strings.TrimSpace(scanner.Text())
		}

		if choice == "" {
			selectedModel = defaultModel
		} else {
			// Check if user entered a number
			found := false
			for i, name := range qwen3Models {
				if choice == fmt.Sprintf("%d", i+1) {
					selectedModel = name
					found = true
					break
				}
			}
			if !found {
				// Treat as a literal model name
				selectedModel = choice
			}
		}
	} else {
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "  No qwen3 models found. The recommended model is qwen3:8b.")
		fmt.Fprintln(w, "  Install it with:")
		fmt.Fprintln(w, "    ollama pull qwen3:8b")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "  Tip: Smaller models (e.g. 1b, 4b) are faster but less capable.")
		fmt.Fprintln(w, "       Larger models (e.g. 8b, 14b) are slower but produce better results.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "  Available models:")
		for _, m := range models {
			fmt.Fprintf(w, "    - %s\n", m.Name)
		}
		fmt.Fprintln(w, "")
		fmt.Fprint(w, "  Enter a model name to use (or press Enter to abort): ")
		var choice string
		if scanner.Scan() {
			choice = strings.TrimSpace(scanner.Text())
		}
		if choice == "" {
			return nil, fmt.Errorf("no qwen3 model available — install one with: ollama pull qwen3:8b")
		}
		selectedModel = choice
	}

	fmt.Fprintf(w, "  Using model: %s\n", selectedModel)

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

	// Step 5: Ask preferred language
	fmt.Fprintln(w, "")
	detectedLang := config.DetectLanguage()
	fmt.Fprintf(w, "  What language should I use? [%s] ", detectedLang)
	var language string
	if scanner.Scan() {
		language = strings.TrimSpace(scanner.Text())
	}
	if language == "" {
		language = detectedLang
	}

	// Update USER.md
	userContent := fmt.Sprintf("# User\n\n- Language: %s\n", language)
	os.WriteFile(config.PersonalityDir()+"/USER.md", []byte(userContent), 0o644)

	// Step 7: Save config
	cfg := config.Defaults()
	cfg.Provider.Ollama.BaseURL = r.BaseURL
	cfg.Model.Default = selectedModel
	cfg.Language = language

	if err := config.Save(cfg); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  Setup complete!")
	fmt.Fprintln(w, "  Starting stefanclaw...")

	return &Result{
		Config: cfg,
		Model:  selectedModel,
	}, nil
}
