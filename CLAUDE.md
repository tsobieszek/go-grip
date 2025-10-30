# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

go-grip is a Go-based tool that renders Markdown files locally with GitHub-style presentation. It's a reimplementation of the Python tool `grip`, but without external API dependencies. The application serves rendered Markdown via a local HTTP server with hot-reload capabilities.

Key characteristics:
- Single binary with embedded assets (CSS, JS, templates, emojis)
- No external API calls (fully offline)
- Cross-platform support (Linux, macOS, Windows, BSD)
- GitHub-flavored markdown with custom extensions

## Development Commands

### Building
```bash
make build              # Build binary to bin/go-grip
make compile            # Cross-compile for all platforms (darwin, linux, windows)
make install            # Install to /usr/local/bin (requires sudo)
```

### Running
```bash
make run README.md                          # Run with file
make run -- README.md --theme dark --port 8080  # Run with custom flags
go run -tags debug main.go <file>           # Direct run (development)
```

Available flags:
- `--theme` (light/dark/auto): CSS theme selection, default: auto
- `--browser/-b` (bool): Auto-open browser, default: true
- `--host/-H` (string): Server host, default: localhost
- `--port/-p` (int): Server port, default: 6419
- `--bounding-box` (bool): Add HTML bounding box, default: true

### Testing & Quality
```bash
make test               # Run tests (currently no test files exist)
make format             # Run gofmt
make lint               # Run golangci-lint
```

Note: Tests are currently missing (acknowledged TODO in README). When adding tests, follow table-driven test patterns.

### Maintenance
```bash
make vendor             # Vendor dependencies
make clean              # Remove bin/ directory
make emojiscraper       # Update emoji data (debug build only)
```

### Emoji Scraper (Debug Build Only)
The emoji scraper updates emoji mappings from GitHub:
```bash
make emojiscraper
# Equivalent to: go run -tags debug main.go emojiscraper defaults/static/emojis pkg/emoji_map.go
```

This scrapes https://gist.github.com/rxaviers/7360908, downloads emoji images, and regenerates `pkg/emoji_map.go`.

## Architecture

### Directory Structure

```
cmd/            - CLI commands (Cobra framework)
  root.go       - Main command with flag definitions
  emojiscraper.go - Debug-only emoji scraper command (build tag: debug)

pkg/            - Core business logic
  server.go     - HTTP server with hot-reload (aarol/reload)
  parser.go     - Markdown→HTML with custom render hooks
  emoji_map.go  - Auto-generated emoji mapping (DO NOT EDIT MANUALLY)
  open.go       - Cross-platform browser launcher

internal/       - Internal utilities (not importable externally)
  emoji.go      - Web scraper using colly for emoji data

defaults/       - Embedded assets (compiled into binary)
  embed.go      - Go embed directives
  static/       - CSS (42KB), JS (mermaid 2.5MB), emojis, favicon
  templates/    - HTML templates (layout, alerts, mermaid)
```

### Data Flow

1. **CLI Layer (cmd/)**: Parses flags via Cobra, initializes Parser and Server
2. **Parser (pkg/parser.go)**: Converts Markdown to HTML using gomarkdown with custom render hooks
3. **Server (pkg/server.go)**: HTTP server that detects `.md` files and renders them, serves static assets, manages hot-reload
4. **Embedded Assets (defaults/)**: Templates and static files compiled into binary via `//go:embed`

### Markdown Preprocessing

Before parsing, markdown is preprocessed to add blank lines before lists that don't have them. This implements GitHub Flavored Markdown behavior where lists can interrupt paragraphs without requiring a blank line.

The preprocessor:
- Adds blank lines before lists that lack them (GFM-style)
- Preserves code blocks (doesn't modify content inside ``` blocks)
- Handles blockquotes with lists
- Doesn't add extra lines between consecutive list items

### Custom Render Hooks

The parser uses custom AST hooks to extend markdown rendering:

- **Code blocks**: Syntax highlighting via Chroma, special handling for Mermaid diagrams
- **Blockquotes**: Detects `[!TYPE]` syntax to render GitHub-style alerts (Note, Tip, Important, Warning, Caution)
- **Text nodes**: Replaces `:emoji:` shortcodes using EmojiMap
- **List items**: Detects `[ ]` and `[x]` to render task list checkboxes

Hook implementation pattern in `parser.go`:
```go
func (m Parser) renderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
    switch node.(type) {
    case *ast.CodeBlock:
        return renderHookCodeBlock(w, node, m.theme)
    // ... other hooks
    }
    return ast.GoToNext, false
}
```

### Server Routing

- `GET /` - Markdown renderer (if `.md` extension) or file server (otherwise)
- `GET /static/*` - Embedded static assets from `defaults.StaticFiles`
- WebSocket - Hot-reload via aarol/reload library

The server uses regex `(?i)\.md$` to detect markdown files (case-insensitive).

## Key Implementation Details

### Embedded Assets Strategy
All static assets are embedded at compile-time using Go's `//go:embed` directive. This creates a single standalone binary with no runtime file dependencies. Assets are accessed via `embed.FS` types:
- `defaults.Templates` - HTML templates
- `defaults.StaticFiles` - CSS, JS, images, emojis

### Theme System
Supports three modes (light/dark/auto). The `auto` mode uses CSS media queries `(prefers-color-scheme: light/dark)` to respect system preferences. Syntax highlighting CSS is dynamically generated via Chroma's `WriteCSS()` method.

### Hot-Reload Mechanism
The `aarol/reload` library watches the specified directory for file changes and injects WebSocket code into served HTML pages. On file change, it sends a reload message to connected browsers.

### Build Tags
The emoji scraper uses `//go:build debug` to exclude it from production builds. Include debug features with:
```bash
go run -tags debug main.go emojiscraper ...
```

## Dependencies

**Core dependencies** (do not remove without replacement):
- `github.com/spf13/cobra` - CLI framework
- `github.com/gomarkdown/markdown` - Markdown parsing with AST
- `github.com/alecthomas/chroma/v2` - Syntax highlighting
- `github.com/aarol/reload` - File watching and hot-reload
- `github.com/gocolly/colly/v2` - Web scraping (emoji scraper only)

All dependencies are managed via Go modules. Vendoring is optional (`make vendor`).

## Platform-Specific Code

### Browser Launching (`pkg/open.go`)
Platform detection via `runtime.GOOS`:
- Windows: `cmd /c start`
- macOS: `open`
- Linux/BSD: `xdg-open`

### systemd Service
Located in `systemd/markdown.service` for running as a user daemon:
```bash
systemctl --user enable markdown.service
systemctl --user start markdown.service
```

### Plan9 Integration
Plumbing rules in `plan9/plumbing` allow opening `.md` files via Plan9 port's plumber.

## Code Generation

**`pkg/emoji_map.go` is auto-generated** by the emoji scraper. Do not edit manually. Regenerate with:
```bash
make emojiscraper
```

The generated file contains a map of emoji shortcodes to either:
- Unicode characters (e.g., `:+1:` → "👍")
- Image paths (e.g., `:bowtie:` → "/static/emojis/bowtie.png")

## Known Limitations

1. **No tests**: Acknowledged in README TODO section
2. **No HTML export**: Feature mentioned in README TODO
3. **No configuration file**: All settings via CLI flags
4. **Fixed port on conflict**: No automatic port selection if 6419 is busy

## CI/CD

GitHub Actions workflows:
- `build.yml` - Runs on push/PR: build, test, format check, golangci-lint
- `release.yml` - Triggered on release: cross-compiles for 9 platform/arch combinations
- `emojiscraper.yml` - Automates emoji updates (assumed)

Cross-compilation targets:
- Linux: 386, amd64, arm64
- Windows: 386, amd64, arm64
- macOS: amd64, arm64

## Nix Support

The project includes `flake.nix` for reproducible builds:
```bash
nix build github:chrishrb/go-grip
nix run github:chrishrb/go-grip
```

Vendor hash is pinned in `flake.nix` and must be updated when dependencies change.
