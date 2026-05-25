```
  ██████╗ ██╗    ██╗   ██╗ ██████╗██╗  ██╗██╗  ██╗ ██████╗ ██╗     ███████╗
  ██╔══██╗██║    ██║   ██║██╔════╝██║ ██╔╝██║  ██║██╔═══██╗██║     ██╔════╝
  ██████╔╝██║    ██║   ██║██║     █████╔╝ ███████║██║   ██║██║     █████╗
  ██╔══██╗██║    ╚██╗ ██╔╝██║     ██╔═██╗ ██╔══██║██║   ██║██║     ██╔══╝
  ██████╔╝███████╗╚████╔╝ ╚██████╗██║  ██╗██║  ██║╚██████╔╝███████╗███████╗
  ╚═════╝ ╚══════╝ ╚═══╝   ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚══════╝
```

One YAML file. One command. A fully configured [Docker Sandbox](https://docs.docker.com/ai/sandboxes/) with your runtimes, packages, and AI agent ready to go.

```bash
blvckhole init       # drop a config in your project
blvckhole start      # build image, create sandbox
blvckhole agent      # launch your AI agent inside it
```

You stop hand-writing Dockerfiles and kit specs. blvckhole reads a YAML config, generates a Dockerfile, builds it, wires up the sandbox, and drops you (or your agent) into a ZSH shell with syntax highlighting, autosuggestions, and aliases already configured.

Supports Claude Code, Codex, Copilot, Cursor, Gemini, Kiro, and others. Supports Node.js, pnpm, Bun, Python, Go, PHP, and Rust. Network is deny-all by default — you allowlist what you need.

## Install

Requires [Docker Desktop](https://www.docker.com/products/docker-desktop/) with [Docker Sandboxes](https://docs.docker.com/ai/sandboxes/) (`sbx` CLI). Building from source needs Go 1.26+.

```bash
git clone https://github.com/neoighodaro/blvckhole.git
cd blvckhole
go build -o blvckhole .

./blvckhole install                     # copies to ~/Developer/bin/
./blvckhole install --path=/usr/local/bin
./blvckhole install --symlink           # picks up rebuilds automatically
```

Make sure the install directory is in your `PATH`.

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

`blvckhole init` creates `.config/blvckhole/blvckhole.yaml`. Also found at `blvckhole.yaml` in the project root.

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

network:                   # deny-all unless listed here
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

## ZSH

Every sandbox gets ZSH with zsh-autosuggestions, zsh-syntax-highlighting, a colored prompt with git branch, and aliases for `bat`, `eza`, `lazygit`, and `please` (sudo last command) when those tools are installed. Add your own via `shell.aliases`.

