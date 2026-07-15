package template

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/neoighodaro/blvckhole/internal/config"
	"github.com/neoighodaro/blvckhole/internal/embedded"
	"github.com/neoighodaro/blvckhole/internal/runtime"
)

type templateData struct {
	Packages    []string
	RootBlocks  []string
	AgentBlocks []string
	EnvBlocks   []string
	Plugins     config.ClaudePlugins
}

// bridgeScriptPath is where the embedded bridge.sh lands inside the image.
const bridgeScriptPath = "/usr/local/lib/blvckhole/bridge.sh"

func Render(cfg *config.Config) (string, error) {
	data := templateData{
		Packages: effectivePackages(cfg),
		Plugins:  cfg.Claude.Plugins,
	}

	names := make([]string, 0, len(cfg.Runtimes))
	for name := range cfg.Runtimes {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		version := cfg.Runtimes[name]
		r := runtime.Get(name)
		if r == nil {
			return "", fmt.Errorf("unknown runtime: %s", name)
		}
		if name == "php" {
			if php, ok := r.(*runtime.PhpRuntime); ok {
				php.Extensions = cfg.Php.Extensions
			}
		}
		if block := r.RootBlock(version); block != "" {
			data.RootBlocks = append(data.RootBlocks, block)
		}
		if block := r.AgentBlock(version); block != "" {
			data.AgentBlocks = append(data.AgentBlocks, block)
		}
		if block := r.EnvBlock(version); block != "" {
			data.EnvBlocks = append(data.EnvBlocks, block)
		}
	}

	if len(cfg.Bridges) > 0 {
		data.RootBlocks = append(data.RootBlocks, bridgeInstallBlock())
	}

	if block := sessionHookBlock(cfg); block != "" {
		data.RootBlocks = append(data.RootBlocks, block)
	}

	funcMap := template.FuncMap{
		"join": strings.Join,
	}

	tmpl, err := template.New("Dockerfile").Funcs(funcMap).Parse(embedded.DockerfileTmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse Dockerfile template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render Dockerfile: %w", err)
	}

	return buf.String(), nil
}

// effectivePackages returns the apt package list to install, adding socat when
// bridges are configured (they need it) unless the user already listed it. The
// user's configured Packages slice is left untouched.
func effectivePackages(cfg *config.Config) []string {
	if len(cfg.Bridges) == 0 {
		return cfg.Packages
	}
	for _, p := range cfg.Packages {
		if p == "socat" {
			return cfg.Packages
		}
	}
	return append(append([]string{}, cfg.Packages...), "socat")
}

func bridgeInstallBlock() string {
	return "USER root\n" +
		"COPY bridge.sh " + bridgeScriptPath + "\n" +
		"RUN chmod +x " + bridgeScriptPath + "\n" +
		"USER agent"
}

// sessionHookBlock returns a Dockerfile RUN that appends bridge bring-up and
// on_start commands to /etc/sandbox-persistent.sh — the base image's
// per-session init hook (wired as BASH_ENV and CLAUDE_ENV_FILE), so they run on
// every shell and agent session, including after a stop/resume. Bridges run
// first so host services are reachable before user on_start commands. Each line
// is run with output suppressed and `|| true` so it can't disrupt or spam the
// host shell; commands must therefore be idempotent. Returns "" when there is
// nothing to run.
//
// Because the hook is sourced by EVERY non-interactive bash (via BASH_ENV), a
// command that itself invokes bash would re-source the hook in that child
// shell, re-run the commands, and recurse forever (fork bomb). The whole block
// is therefore wrapped in an *exported* sentinel guard: child shells inherit
// BLVCKHOLE_ON_START and skip the block, so it runs once per process tree.
func sessionHookBlock(cfg *config.Config) string {
	if len(cfg.Bridges) == 0 && len(cfg.Scripts.OnStart) == 0 {
		return ""
	}

	workDir := cfg.ProjectDir
	if cfg.Workspace != "" {
		workDir = cfg.Workspace
	}

	lines := []string{`if [ -z "${BLVCKHOLE_ON_START:-}" ]; then export BLVCKHOLE_ON_START=1`}
	for _, b := range cfg.Bridges {
		args := fmt.Sprintf("%s %d %d", b.Name, b.Port, b.HostPort)
		if b.Env != "" {
			args += " " + b.Env
		}
		lines = append(lines, fmt.Sprintf("%s %s >/dev/null 2>&1 || true", bridgeScriptPath, args))
	}
	for _, cmd := range cfg.Scripts.OnStart {
		lines = append(lines, fmt.Sprintf("( cd %s && %s ) >/dev/null 2>&1 || true", workDir, cmd))
	}
	lines = append(lines, "fi")

	var b strings.Builder
	b.WriteString("USER root\n")
	b.WriteString("RUN printf '%s\\n' \\\n")
	for _, line := range lines {
		b.WriteString("      " + shellSingleQuote(line) + " \\\n")
	}
	b.WriteString("    >> /etc/sandbox-persistent.sh\n")
	b.WriteString("USER agent")
	return b.String()
}

// shellSingleQuote wraps s in single quotes, escaping any embedded single
// quotes, so it survives as one literal argument to printf in a RUN.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func Build(cfg *config.Config) error {
	dockerfile, err := Render(cfg)
	if err != nil {
		return err
	}

	kitDir := cfg.KitDir()

	if err := os.WriteFile(filepath.Join(kitDir, "Dockerfile"), []byte(dockerfile), 0644); err != nil {
		return err
	}

	if len(cfg.Bridges) > 0 {
		if err := os.WriteFile(filepath.Join(kitDir, "bridge.sh"), embedded.BridgeSh, 0755); err != nil {
			return err
		}
	}

	cmd := exec.Command("docker", "build", "-t", cfg.SandboxImageName(), kitDir)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker build failed:\n%s", string(output))
	}

	return nil
}

func LoadTemplate(cfg *config.Config) error {
	tarPath := fmt.Sprintf("/tmp/%s.tar", cfg.SandboxImageName())

	save := exec.Command("docker", "image", "save", cfg.SandboxImageName(), "-o", tarPath)
	if output, err := save.CombinedOutput(); err != nil {
		return fmt.Errorf("docker image save failed:\n%s", string(output))
	}

	load := exec.Command("sbx", "template", "load", tarPath)
	if output, err := load.CombinedOutput(); err != nil {
		exec.Command("rm", "-f", tarPath).Run()
		return fmt.Errorf("sbx template load failed:\n%s", string(output))
	}

	exec.Command("rm", "-f", tarPath).Run()
	return nil
}
