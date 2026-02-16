# stefanclaw

A personal AI assistant that ships as a single Go binary. Chat with local LLMs via Ollama through a terminal UI with personality, memory, and session management.

## Quickstart

```bash
# Prerequisites: Ollama running locally
ollama serve

# Build and run
make build
./stefanclaw
```

On first run, an onboarding wizard configures your setup.

## Features

- TUI chat interface with streaming responses and markdown rendering
- Ollama as the LLM backend
- Personality system (IDENTITY, SOUL, USER, MEMORY, BOOT, BOOTSTRAP)
- Persistent memory with automatic fact extraction
- Session management with JSONL transcripts
- Conversation compaction for long chats
- First-run onboarding wizard
- Slash commands: `/help`, `/models`, `/model`, `/session`, `/memory`, `/remember`, `/forget`, `/clear`, `/personality edit`

## Architecture

```
cmd/stefanclaw/     Entry point, wiring, CLI flags
internal/
  config/           YAML config, paths
  prompt/           Personality file loader, system prompt assembler
  provider/ollama/  Ollama REST API client (streaming + blocking)
  session/          Session store, JSONL transcripts, compaction
  memory/           Persistent memory, fact extraction, search
  onboard/          First-run wizard
  tui/              Bubble Tea terminal UI with markdown rendering
  channel/          Channel interface (future: Telegram, etc.)
personality/        Default personality templates (embedded)
```

## Development

```bash
make test    # Run tests (67 tests across all packages)
make build   # Build binary
make lint    # Run go vet
make clean   # Remove binary
```

## Configuration

Config lives in `~/.config/stefanclaw/`. Override with `STEFANCLAW_CONFIG_DIR`.

## Uninstall

To completely remove stefanclaw from your system:

```bash
# Interactive: removes config dir, tells you where to delete the binary
./stefanclaw --uninstall

# Or manually:
rm -rf ~/.config/stefanclaw   # Remove all config, sessions, memory, personality
rm ./stefanclaw                # Remove the binary (or wherever you placed it)
```

This removes:
- `~/.config/stefanclaw/config.yaml` - configuration
- `~/.config/stefanclaw/personality/` - personality files (IDENTITY, SOUL, USER, MEMORY, BOOT, BOOTSTRAP)
- `~/.config/stefanclaw/sessions/` - all conversation history
- The binary itself (you must delete it manually)

## License

MIT - Copyright 2026 Stefan Wintermeyer
