```
  ██████╗ ██╗    ██╗   ██╗ ██████╗██╗  ██╗██╗  ██╗ ██████╗ ██╗     ███████╗
  ██╔══██╗██║    ██║   ██║██╔════╝██║ ██╔╝██║  ██║██╔═══██╗██║     ██╔════╝
  ██████╔╝██║    ██║   ██║██║     █████╔╝ ███████║██║   ██║██║     █████╗
  ██╔══██╗██║    ╚██╗ ██╔╝██║     ██╔═██╗ ██╔══██║██║   ██║██║     ██╔══╝
  ██████╔╝███████╗╚████╔╝ ╚██████╗██║  ██╗██║  ██║╚██████╔╝███████╗███████╗
  ╚═════╝ ╚══════╝ ╚═══╝   ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚══════╝
```

A CLI tool that creates and manages [Docker Sandboxes](https://docs.docker.com/ai/sandboxes/) from a single YAML config file. No more hand-writing Dockerfiles and kit specs for every project.

## Features

- **Declarative config** — one YAML file defines packages, runtimes, ports, env vars, and agent settings
- **Multi-runtime support** — Node.js, pnpm, Bun, Python, Go, PHP, Rust
- **ZSH by default** — syntax highlighting, autosuggestions, colored prompt, and sensible aliases out of the box
- **Agent-ready** — first-class support for Claude Code, Codex, Copilot, Cursor, Gemini, Kiro, and more
- **Custom themes & plugins** — ship your Claude Code theme and plugin configuration into the sandbox
- **Network policies** — allowlist-only networking for sandbox isolation

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) with [Docker Sandboxes](https://docs.docker.com/ai/sandboxes/) (`sbx` CLI)
- Go 1.26+ (for building from source)

## Installation

```bash
# Clone and build
git clone https://github.com/neoighodaro/blvckhole.git
cd blvckhole
go build -o blvckhole .

# Install to ~/Developer/bin/ (default)
./blvckhole install

# Install to a custom directory
./blvckhole install --path=/usr/local/bin

# Or symlink (picks up rebuilds automatically)
./blvckhole install --symlink
```

Make sure the install directory is in your `PATH`.

## Quick Start

```bash
# Initialize a config in your project
cd my-project
blvckhole init

# Edit the generated config
$EDITOR .config/blvckhole/blvckhole.yaml

# Build and launch the sandbox
blvckhole start

# Open a shell
blvckhole ssh

# Launch the AI agent
blvckhole agent
```

## Configuration

`blvckhole init` creates `.config/blvckhole/blvckhole.yaml`. The full schema:

```yaml
name: my-project          # Sandbox name (default: directory name)
agent: claude-code         # AI agent to use

# Use a pre-built Docker image instead of generating a Dockerfile.
# When set, 'packages' and 'runtimes' are ignored.
# template: docker.io/my-org/custom-template:v1

# System packages (apt-get)
packages:
  - ripgrep
  - lazygit
  - eza

# Language runtimes
runtimes:
  node: "24"               # Major version (NodeSource) or exact e.g. "24.16.0"
  pnpm: "11.1.3"           # Requires node runtime
  # bun: "latest"
  # python: "3.12"
  # go: "1.23"
  # php: "8.4"
  # rust: "stable"

# Ports exposed from sandbox to host
ports:
  - 3000
  - "8080:80"

# Environment variables
env:
  NODE_ENV: development

# Environment files (loaded in order, later overrides earlier)
env_file:
  - .env
  - .env.sandbox

# Shell configuration (ZSH is the default shell)
shell:
  directory: /home/agent/project/src   # Working directory when opening a shell
  aliases:                              # Custom aliases alongside built-in defaults
    g: "git"
    dc: "docker compose"

# Network allowlist (deny-all by default)
network:
  - "*.npmjs.org"
  - "registry.yarnpkg.com"
  - "api.github.com"

# Claude Code agent customization
claude:
  theme: ./my-theme.json               # Custom theme (Catppuccin Mocha by default)
  plugins:
    marketplaces:
      - anthropics/claude-plugins-official
    install:
      - superpowers@claude-plugins-official
  settings:
    alwaysThinkingEnabled: true

# Instructions injected into the agent's CLAUDE.md
memory: |
  Project-specific instructions here.
```

Config is discovered at `blvckhole.yaml` (project root) or `.config/blvckhole/blvckhole.yaml`.

### Supported Agents

`claude-code` (default), `codex`, `copilot`, `cursor`, `docker-agent`, `droid`, `gemini`, `kiro`, `opencode`, `shell`

### Supported Runtimes

| Runtime | Version format | Notes |
|---------|---------------|-------|
| `node` | `"24"` or `"24.16.0"` | Major version uses NodeSource, exact version downloads from nodejs.org |
| `pnpm` | `"11.1.3"` | Installed via corepack, requires `node` |
| `bun` | `"latest"` or `"1.2.0"` | |
| `python` | `"3.12"` | |
| `go` | `"1.23"` | |
| `php` | `"8.4"` | |
| `rust` | `"stable"` or `"1.80"` | |

## CLI Commands

| Command | Description |
|---------|-------------|
| `blvckhole init` | Create a starter config in `.config/blvckhole/` |
| `blvckhole start` | Build the Docker image, generate kit, and create the sandbox |
| `blvckhole stop` | Stop the sandbox (state is retained) |
| `blvckhole restart` | Stop, remove, and recreate the sandbox |
| `blvckhole shell` | Open a ZSH shell in the sandbox |
| `blvckhole ssh` | Alias for `shell` |
| `blvckhole agent` | Launch the AI agent in the sandbox (auto-starts if needed) |
| `blvckhole agent --rebuild` | Force rebuild the image before launching the agent |
| `blvckhole status` | Show sandbox state, ports, agent, and runtimes |
| `blvckhole install` | Install the binary to `~/Developer/bin/` |
| `blvckhole install --path=<dir>` | Install to a custom directory |
| `blvckhole install --symlink` | Symlink instead of copying |

## Shell Environment

Every sandbox ships with ZSH as the default shell, configured with:

- **zsh-autosuggestions** — suggests commands from history as you type
- **zsh-syntax-highlighting** — colorizes commands in real-time
- **Colored prompt** — shows current directory and git branch
- **Built-in aliases** — `bat`/`eza` integration, `please` (sudo last command), `lazygit`, and more (when the tools are installed)

Add your own aliases per project via `shell.aliases` in the config.

## Contributing

### Getting Started

```bash
git clone https://github.com/neoighodaro/blvckhole.git
cd blvckhole
go build ./...
go test ./...
```

### Project Structure

```
cmd/                     CLI commands (cobra)
internal/
  config/                YAML config parsing and validation
  embedded/              Embedded assets (Dockerfile template, ZSH config, theme)
  envfile/               .env file parser
  kit/                   Kit generator (spec.yaml + files for sbx)
  runtime/               Runtime installers (node, pnpm, bun, python, go, php, rust)
  sandbox/               Wrapper around the sbx CLI
  template/              Dockerfile template renderer and builder
  ui/                    Terminal styling (lipgloss)
```

### Adding a New Runtime

1. Create `internal/runtime/<name>.go` implementing the `Runtime` interface:

```go
type Runtime interface {
    Name() string
    Validate(version string) error
    RootBlock(version string) string   // Dockerfile commands as root
    AgentBlock(version string) string  // Dockerfile commands as agent user
    EnvBlock(version string) string    // ENV directives
}
```

2. Register it in `internal/runtime/runtime.go`
3. Add the name to `validRuntimes` in `internal/config/config.go`
4. Add a commented example in the starter config in `cmd/init.go`

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o blvckhole .
```

### Guidelines

- Keep the CLI simple — one YAML file, no subcommands for configuration
- Follow existing patterns in the codebase
- Add tests for new functionality
- Runtime blocks should be minimal — install the runtime and clean up apt caches
- The Dockerfile template always installs ZSH; don't make this conditional
