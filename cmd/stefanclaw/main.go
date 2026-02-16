package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stefanclaw/stefanclaw/internal/config"
	"github.com/stefanclaw/stefanclaw/internal/memory"
	"github.com/stefanclaw/stefanclaw/internal/onboard"
	"github.com/stefanclaw/stefanclaw/internal/prompt"
	"github.com/stefanclaw/stefanclaw/internal/provider/ollama"
	"github.com/stefanclaw/stefanclaw/internal/session"
	"github.com/stefanclaw/stefanclaw/internal/tui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
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
		}
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// First run: onboarding
	if config.IsFirstRun() {
		runner := onboard.NewRunner()
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
	})

	p := tea.NewProgram(tuiModel, tea.WithAltScreen())
	_, err = p.Run()
	return err
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
  stefanclaw              Start the TUI chat interface
  stefanclaw --version    Print version and exit
  stefanclaw --help       Show this help
  stefanclaw --uninstall  Remove all stefanclaw data from your system

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

Configuration:
  Config is stored in %s
  Override with STEFANCLAW_CONFIG_DIR environment variable.

Requires:
  Ollama running locally (https://ollama.ai)

Examples:
  stefanclaw                          Start chatting
  STEFANCLAW_CONFIG_DIR=/tmp/test stefanclaw  Use custom config dir
`, version, config.Dir())
}
