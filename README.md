# stefanclaw

I found it too hard to install [OpenClaw](https://github.com/openclaw/openclaw), so portted the core functionality to Go. The result is a single binary you can install by just copying one file. To make it easy for everybody to play with, it uses a local [Ollama](https://ollama.com) instance — no API keys, no cloud, everything runs on your machine. Just a plain old TUI!

OpenClaw has way more features and probably a brighter future since it is foreseeable that VC or Meta money is flowing in that direction. Check out [openclaw.ai](https://openclaw.ai) for the full project. But do not hesitate to contact me in case you want this project to be continued. I believe that the usability of OpenClaw has a lot of room for improvement and that usability is paramount. BTW: I didn't have time to search for a better name.

**Warning:** This is beta software at best and potentially dangerous to use! No warranties, no guarantees — use at your own risk.

## Installation

Download the latest binary for your platform:

```bash
# macOS (Apple Silicon)
curl -sL https://github.com/stefanclaw/stefanclaw/releases/latest/download/stefanclaw_darwin_arm64.tar.gz | tar xz
./stefanclaw

# macOS (Intel)
curl -sL https://github.com/stefanclaw/stefanclaw/releases/latest/download/stefanclaw_darwin_amd64.tar.gz | tar xz
./stefanclaw

# Linux (x86_64)
curl -sL https://github.com/stefanclaw/stefanclaw/releases/latest/download/stefanclaw_linux_amd64.tar.gz | tar xz
./stefanclaw

# Linux (ARM64)
curl -sL https://github.com/stefanclaw/stefanclaw/releases/latest/download/stefanclaw_linux_arm64.tar.gz | tar xz
./stefanclaw
```

**Windows:** Download the `.zip` from the [latest release](https://github.com/stefanclaw/stefanclaw/releases/latest), extract it, and add the folder to your PATH.

**All releases:** [github.com/stefanclaw/stefanclaw/releases/latest](https://github.com/stefanclaw/stefanclaw/releases/latest)

### Build from source

```bash
git clone https://github.com/stefanclaw/stefanclaw.git
cd stefanclaw
make build
./stefanclaw
```

Requires [Go](https://go.dev/) 1.21+.

### Prerequisites

[Ollama](https://ollama.ai) must be running locally (or on a reachable host):

```bash
ollama serve
```

To use a remote Ollama instance:
```bash
stefanclaw --ollama-url http://192.168.1.100:11434
# or
OLLAMA_HOST=http://192.168.1.100:11434 stefanclaw
```

Priority: `--ollama-url` flag > `OLLAMA_HOST` env var > `config.yaml` > default (`http://127.0.0.1:11434`).

On first run, an onboarding wizard configures your setup (name, language, model).

## Features

- TUI chat interface with streaming responses and markdown rendering
- Ollama as the LLM backend
- Personality system (IDENTITY, SOUL, USER, MEMORY, BOOT, HEARTBEAT, BOOTSTRAP)
- Persistent memory with automatic fact extraction
- Session management with JSONL transcripts
- Conversation compaction for long chats
- First-run onboarding wizard
- **Language support** — auto-detects system locale, asks during onboarding, LLM responds in your language
- **Heartbeat check-ins** — configurable periodic proactive messages when idle
- **Adaptive context scaling** — starts with 4K context, automatically grows to 8K/16K/32K as conversations get longer
- **Web fetch** — fetch any web page as markdown via Jina Reader
- **Web search** — search the web via DuckDuckGo (no API key needed)
- **Auto-update** — checks for updates on startup, upgrade in-place with `/update` or `--update`
- Slash commands: `/help`, `/quit`, `/bye`, `/exit`, `/models`, `/model`, `/session`, `/memory`, `/remember`, `/forget`, `/clear`, `/language`, `/heartbeat`, `/fetch`, `/search`, `/personality edit`, `/update`

## Language Support

Stefanclaw detects your system language from `LC_ALL`, `LANG`, or `LANGUAGE` environment variables. During onboarding, you can accept the detected language or choose a different one. The LLM will respond in your chosen language.

- `/language` — show current language
- `/language Deutsch` — switch to German

## Heartbeat

Heartbeat check-ins are periodic proactive messages from the assistant when you've been idle. The assistant reviews memory and conversation context, and speaks up only if there's something relevant.

- `/heartbeat` — show status and interval
- `/heartbeat on` — enable heartbeats
- `/heartbeat off` — disable heartbeats
- `/heartbeat 2h` — set interval to 2 hours

Configure in `config.yaml`:
```yaml
heartbeat:
  enabled: false
  interval: "4h"
```

## Web Fetch

Fetch any web page and display it as markdown directly in the chat. Powered by [Jina Reader](https://r.jina.ai/) — no API key needed (free tier: 100 RPM). Content is capped at 32KB.

- `/fetch https://example.com` — fetch and display a page

## Web Search

Search the web directly from the chat. Powered by DuckDuckGo routed through Jina Reader — no API key needed.

- `/search capital of france` — search and display results

## Updating

Stefanclaw checks for updates on startup and notifies you when a new version is available.

- `/update` — download and install the latest version (in TUI)
- `stefanclaw --update` — update from the command line

After updating, restart stefanclaw to use the new version.

## Adaptive Context Scaling

Ollama defaults to 4096 tokens of context (`num_ctx`). Stefanclaw automatically scales the context window as conversations grow, to avoid wasting VRAM on short chats while supporting longer ones.

| Tier | Context size | Triggers when |
|------|-------------|---------------|
| 1 | 4096 | Initial |
| 2 | 8192 | Prompt tokens exceed 60% of current size |
| 3 | 16384 | Prompt tokens exceed 60% of current size |
| 4 | 32768 | Prompt tokens exceed 60% of current size |

When the context grows, a system message appears and the model reloads briefly (a few seconds). Configure the upper limit in `config.yaml`:

```yaml
provider:
  ollama:
    max_num_ctx: 32768
```

## Architecture

```
cmd/stefanclaw/     Entry point, wiring, CLI flags
internal/
  config/           YAML config, paths, locale detection
  fetch/            Web fetch via Jina Reader
  prompt/           Personality file loader, system prompt assembler
  provider/ollama/  Ollama REST API client (streaming + blocking)
  session/          Session store, JSONL transcripts, compaction
  memory/           Persistent memory, fact extraction, search
  onboard/          First-run wizard
  tui/              Bubble Tea terminal UI, command registry, handlers
  update/           Auto-update via GitHub Releases
  channel/          Channel interface (future: Telegram, etc.)
personality/        Default personality templates (embedded)
```

## Development

```bash
make test    # Run tests
make build   # Build binary
make lint    # Run go vet
make clean   # Remove binary
```

Build from source:
```bash
git clone https://github.com/stefanclaw/stefanclaw.git
cd stefanclaw
make build
./stefanclaw
```

### Releasing

Releases are automated via GoReleaser and GitHub Actions. To create a new release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

This triggers the release workflow, which builds binaries for all platforms and creates a GitHub Release.

To test the release process locally:
```bash
make release-dry-run
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
- `~/.config/stefanclaw/personality/` - personality files (IDENTITY, SOUL, USER, MEMORY, BOOT, HEARTBEAT, BOOTSTRAP)
- `~/.config/stefanclaw/sessions/` - all conversation history
- The binary itself (you must delete it manually)

## License

MIT - Copyright 2026 Stefan Wintermeyer
