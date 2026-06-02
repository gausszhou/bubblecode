# bubblecode

TUI for AI agents via the [ACP protocol](https://github.com/coder/acp-go-sdk) (`opencode acp`).

## Build

```bash
make build              # build client and agent
make build-client       # build TUI app only
make build-agent        # build standalone example agent only
```

Or manually:

```bash
go build -o bin/client.exe .        # TUI app
go build -o bin/agent.exe ./agent   # standalone example agent
```

## Run

```bash
go run .                              # requires `opencode` in PATH
go run . --debug                      # log to ./bubblecode.log in current directory
go run ./agent                        # example agent, connects via stdio
```

## Development

```bash
make fmt          # format code (go fmt ./...)
make vet          # run go vet
make lint         # run golangci-lint
make check        # run fmt + vet + lint
make clean        # remove bin/ directory
```

## Architecture

```
main.go → spawns agent.exe subprocess
          ↓ stdin/stdout pipes
    client/ → ACP client (sends/receives commands)
          ↓ channels
       tui/ → Bubble Tea TUI (chat + sidebar)
```

- `main.go` — entrypoint. Spawns `agent.exe` as subprocess, connects via stdin/stdout pipes using `acp-go-sdk`.
- `client/client.go` — implements `acp.Client` interface. Sends/receives commands over channels. Also handles fs ops (ReadTextFile, WriteTextFile) and terminal stubs.
- `tui/` — Bubble Tea v2 app. Left-right layout: chat+input+status (68%) | todo+session sidebar.
- `tui/theme/theme.go` — centralized color and style definitions.
- `tui/component/` — reusable UI components (TodoList, StatusBar, SessionList, etc.).
- `agent/agent.go` — separate `package main` binary (NOT importable). Demo agent with simulated streaming, tool calls, and permission requests.
- `docs/` — design references and best practices guides.

## Key conventions

- **Enter** sends message, **Shift+Enter** inserts newline.
- **Double-Esc** interrupts running prompt.
- **Ctrl+P** opens command overlay, **Ctrl+N** new session, **Ctrl+S** switch session, **Ctrl+C** quit.
- Paste protection: keystrokes <20ms apart are treated as paste and inserted as text, not sent.
- Design: warm dark (`#201d1d`) bg, Berkeley Mono aesthetic, flat surfaces (no shadows), 4px border radius by default (6px for inputs). See `tui/theme/theme.go`.

## Go Best Practices

### Code Organization

- One package per directory, package name matches directory name
- `agent/` has its own `package main` — do not import from main app
- Exported identifiers use PascalCase, unexported use camelCase
- Interface names end with `-er` when applicable (e.g., `Reader`, `Writer`)

### Error Handling

- Errors are values, wrap with context using `fmt.Errorf("context: %w", err)`
- Check errors immediately, don't defer error handling
- Use custom error types for domain-specific errors

### Testing

- Test files named `*_test.go` in same package
- Table-driven tests for multiple cases
- Run `go test ./...` to execute all tests

### Dependencies

- Use `go mod tidy` after adding/removing dependencies
- Pin versions in `go.mod`, don't use `latest`
- Vendor dependencies only if required for offline builds

## Workflow

- All development on `develop` branch. `master` is protected — do NOT push directly.
- Merge into `master` via PR only.

## Repo quirks

- `agent/` has its own `package main` — do not import from main app.
- `opencode acp` must be discoverable in PATH at runtime.
- Windows: binaries use `.exe` extension, Makefile uses `rm -rf` (requires Git Bash or WSL).
