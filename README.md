```
  ██████╗ ██╗    ██╗   ██╗ ██████╗██╗  ██╗██╗  ██╗ ██████╗ ██╗     ███████╗
  ██╔══██╗██║    ██║   ██║██╔════╝██║ ██╔╝██║  ██║██╔═══██╗██║     ██╔════╝
  ██████╔╝██║    ██║   ██║██║     █████╔╝ ███████║██║   ██║██║     █████╗
  ██╔══██╗██║    ╚██╗ ██╔╝██║     ██╔═██╗ ██╔══██║██║   ██║██║     ██╔══╝
  ██████╔╝███████╗╚████╔╝ ╚██████╗██║  ██╗██║  ██║╚██████╔╝███████╗███████╗
  ╚═════╝ ╚══════╝ ╚═══╝   ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚══════╝
```

Declarative [Docker Sandbox](https://docs.docker.com/ai/sandboxes/) manager. One YAML config, one command — sandbox is up with your runtimes, packages, and AI agent.

Sandboxes isolate your development environment from your host. Compromised dependencies can't touch your filesystem or phone home unless you explicitly allowlist the domain. Good default against supply chain attacks.

```bash
blvckhole init       # scaffold config
blvckhole start      # build image, create sandbox
blvckhole agent      # launch AI agent inside
```

## Install

Requires [Docker Desktop](https://www.docker.com/products/docker-desktop/) with [Docker Sandboxes](https://docs.docker.com/ai/sandboxes/) (`sbx` CLI).

Download a pre-built binary from [GitHub Releases](https://github.com/neoighodaro/blvckhole/releases), extract it, and put it somewhere in your `PATH`.

Or build from source (needs Go 1.26+):

```bash
git clone https://github.com/neoighodaro/blvckhole.git
cd blvckhole
go build -o blvckhole .

./blvckhole install                     # copies to ~/Developer/bin/
./blvckhole install --path=/usr/local/bin
./blvckhole install --symlink           # picks up rebuilds automatically
```

## Usage

```bash
blvckhole init                # scaffold config
blvckhole start               # build + launch
blvckhole ssh                 # open a shell
blvckhole agent               # launch the agent (starts sandbox if needed)
blvckhole agent --rebuild     # force image rebuild first
blvckhole stop                # stop (state is kept)
blvckhole restart             # tear down + recreate
blvckhole status              # show sandbox state, ports, runtimes
```

## Config

`blvckhole init` creates `.config/blvckhole/blvckhole.yaml`. Also discovered at `blvckhole.yaml` in the project root.

```yaml
name: my-project          # default: directory name
agent: claude-code         # or codex, copilot, cursor, docker-agent, droid, gemini, kiro, opencode, shell

# Skip Dockerfile generation — use your own image instead
# template: docker.io/my-org/custom-template:v1

packages:                  # apt-get
  - ripgrep
  - lazygit
  - eza

runtimes:
  node: "24"               # major version (NodeSource) or exact: "24.16.0"
  pnpm: "11.1.3"           # needs node
  # bun: "latest"
  # python: "3.12"
  # go: "1.23"
  # php: "8.4"
  # rust: "stable"

ports:
  - 3000
  - "8080:80"

env:
  NODE_ENV: development

env_file:                  # loaded in order, later wins
  - .env
  - .env.sandbox

shell:
  directory: /home/agent/project/src
  aliases:
    g: "git"
    dc: "docker compose"

network:                   # allowlist — when set, only these domains are reachable
  - "*.npmjs.org"
  - "registry.yarnpkg.com"
  - "api.github.com"

claude:                    # Claude Code-specific
  theme: ./my-theme.json   # Catppuccin Mocha by default
  plugins:
    marketplaces:
      - anthropics/claude-plugins-official
    install:
      - superpowers@claude-plugins-official
  settings:
    alwaysThinkingEnabled: true

memory: |                  # injected into the agent's CLAUDE.md
  Project-specific instructions here.
```

## Shell

Sandboxes use bash with a colored prompt, git branch display, and persistent history. Aliases for `bat`, `eza`, `lazygit`, and `please` (sudo last command) are available when those tools are installed. Add your own via `shell.aliases`.

## Agents

Supported: `claude-code`, `codex`, `copilot`, `cursor`, `docker-agent`, `droid`, `gemini`, `kiro`, `opencode`, `shell`

## Runtimes

| Runtime | Version format | Notes |
|---------|---------------|-------|
| `node` | `"24"` or `"24.16.0"` | Major version uses NodeSource, exact downloads from nodejs.org |
| `pnpm` | `"11.1.3"` | Requires `node` |
| `bun` | `"latest"` or `"1.2.0"` | |
| `python` | `"3.12"` | |
| `go` | `"1.23"` | |
| `php` | `"8.4"` | |
| `rust` | `"stable"` or `"1.80"` | |

## Contributing

```bash
git clone https://github.com/neoighodaro/blvckhole.git
cd blvckhole
go build ./...
go test ./...
```

### Structure

```
cmd/                     CLI commands (cobra)
internal/
  config/                YAML config parsing and validation
  embedded/              Embedded assets (Dockerfile template, bashrc, theme)
  envfile/               .env file parser
  kit/                   Kit generator (spec.yaml + files for sbx)
  runtime/               Runtime installers (node, pnpm, bun, python, go, php, rust)
  sandbox/               Wrapper around the sbx CLI
  template/              Dockerfile template renderer and builder
  ui/                    Terminal styling (lipgloss)
```

### Adding a Runtime

1. Create `internal/runtime/<name>.go` implementing:

```go
type Runtime interface {
    Name() string
    Validate(version string) error
    RootBlock(version string) string   // Dockerfile commands as root
    AgentBlock(version string) string  // Dockerfile commands as agent user
    EnvBlock(version string) string    // ENV directives
}
```

2. Register in `internal/runtime/runtime.go`
3. Add to `validRuntimes` in `internal/config/config.go`
4. Add a commented example in the starter config in `cmd/init.go`
