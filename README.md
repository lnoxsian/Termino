# Termino (TermWeb)

A high-performance, low-overhead **single-binary Go web application** providing a minimal, flat workstation desktop environment with draggable, resizable terminal windows running directly in modern web browsers. Powered by **xterm.js + WebSocket + htmx + Go PTY**.

```text
┌──────────────────────────────────────────────────────────────┐
│  ┌───────────────────────── bash ─────────────────────[□][X]┐│
│  │ user@host:~$ ls                                          ││
│  │ bin  cmd  config.toml  internal  Makefile  web           ││
│  │ user@host:~$ █                                           ││
│  └──────────────────────────────────────────────────────────┘│
│                                                              │
│                   ┌──────────────── bash ─────────────[□][X]┐│
│                   │ $ top                                   ││
│                   │ PID USER   RES  %CPU COMMAND            ││
│                   │   1 root   10M   0.0 termino            ││
│                   └─────────────────────────────────────────┘│
│                                                              │
│       ┌─ [Right Click] ──────┐                               │
│       │ New Terminal         │                               │
│       │ ───────────────────  │                               │
│       │ Cascade Windows      │                               │
│       │ Tile Windows         │                               │
│       │ ───────────────────  │                               │
│       │ About                │                               │
│       └──────────────────────┘                               │
└──────────────────────────────────────────────────────────────┘
```

## Features

- **Single Self-Contained Binary**: All static templates, CSS, JS, and vendor bundles (`xterm.js`, `htmx`) are embedded into the Go executable via `//go:embed`. No external runtime dependencies, node packages, or CDNs required.
- **PTY Native Architecture**: The browser executes zero shell logic. Go spawns native Linux pseudo-terminals (`/bin/bash`, `$SHELL`) and bridges binary I/O to `xterm.js` over high-efficiency WebSockets.
- **Minimal Flat Workstation UI**:
  - Full-screen desktop without taskbars or icon clutter.
  - Quick **Right-Click Context Menu**:
    - **New Terminal**: Opens a new terminal window.
    - **Cascade Windows**: Neatly staggers open windows diagonally.
    - **Tile Windows**: Grid-tiles all open windows across the viewport.
    - **About**: Workstation specs, architecture info, and live theme switcher dropdown.
  - Clean, centered process title (e.g. `bash`) without PIDs or instance counters.
  - Flat 1px single-color borders and zero shadows/animations for minimal latency.
- **High-Performance & Low-Resource Footprint**:
  - Buffer pooling via `sync.Pool` on WebSocket reading and writing.
  - Direct binary packet routing bypassing JSON parsing on keyboard streaming.
  - Process group teardown (`syscall.Kill(-pgid, SIGKILL)`) preventing orphan processes.
  - Throttled window repositioning via `requestAnimationFrame`.
  - Stripped symbols (`-ldflags="-s -w" -trimpath`) resulting in ~9MB binaries.
  - Idle memory: **~10 MB RSS**, CPU: **0.0%**.
- **Themes**:
  - `Classic Teal`
  - `Green Phosphor` (VT100 CRT)
  - `Amber CRT`
  - `Cobalt Blue`
  - `Monochrome`

---

## Precompiled Architecture Binaries

Termino can be cross-compiled cleanly with pure Go (`CGO_ENABLED=0`) across all major Linux architectures:

| Architecture | Target Binary | OS / Arch | Recommended Platforms |
| :--- | :--- | :--- | :--- |
| **AMD64** | `bin/termino-linux-amd64` | `linux/amd64` | Intel & AMD 64-bit servers, PCs, VPS |
| **x86** | `bin/termino-linux-x86` | `linux/386` | Legacy 32-bit PCs, x86 embedded SBCs |
| **ARM64** | `bin/termino-linux-arm64` | `linux/arm64` | Raspberry Pi 3/4/5 (64-bit), AWS Graviton, Apple Silicon |
| **ARM32** | `bin/termino-linux-arm32` | `linux/arm (v7)` | Raspberry Pi 2/3/4 (32-bit), BeagleBone, Orange Pi |

---

## Makefile Usage

The provided `Makefile` streamlines building, cross-compilation, testing, and distribution.

```bash
# Build binary for your current host architecture into bin/termino
make build

# Build all 4 architecture binaries + compute SHA256 checksums
make build-all

# Build specific architectures:
make build-amd64    # Linux 64-bit x86
make build-x86      # Linux 32-bit x86
make build-arm64    # Linux 64-bit ARM
make build-arm32    # Linux 32-bit ARM (ARMv7)

# Verify checksums of built binaries
make checksums

# Pull latest vendor assets (htmx.min.js, xterm.js, etc.)
make vendor

# Run test suite
make test

# Run directly from source
make run

# Clean build artifacts
make clean

# View all targets
make help
```

---

## Quick Start

### 1. Build and Run

```bash
# Build host binary
make build

# Launch Termino
./bin/termino
```

Open your browser at `http://localhost:7681`.

### 2. Command-Line Options

```bash
# Print version and build info
./bin/termino -version

# Bind to custom port and shell
./bin/termino -host 127.0.0.1 -port 8080 -shell /bin/zsh

# Choose initial theme
./bin/termino -theme green

# Custom config file
./bin/termino -config my-config.toml
```

### 3. Configuration File (`config.toml`)

```toml
[server]
host = "0.0.0.0"
port = 7681

[terminal]
shell = "/bin/bash"
scrollback = 5000
font_size = 14

[desktop]
theme = "classic" # "classic", "green", "amber", "blue", "monochrome"
```

---

## Project Structure

```text
termweb/
├── cmd/
│   └── termino/
│       └── main.go           # CLI entrypoint, flag parsing, graceful shutdown
│
├── internal/
│   ├── config/
│   │   ├── config.go         # TOML and default configuration
│   │   └── config_test.go    # Configuration tests
│   ├── server/
│   │   ├── server.go         # HTTP server lifecycle
│   │   ├── routes.go         # HTMX routes and static handlers
│   │   ├── websocket.go      # Low-overhead WebSocket-to-PTY bridge
│   │   └── server_test.go    # Route & WebSocket tests
│   └── terminal/
│       ├── pty.go            # Native Linux PTY creation and window resize
│       ├── session.go        # Session wrapper & process group lifecycle
│       ├── manager.go        # Concurrent session manager
│       └── terminal_test.go  # Session manager unit tests
│
├── web/
│   ├── templates/
│   │   ├── index.html        # Main workstation shell
│   │   ├── desktop.html      # Desktop surface and right-click context menu
│   │   └── terminal.html     # Minimal terminal window component
│   ├── static/
│   │   ├── css/
│   │   │   ├── desktop.css   # Flat desktop theme & context menu styling
│   │   │   └── terminal.css  # Minimal single-border terminal styling
│   │   └── js/
│   │       ├── desktop.js    # Window manager (drag, resize, tile, cascade)
│   │       └── terminal.js   # xterm.js setup, FitAddon, binary WebSocket
│   └── vendor/
│       ├── htmx.min.js       # HTMX library
│       ├── xterm.js          # xterm.js core
│       ├── xterm.css         # xterm.js base CSS
│       └── xterm-addon-fit.js# xterm FitAddon
│
├── scripts/
│   └── pull_vendor.sh        # Pull latest vendor assets (htmx, xterm.js)
│
├── Makefile                  # Cross-platform build automation
├── config.toml               # Server and terminal configuration
├── embed.go                  # Embedded file system definition
├── go.mod                    # Go module
└── README.md
```

---

## License

MIT
