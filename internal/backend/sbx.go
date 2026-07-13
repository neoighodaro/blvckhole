package backend

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/neoighodaro/blvckhole/internal/config"
	"github.com/neoighodaro/blvckhole/internal/kit"
	"github.com/neoighodaro/blvckhole/internal/template"
	"github.com/neoighodaro/blvckhole/internal/ui"
)

// sbxConfigHashPath is where the hash of blvckhole.yaml is stored inside the
// container, used to detect config drift.
const sbxConfigHashPath = "/home/agent/.blvckhole-config-hash"

// SbxBackend drives Docker Sandboxes via the sbx CLI.
type SbxBackend struct{}

var _ Backend = (*SbxBackend)(nil)

func (s *SbxBackend) Name() string { return "sbx" }

func (s *SbxBackend) EnsureAvailable() error {
	if _, err := exec.LookPath("sbx"); err != nil {
		return fmt.Errorf("sbx is not installed. Install it from: https://docs.docker.com/ai/sandboxes/")
	}
	return nil
}

func (s *SbxBackend) Exists(cfg *config.Config) bool {
	cmd := exec.Command("sbx", "ls", "--quiet")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) == cfg.Name {
			return true
		}
	}
	return false
}

func (s *SbxBackend) IsRunning(cfg *config.Config) bool {
	info, err := s.Status(cfg)
	if err != nil || info == nil {
		return false
	}
	return info.Status == "running"
}

func (s *SbxBackend) Status(cfg *config.Config) (*Info, error) {
	cmd := exec.Command("sbx", "ls", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("sbx ls failed: %w", err)
	}

	sandboxes, err := parseSandboxList(output)
	if err != nil {
		return nil, err
	}

	for _, sb := range sandboxes {
		if sb.Name == cfg.Name {
			return &sb, nil
		}
	}

	return nil, nil
}

// parseSandboxList decodes `sbx ls --json` output, which is either a plain
// array of sandboxes or an object wrapping one.
func parseSandboxList(output []byte) ([]Info, error) {
	var sandboxes []Info
	if err := json.Unmarshal(output, &sandboxes); err != nil {
		var wrapped map[string]json.RawMessage
		if err2 := json.Unmarshal(output, &wrapped); err2 != nil {
			return nil, fmt.Errorf("failed to parse sbx ls output: %w", err)
		}
		for _, v := range wrapped {
			if err2 := json.Unmarshal(v, &sandboxes); err2 == nil {
				break
			}
		}
	}
	return sandboxes, nil
}

// Provision builds the sandbox image (unless a custom template is set),
// generates the kit, creates the sandbox, and applies workspace link,
// network rules, published ports, and on_create scripts — the exact flow
// previously inlined in cmd/start.go.
func (s *SbxBackend) Provision(cfg *config.Config) error {
	if s.Exists(cfg) {
		fmt.Println(ui.Info.Render("Stale sandbox detected, removing..."))
		s.Remove(cfg)
	}

	kitDir := cfg.KitDir()
	if err := os.MkdirAll(kitDir, 0755); err != nil {
		return fmt.Errorf("failed to create kit directory: %w", err)
	}

	if cfg.Template == "" {
		fmt.Println(ui.Accent.Render("Building sandbox image..."))
		if err := template.Build(cfg); err != nil {
			return err
		}

		fmt.Println(ui.Accent.Render("Loading image into sandbox runtime..."))
		if err := template.LoadTemplate(cfg); err != nil {
			return err
		}
	}

	fmt.Println(ui.Accent.Render("Generating kit..."))
	if err := kit.Generate(cfg, kitDir); err != nil {
		return err
	}

	templateImage := cfg.SandboxImageName()
	if cfg.Template != "" {
		templateImage = cfg.Template
	}

	fmt.Println(ui.Accent.Render("Creating sandbox..."))
	if err := s.create(cfg.Name, templateImage, kitDir, cfg.SbxAgent(), "."); err != nil {
		return fmt.Errorf("failed to create sandbox: %w", err)
	}

	if hash := cfg.FileHash(); hash != "" {
		s.WriteConfigHash(cfg, hash)
	}

	if cfg.Workspace != "" {
		fmt.Println(ui.Accent.Render("Linking project to " + cfg.Workspace + "..."))
		if err := s.linkWorkspace(cfg.Name, cfg.ProjectDir, cfg.Workspace); err != nil {
			return fmt.Errorf("failed to link project to workspace: %w", err)
		}
	}

	if len(cfg.Network) > 0 {
		fmt.Println(ui.Accent.Render("Applying network whitelist..."))
		if err := s.AllowNetwork(cfg, cfg.Network); err != nil {
			return fmt.Errorf("failed to set network policy: %w", err)
		}
	}

	if cfg.Handoff.Enabled {
		resource := "localhost:" + cfg.HandoffPort()
		fmt.Println(ui.Accent.Render("Allowing handoff broker (" + resource + ")..."))
		if err := s.AllowNetwork(cfg, []string{resource}); err != nil {
			return fmt.Errorf("failed to allow handoff broker network: %w", err)
		}
	}

	for _, port := range cfg.Ports {
		fmt.Println(ui.Accent.Render("Publishing port " + port + "..."))
		if err := s.publishPort(cfg.Name, port); err != nil {
			return fmt.Errorf("failed to publish port %s: %w", port, err)
		}
	}

	// on_start commands run on every shell/agent session via the per-session
	// init hook (/etc/sandbox-persistent.sh, baked into the image by the
	// Dockerfile), so they are not run here — only on_create runs at creation.
	for _, command := range cfg.Scripts.OnCreate {
		fmt.Println(ui.Accent.Render("Running: " + command))
		if err := s.RunCommand(cfg, command); err != nil {
			return fmt.Errorf("on_create command failed (%s): %w", command, err)
		}
	}

	return nil
}

func (s *SbxBackend) create(name, templateImage, kitDir, agent, workDir string) error {
	args := []string{"create", "--template", templateImage, "--name", name, "--kit", kitDir, agent, workDir}
	cmd := exec.Command("sbx", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *SbxBackend) linkWorkspace(name, source, dest string) error {
	script := fmt.Sprintf("mkdir -p \"$(dirname '%s')\" && ln -sfn '%s' '%s'",
		dest, source, dest)
	cmd := exec.Command("sbx", "exec", "-u", "root", name, "--", "bash", "-c", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *SbxBackend) publishPort(name string, mapping string) error {
	cmd := exec.Command("sbx", "ports", name, "--publish", mapping)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *SbxBackend) Run(cfg *config.Config, extraArgs ...string) error {
	args := []string{"run", "--name", cfg.Name}
	if len(extraArgs) > 0 {
		args = append(args, "--")
		args = append(args, extraArgs...)
	}
	cmd := exec.Command("sbx", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *SbxBackend) RunCommand(cfg *config.Config, command string) error {
	workDir := cfg.Workspace
	if workDir == "" {
		workDir = cfg.ProjectDir
	}
	script := fmt.Sprintf("cd %s && %s", shellQuote(workDir), command)
	return s.exec(cfg.Name, false, "bash", "-c", script)
}

func (s *SbxBackend) ExecSilent(cfg *config.Config, command ...string) (string, error) {
	return s.execSilent(cfg.Name, command...)
}

func (s *SbxBackend) Shell(cfg *config.Config, dir string) error {
	shellArgs := []string{"bash"}
	if dir != "" {
		shellArgs = []string{"bash", "-c", fmt.Sprintf("cd %s && exec bash", shellQuote(dir))}
	}

	sbxPath, err := exec.LookPath("sbx")
	if err != nil {
		return fmt.Errorf("sbx not found in PATH: %w", err)
	}

	execArgs := []string{"sbx", "exec", "-it", cfg.Name, "--"}
	execArgs = append(execArgs, shellArgs...)

	return syscall.Exec(sbxPath, execArgs, os.Environ())
}

func (s *SbxBackend) Stop(cfg *config.Config) error {
	cmd := exec.Command("sbx", "stop", cfg.Name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *SbxBackend) Remove(cfg *config.Config) error {
	cmd := exec.Command("sbx", "rm", "-f", cfg.Name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *SbxBackend) AllowNetwork(cfg *config.Config, domains []string) error {
	if len(domains) == 0 {
		return nil
	}
	joined := strings.Join(domains, ",")
	cmd := exec.Command("sbx", "policy", "allow", "network", "--sandbox", cfg.Name, joined)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *SbxBackend) DenyNetwork(cfg *config.Config, domains []string) error {
	if len(domains) == 0 {
		return nil
	}
	joined := strings.Join(domains, ",")
	cmd := exec.Command("sbx", "policy", "deny", "network", "--sandbox", cfg.Name, joined)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *SbxBackend) RemoveNetwork(cfg *config.Config, domains []string) error {
	for _, domain := range domains {
		cmd := exec.Command("sbx", "policy", "rm", "network", "--sandbox", cfg.Name, "--resource", domain)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func (s *SbxBackend) ReadConfigHash(cfg *config.Config) (string, error) {
	output, err := s.execSilent(cfg.Name, "cat", sbxConfigHashPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (s *SbxBackend) WriteConfigHash(cfg *config.Config, hash string) error {
	script := fmt.Sprintf("cat > %s << 'BLVCKHOLE_EOF'\n%s\nBLVCKHOLE_EOF", sbxConfigHashPath, hash)
	_, err := s.execSilent(cfg.Name, "bash", "-c", script)
	return err
}

// exec runs a command inside the sandbox attached to the terminal.
func (s *SbxBackend) exec(name string, interactive bool, command ...string) error {
	args := []string{"exec"}
	if interactive {
		args = append(args, "-it")
	}
	args = append(args, name, "--")
	args = append(args, command...)
	cmd := exec.Command("sbx", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// execSilent runs a command inside the sandbox and returns combined output.
func (s *SbxBackend) execSilent(name string, command ...string) (string, error) {
	args := []string{"exec", name, "--"}
	args = append(args, command...)
	cmd := exec.Command("sbx", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// PrepareAgent merges blvckhole's Claude settings (statusline, plugins,
// theme) into ~/.claude/settings.json inside the sandbox — the sandbox's
// copy, never the host's.
func (s *SbxBackend) PrepareAgent(cfg *config.Config) error {
	script := fmt.Sprintf(`
set -e
SETTINGS="$HOME/.claude/settings.json"
[ -f "$SETTINGS" ] || exit 0

export SLPATH="$(ls -d ~/.claude/plugins/cache/claude-dashboard/claude-dashboard/*/dist/index.js 2>/dev/null | sort -V | tail -1)"

SANDBOX_SETTINGS="$HOME/.claude/settings.sandbox.json"
[ -f "$HOME/.claude/themes/sandbox.json" ] && export HAS_THEME=1 || export HAS_THEME=0

if [ -f "$SANDBOX_SETTINGS" ]; then
  jq -s '%s' "$SETTINGS" "$SANDBOX_SETTINGS" > "$SETTINGS.tmp"
else
  jq '%s' "$SETTINGS" > "$SETTINGS.tmp"
fi
mv "$SETTINGS.tmp" "$SETTINGS"
`, jqMergeFilter(cfg), jqNoMergeFilter(cfg))

	_, err := s.execSilent(cfg.Name, "bash", "-c", script)
	return err
}

func jqSettingsFilter(cfg *config.Config) string {
	f := ""
	f += `if env.SLPATH != "" then .statusLine = {type: "command", command: ("node " + env.SLPATH)} else . end`

	if len(cfg.Claude.Plugins.Install) > 0 {
		f += ` | .enabledPlugins = (.enabledPlugins // {})`
		for _, plugin := range cfg.Claude.Plugins.Install {
			f += fmt.Sprintf(` | .enabledPlugins["%s"] = true`, plugin)
		}
	}

	f += ` | if env.HAS_THEME == "1" then .theme = "custom:sandbox" | .themeId = "custom:sandbox" else . end`
	return f
}

func jqMergeFilter(cfg *config.Config) string {
	return ".[0] * .[1] | " + jqSettingsFilter(cfg)
}

func jqNoMergeFilter(cfg *config.Config) string {
	return jqSettingsFilter(cfg)
}
