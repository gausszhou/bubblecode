# bubblecode

TUI for AI agents via the [ACP protocol](https://github.com/coder/acp-go-sdk).

Built with [Bubble Tea v2](charm.land/bubbletea/v2), [Lip Gloss v2](charm.land/lipgloss/v2).

## Build & Run

```bash
go build -o bin/bubblecode .                        # single binary
CGO_ENABLED=0 go build -ldflags="-s -w" .            # production build
go run .                                             # requires config
BUBBLECODE_API_KEY=sk-... go run .                   # quick start with env
go run . --debug                                     # log to ~/.gausszhou/bubblecode/logs/
go install .                                         # install to $GOPATH/bin
```

The Makefile offers cross-compilation targets (`build-linux`, `build-darwin`, `build-windows`, `build-all`) and `package` for release tarballs:

```bash
make fmt          # go fmt ./...
make vet          # go vet ./...
make lint         # golangci-lint run ./...
make test         # go test ./...
make build-all    # cross-compile for all platforms
make package      # build + tar.gz per platform
make clean        # rm -rf bin/ dist/
```

## Architecture

```
main.go → cmd/bubblecode/ (cobra CLI)
            ├── chat       → TUI (default)
            │     spawns subprocess: bubblecode acp
            │     ↓ stdin/stdout pipes
            │   client/ → ACP client (sends/receives commands)
            │        ↓ channels
            │     tui/ → Bubble Tea app (chat viewport + sidebar)
            ├── acp         → ACP server (stdio mode, spawned by `chat`)
            │     uses agent/ package (LLM client, tools, config)
            ├── providers   → manage API provider configs
            └── models      → manage model configs
```

Key directories:
- `cmd/bubblecode/` — cobra commands: `chat.go`, `acp.go`, `providers.go`, `models.go`
- `client/` — ACP protocol client (`ACPClient`, `PromptRunner`, `Connection`)
- `tui/` — Bubble Tea model/view/update + `layout/`, `overlay/`, `component/`, `theme/`
- `agent/` — importable package: LLM client, tool executor (read_file, write_file, bash, glob), config
- `docs/` — design guides, best practices
- `examples/` — Bubble Tea experimental code (markdown, streaming, viewports, etc.)

## Config

File: `~/.config/bubblecode/config.json` (multiple providers, each with models)

Env: `BUBBLECODE_API_KEY` — fallback for default provider's API key.

Presets: `deepseek` (`api.deepseek.com`) and `gausszhou` (`mock.gausszhou.top`).

Legacy single-key config is auto-migrated on load.

## Logger

Writes per-component files to `~/.gausszhou/bubblecode/logs/`:
- `client.log` — TUI client
- `agent.log` — ACP agent server
- `change.log` — event collector diagnostics

## Agent Tools

The LLM agent exposes 4 tools: `read_file`, `write_file`, `bash`, `glob`.
- `bash` runs `sh -c`, default 30s timeout, configurable via `timeout` param.
- Paths are resolved relative to session `cwd` and sandboxed (can't escape working directory).
- Agent can switch model via `/model <id>` and provider via `/provider <name>` in-chat commands.

## Key Bindings

| Key | Action |
|---|---|
| `Enter` | Send message |
| `Shift+Enter` | Insert newline |
| `Esc` `Esc` | Interrupt running prompt |
| `Ctrl+P` | Commands panel overlay |
| `Ctrl+N` | New session |
| `Ctrl+S` | Session switcher overlay |
| `↑`/`↓` / `k`/`j` | Scroll chat viewport |
| `PgUp`/`PgDn` | Page scroll chat |
| `Ctrl+C` | Quit |

Paste protection: keystrokes <20ms apart treated as paste (inserted as text, not sent).

## Repo quirks

- `agent/` uses `package agent` (NOT standalone binary; imported from `cmd/bubblecode/chat.go`).
- `opencode` CLI is NOT required at runtime — the app spawns itself (`bubblecode acp`) as the ACP agent subprocess.
- `docs/` contains authoritative design references (`opencode-ui-design.md`, `overlay.md`, `lipgloss-best-practices.md`). Check these before making UI changes.
- CI runs on `main` branch; release is triggered by `v*` tags.
