package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stefanclaw/stefanclaw/internal/config"
	"github.com/stefanclaw/stefanclaw/internal/fetch"
	"github.com/stefanclaw/stefanclaw/internal/memory"
	"github.com/stefanclaw/stefanclaw/internal/onboard"
	"github.com/stefanclaw/stefanclaw/internal/prompt"
	"github.com/stefanclaw/stefanclaw/internal/provider"
	"github.com/stefanclaw/stefanclaw/internal/provider/ollama"
	"github.com/stefanclaw/stefanclaw/internal/session"
	"github.com/stefanclaw/stefanclaw/internal/tui"
	"github.com/stefanclaw/stefanclaw/internal/update"
)

var version = "dev"

func main() {
	// Parse --ollama-url and --pipe flags from args
	var ollamaURL string
	var pipeMode bool
	filteredArgs := []string{os.Args[0]}
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--ollama-url" && i+1 < len(os.Args) {
			ollamaURL = os.Args[i+1]
			i++ // skip the value
		} else if os.Args[i] == "--pipe" {
			pipeMode = true
		} else {
			filteredArgs = append(filteredArgs, os.Args[i])
		}
	}
	os.Args = filteredArgs

	// Fall back to OLLAMA_HOST env var
	if ollamaURL == "" {
		ollamaURL = os.Getenv("OLLAMA_HOST")
	}

	if !pipeMode && len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("stefanclaw %s\n", version)
			return
		case "--help", "-h":
			printHelp()
			return
		case "--uninstall":
			runUninstall()
			return
		case "--update":
			runUpdate()
			return
		}
	}

	if pipeMode {
		// Collect remaining args as the question
		question := strings.Join(os.Args[1:], " ")
		if err := runPipe(ollamaURL, question); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := run(ollamaURL); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ollamaURL string) error {
	// First run: onboarding
	if config.IsFirstRun() {
		runner := onboard.NewRunner()
		if ollamaURL != "" {
			runner.BaseURL = ollamaURL
		}
		result, err := runner.Run()
		if err != nil {
			return err
		}
		_ = result
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// CLI flag / env var override config file
	if ollamaURL != "" {
		cfg.Provider.Ollama.BaseURL = ollamaURL
	}

	// Create Ollama provider
	ollamaProvider := ollama.New(cfg.Provider.Ollama.BaseURL)

	// Check availability
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ollamaProvider.IsAvailable(ctx); err != nil {
		fmt.Println("\nOllama is not running.")
		fmt.Println("Start it with: ollama serve")
		fmt.Println("Then re-run stefanclaw.")
		return err
	}

	// Build system prompt
	personalityDir := config.PersonalityDir()
	asm := prompt.NewAssembler(personalityDir)
	asm.LoadFiles()
	systemPrompt := asm.BuildSystemPromptWithLanguage(cfg.Language)

	// Initialize session store
	sessStore := session.NewFileStore(config.SessionsDir())

	// Get or create current session
	sess, err := sessStore.Current()
	if err != nil {
		return fmt.Errorf("loading current session: %w", err)
	}
	if sess == nil {
		sess, err = sessStore.Create("New Chat", cfg.Model.Default)
		if err != nil {
			return fmt.Errorf("creating session: %w", err)
		}
		sessStore.SetCurrent(sess.ID)
	}

	// Load conversation history from transcript
	history, _ := sessStore.LoadTranscript(sess.ID)

	// Initialize memory store
	memStore := memory.NewStore(config.PersonalityDir() + "/MEMORY.md")

	// Start TUI
	tuiModel := tui.New(tui.Options{
		Provider:       ollamaProvider,
		SessionStore:   sessStore,
		MemoryStore:    memStore,
		PromptAsm:      asm,
		SystemPrompt:   systemPrompt,
		Model:          cfg.Model.Default,
		Session:        sess,
		PersonalityDir: personalityDir,
		Language:       cfg.Language,
		Heartbeat:      cfg.Heartbeat,
		MaxNumCtx:      cfg.Provider.Ollama.MaxNumCtx,
		Version:        version,
		History:        history,
	})

	p := tea.NewProgram(tuiModel, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func runPipe(ollamaURL, question string) error {
	// Pipe mode requires config to exist already (no onboarding)
	if config.IsFirstRun() {
		return fmt.Errorf("no config found — run stefanclaw interactively first to complete onboarding")
	}

	// Read question from stdin if not provided as args
	question = strings.TrimSpace(question)
	if question == "" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		question = strings.TrimSpace(string(data))
	}
	if question == "" {
		return fmt.Errorf("no question provided — pass it as arguments or pipe to stdin")
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if ollamaURL != "" {
		cfg.Provider.Ollama.BaseURL = ollamaURL
	}

	// Create Ollama provider and check availability
	ollamaProvider := ollama.New(cfg.Provider.Ollama.BaseURL)
	ctx := context.Background()
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := ollamaProvider.IsAvailable(checkCtx); err != nil {
		return fmt.Errorf("ollama is not running (start with: ollama serve): %w", err)
	}

	// Build system prompt
	personalityDir := config.PersonalityDir()
	asm := prompt.NewAssembler(personalityDir)
	asm.LoadFiles()
	systemPrompt := asm.BuildSystemPromptWithLanguage(cfg.Language)

	// Auto-fetch URLs in the question
	fetchClient := fetch.New()
	augmented := fetch.AugmentWithWebContent(ctx, fetchClient, question)

	// Build messages
	var msgs []provider.Message
	if systemPrompt != "" {
		msgs = append(msgs, provider.Message{Role: "system", Content: systemPrompt})
	}
	msgs = append(msgs, provider.Message{Role: "user", Content: augmented})

	// Call the model (non-streaming, blocking)
	resp, err := ollamaProvider.Chat(ctx, provider.ChatRequest{
		Model:    cfg.Model.Default,
		Messages: msgs,
	})
	if err != nil {
		return fmt.Errorf("chat: %w", err)
	}

	fmt.Println(resp.Message.Content)
	return nil
}

func runUpdate() {
	if version == "dev" {
		fmt.Println("Auto-update is not available for development builds.")
		return
	}
	fmt.Println("Checking for updates...")
	res, err := update.Apply(context.Background(), version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
		os.Exit(1)
	}
	if res.Applied {
		fmt.Printf("Updated to v%s. Restart stefanclaw to use the new version.\n", res.LatestVersion)
	} else {
		fmt.Println("Already running the latest version.")
	}
}

func runUninstall() {
	configDir := config.Dir()
	fmt.Println("Stefanclaw Uninstall")
	fmt.Println("====================")
	fmt.Println("")
	fmt.Println("This will remove all stefanclaw data:")
	fmt.Printf("  Config & data: %s\n", configDir)
	fmt.Println("")
	fmt.Print("Are you sure? (y/N) ")

	var answer string
	fmt.Scanln(&answer)
	if answer != "y" && answer != "Y" {
		fmt.Println("Cancelled.")
		return
	}

	if err := os.RemoveAll(configDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing %s: %v\n", configDir, err)
		os.Exit(1)
	}
	fmt.Printf("Removed %s\n", configDir)

	// Find and report binary location
	exe, err := os.Executable()
	if err == nil {
		fmt.Printf("\nTo complete removal, delete the binary:\n  rm %s\n", exe)
	}
	fmt.Println("\nStefanclaw has been uninstalled.")
}

func printHelp() {
	fmt.Printf(`stefanclaw %s — your personal AI assistant

Usage:
  stefanclaw                          Start the TUI chat interface
  stefanclaw --pipe "question"        Non-interactive mode (prints response to stdout)
  stefanclaw --ollama-url <url>       Use a custom Ollama endpoint
  stefanclaw --version                Print version and exit
  stefanclaw --help                   Show this help
  stefanclaw --update                 Update to the latest version
  stefanclaw --uninstall              Remove all stefanclaw data from your system

Slash commands (in TUI):
  /help                Show available commands
  /quit, /bye, /exit   Exit stefanclaw
  /models              List available Ollama models
  /model <name>        Switch model
  /session new         Start a new session
  /session list        List all sessions
  /clear               Clear conversation display
  /memory              Show memory entries
  /remember <fact>     Save a fact to memory
  /forget <keyword>    Remove matching memory entries
  /language [<name>]   Show or change response language
  /heartbeat [on|off|<interval>]  Manage heartbeat check-ins
  /personality edit    Open personality files for editing
  /update              Check for updates and upgrade

Configuration:
  Config is stored in %s
  Override with STEFANCLAW_CONFIG_DIR environment variable.

Ollama endpoint (priority: flag > env > config > default):
  --ollama-url <url>   Override the Ollama base URL
  OLLAMA_HOST          Environment variable (matches Ollama's own convention)

Requires:
  Ollama running locally or at the specified endpoint (https://ollama.ai)

Pipe mode (non-interactive, for scripting):
  stefanclaw --pipe "What is 2+2?"                          Question as argument
  echo "What is 2+2?" | stefanclaw --pipe                   Question from stdin
  stefanclaw --pipe "Summarize https://example.com" | pbcopy  Pipe into other tools

Examples:
  stefanclaw                                                Start chatting
  stefanclaw --pipe "Hello, who are you?"                   Non-interactive query
  stefanclaw --ollama-url http://192.168.1.100:11434        Use remote Ollama
  OLLAMA_HOST=http://192.168.1.100:11434 stefanclaw         Same via env var
  STEFANCLAW_CONFIG_DIR=/tmp/test stefanclaw                Use custom config dir
`, version, config.Dir())
}
