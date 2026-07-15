package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/neoighodaro/blvckhole/internal/config"
	"github.com/neoighodaro/blvckhole/internal/ui"
)

// nonoAgents maps blvckhole agent names to the nono profile preset to
// extend and the host binary to launch. Agents absent here are rejected by
// validateForNono.
var nonoAgents = map[string]struct {
	preset string
	binary string
}{
	"claude-code": {preset: "claude-code", binary: "claude"},
	"codex":       {preset: "codex", binary: "codex"},
	"opencode":    {preset: "opencode", binary: "opencode"},
	"shell":       {preset: "", binary: "bash"},
}

// nonoRuntimes maps blvckhole runtime names to the host binary to verify,
// the argument that prints its version, and the nono profile group exposing
// its toolchain paths. An empty group means the runtime is granted via
// filesystem access instead (bun). checkVersion=false skips pin checking
// (rust versions are rustup-managed, not meaningful to pin against cargo).
var nonoRuntimes = map[string]struct {
	binary       string
	versionArg   string
	group        string
	checkVersion bool
}{
	"node":   {binary: "node", versionArg: "--version", group: "node_runtime", checkVersion: true},
	"pnpm":   {binary: "pnpm", versionArg: "--version", group: "node_runtime", checkVersion: true},
	"bun":    {binary: "bun", versionArg: "--version", group: "", checkVersion: true},
	"python": {binary: "python3", versionArg: "--version", group: "python_runtime", checkVersion: true},
	"go":     {binary: "go", versionArg: "version", group: "go_runtime", checkVersion: true},
	"rust":   {binary: "cargo", versionArg: "", group: "rust_runtime", checkVersion: false},
	"php":    {binary: "php", versionArg: "--version", group: "homebrew", checkVersion: true},
}

// Host probes, replaceable in tests.
var (
	nonoLookPath   = exec.LookPath
	nonoRunVersion = func(binary, versionArg string) (string, error) {
		out, err := exec.Command(binary, versionArg).CombinedOutput()
		return string(out), err
	}
)

// validateForNono strictly checks a config against what nono can honor.
// Errors abort provisioning before anything is generated; warnings name
// keys that will be skipped. Warnings are nil whenever err is non-nil.
func validateForNono(cfg *config.Config) (warnings []string, err error) {
	if len(cfg.Packages) > 0 {
		return nil, fmt.Errorf("nono cannot install apt packages on the host; remove 'packages:' or install them yourself")
	}
	if cfg.Template != "" {
		return nil, fmt.Errorf("nono does not use Docker images; remove 'template:'")
	}
	if len(cfg.Ports) > 0 {
		return nil, fmt.Errorf("nono sessions are host processes; ports need no publishing — remove 'ports:'")
	}
	if _, ok := nonoAgents[cfg.Agent]; !ok {
		return nil, fmt.Errorf("agent %q is not supported by the nono backend; supported: claude-code, codex, opencode, shell", cfg.Agent)
	}

	names := make([]string, 0, len(cfg.Runtimes))
	for name := range cfg.Runtimes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		// config.Validate has already rejected unknown runtimes.
		rt := nonoRuntimes[name]
		if _, lookErr := nonoLookPath(rt.binary); lookErr != nil {
			return nil, fmt.Errorf("runtime %q declared in %s but %q was not found on this machine; install it or remove the runtime", name, cfg.ConfigPath, rt.binary)
		}
		pin := cfg.Runtimes[name]
		if pin == "" || !rt.checkVersion {
			continue
		}
		out, verErr := nonoRunVersion(rt.binary, rt.versionArg)
		if verErr != nil {
			return nil, fmt.Errorf("could not determine %s version (%s %s): %w", name, rt.binary, rt.versionArg, verErr)
		}
		hostVersion := extractVersion(out)
		if !versionMatches(pin, hostVersion) {
			return nil, fmt.Errorf("runtime %q pinned to %s but host has %s", name, pin, hostVersion)
		}
	}

	if cfg.Claude.Theme != "" || len(cfg.Claude.Plugins.Marketplaces) > 0 || len(cfg.Claude.Plugins.Install) > 0 || len(cfg.Claude.Settings) > 0 {
		warnings = append(warnings, "nono backend: ignoring 'claude:' settings (they would rewrite your real ~/.claude)")
	}
	if len(cfg.Shell.Aliases) > 0 {
		warnings = append(warnings, "nono backend: ignoring 'shell.aliases' (they would modify your real shell config)")
	}
	if cfg.Memory != "" {
		warnings = append(warnings, "nono backend: ignoring 'memory:' (kits are sbx-only)")
	}
	if len(cfg.Scripts.OnStart) > 0 {
		warnings = append(warnings, "nono backend: ignoring 'scripts.on_start' (nono has no per-session start hook)")
	}
	if len(cfg.Bridges) > 0 {
		warnings = append(warnings, "nono backend: ignoring 'bridges' (a nono agent runs on the host and reaches host services directly)")
	}
	if len(cfg.Php.Extensions) > 0 {
		warnings = append(warnings, "nono backend: ignoring 'php.extensions' (cannot install PHP extensions on the host)")
	}
	return warnings, nil
}

var versionRegexp = regexp.MustCompile(`\d+(\.\d+)*`)

// extractVersion pulls the first dotted-number token out of version-command
// output ("v22.1.0" → "22.1.0", "go version go1.24.1 …" → "1.24.1",
// "Python 3.12.3" → "3.12.3"). Empty string when none is found.
func extractVersion(output string) string {
	return versionRegexp.FindString(output)
}

// versionMatches reports whether hostVersion satisfies pin: every segment
// the pin specifies must equal the host's corresponding segment ("20"
// matches "20.11.1"; "1.24" matches "1.24.1" but not "1.23.0").
func versionMatches(pin, hostVersion string) bool {
	if hostVersion == "" {
		return false
	}
	pinParts := strings.Split(pin, ".")
	hostParts := strings.Split(hostVersion, ".")
	if len(hostParts) < len(pinParts) {
		return false
	}
	for i, p := range pinParts {
		if hostParts[i] != p {
			return false
		}
	}
	return true
}

// nonoProfile is the generated profile document. Field order here is the
// JSON output order; pointer fields are omitted entirely when unused.
type nonoProfile struct {
	Extends     []string         `json:"extends,omitempty"`
	Meta        nonoMeta         `json:"meta"`
	Groups      *nonoGroups      `json:"groups,omitempty"`
	Workdir     nonoWorkdir      `json:"workdir"`
	Filesystem  *nonoFilesystem  `json:"filesystem,omitempty"`
	Network     *nonoNetwork     `json:"network,omitempty"`
	Environment *nonoEnvironment `json:"environment,omitempty"`
}

type nonoMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type nonoGroups struct {
	Include []string `json:"include"`
}

type nonoWorkdir struct {
	Access string `json:"access"`
}

type nonoFilesystem struct {
	Allow []string `json:"allow"`
}

type nonoNetwork struct {
	AllowDomain []string `json:"allow_domain,omitempty"`
	OpenPort    []int    `json:"open_port,omitempty"`
}

type nonoEnvironment struct {
	SetVars map[string]string `json:"set_vars"`
}

// generateProfile renders the nono profile JSONC for cfg: a comment header
// marking the file as generated, then the JSON body. Output is
// deterministic (sorted groups; json.Marshal sorts set_vars keys) so
// regenerating from an unchanged config is a byte-identical file.
func generateProfile(cfg *config.Config) ([]byte, error) {
	agent := nonoAgents[cfg.Agent]

	p := nonoProfile{
		Meta: nonoMeta{
			Name:        cfg.Name,
			Description: fmt.Sprintf("Generated by blvckhole from %s. Do not edit.", cfg.ConfigPath),
		},
		Workdir: nonoWorkdir{Access: "readwrite"},
	}
	if agent.preset != "" {
		p.Extends = []string{agent.preset}
	}

	groupSet := map[string]bool{}
	for name := range cfg.Runtimes {
		rt := nonoRuntimes[name]
		if rt.group != "" {
			groupSet[rt.group] = true
		}
		if name == "bun" {
			p.Filesystem = &nonoFilesystem{Allow: []string{"$HOME/.bun"}}
		}
	}
	if len(groupSet) > 0 {
		groups := make([]string, 0, len(groupSet))
		for g := range groupSet {
			groups = append(groups, g)
		}
		sort.Strings(groups)
		p.Groups = &nonoGroups{Include: groups}
	}

	network := &nonoNetwork{AllowDomain: cfg.Network}
	if cfg.Handoff.Enabled {
		port, err := strconv.Atoi(cfg.HandoffPort())
		if err != nil {
			return nil, fmt.Errorf("cannot derive handoff port from %q: %w", cfg.Handoff.URL, err)
		}
		network.OpenPort = []int{port}
	}
	if len(network.AllowDomain) > 0 || len(network.OpenPort) > 0 {
		p.Network = network
	}

	if len(cfg.MergedEnv) > 0 {
		p.Environment = &nonoEnvironment{SetVars: cfg.MergedEnv}
	}

	body, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to render nono profile: %w", err)
	}

	header := fmt.Sprintf("// Generated by blvckhole from %s. Do not edit — edit blvckhole.yaml and run 'blvckhole start'.\n// Note: the 'workspace' key is ignored under nono; sessions run in the real project directory.\n", cfg.ConfigPath)
	return append([]byte(header), append(body, '\n')...), nil
}

// nonoSession is the subset of `nono ps --json` output we rely on. Field
// names are confirmed against a real nono install in the manual host
// checklist (spec §8); parseSessions tolerates both plain-array and
// wrapped-object shapes, mirroring parseSandboxList.
type nonoSession struct {
	ID      string `json:"id"`
	Profile string `json:"profile"`
}

func parseSessions(output []byte) ([]nonoSession, error) {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return nil, nil
	}

	var sessions []nonoSession
	if err := json.Unmarshal(output, &sessions); err == nil {
		return sessions, nil
	}

	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(output, &wrapped); err != nil {
		return nil, fmt.Errorf("failed to parse nono ps output: %w", err)
	}

	// An explicit "sessions" key wins even when empty — an empty
	// "warnings"/"errors" key elsewhere in the object must not shadow it.
	if raw, ok := wrapped["sessions"]; ok {
		if err := json.Unmarshal(raw, &sessions); err == nil {
			return sessions, nil
		}
		return nil, fmt.Errorf("unexpected 'nono ps --json' output: %s", truncateForError(trimmed))
	}

	// No "sessions" key: scan the remaining keys in sorted order (map
	// iteration order is random in Go) and use the first value that
	// unmarshals as a session array.
	keys := make([]string, 0, len(wrapped))
	for k := range wrapped {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if err := json.Unmarshal(wrapped[k], &sessions); err == nil {
			return sessions, nil
		}
	}

	return nil, fmt.Errorf("unexpected 'nono ps --json' output: %s", truncateForError(trimmed))
}

// truncateForError returns a short prefix of b suitable for embedding in an
// error message, so a malformed `nono ps --json` payload doesn't flood logs.
func truncateForError(b []byte) string {
	const maxLen = 200
	if len(b) <= maxLen {
		return string(b)
	}
	return string(b[:maxLen]) + "..."
}

// matchSessions filters sessions launched from our profile. nono may render
// the profile as its filesystem path or as the profile's meta.name — accept
// either.
func matchSessions(sessions []nonoSession, profilePath, name string) []nonoSession {
	var matched []nonoSession
	for _, s := range sessions {
		if s.Profile == profilePath || s.Profile == name {
			matched = append(matched, s)
		}
	}
	return matched
}

// NonoBackend drives nono (nono.sh): kernel-enforced cages (Seatbelt on
// macOS, Landlock on Linux) around normal host processes, configured by a
// generated profile file. There is no image, no ports, and no persistent
// sandbox — the profile is the identity and every run is a fresh supervised
// session in the real project directory.
type NonoBackend struct{}

func (n *NonoBackend) Name() string { return "nono" }

func (n *NonoBackend) EnsureAvailable() error {
	if _, err := exec.LookPath("nono"); err != nil {
		return fmt.Errorf("nono is not installed. Install it from: https://nono.sh")
	}
	return nil
}

// profileDir holds all generated nono state for the project:
// .config/blvckhole/nono/{profile.jsonc,config-hash}.
func (n *NonoBackend) profileDir(cfg *config.Config) string {
	return filepath.Join(cfg.ProjectDir, ".config", "blvckhole", "nono")
}

func (n *NonoBackend) profilePath(cfg *config.Config) string {
	return filepath.Join(n.profileDir(cfg), "profile.jsonc")
}

func (n *NonoBackend) hashPath(cfg *config.Config) string {
	return filepath.Join(n.profileDir(cfg), "config-hash")
}

func (n *NonoBackend) Exists(cfg *config.Config) bool {
	_, err := os.Stat(n.profilePath(cfg))
	return err == nil
}

// liveSessions returns the running nono sessions launched from our profile.
func (n *NonoBackend) liveSessions(cfg *config.Config) ([]nonoSession, error) {
	output, err := exec.Command("nono", "ps", "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("nono ps failed: %w", err)
	}
	sessions, err := parseSessions(output)
	if err != nil {
		return nil, err
	}
	return matchSessions(sessions, n.profilePath(cfg), cfg.Name), nil
}

func (n *NonoBackend) IsRunning(cfg *config.Config) bool {
	if !n.Exists(cfg) {
		return false
	}
	sessions, err := n.liveSessions(cfg)
	return err == nil && len(sessions) > 0
}

func (n *NonoBackend) Status(cfg *config.Config) (*Info, error) {
	if !n.Exists(cfg) {
		return nil, nil
	}
	sessions, err := n.liveSessions(cfg)
	if err != nil {
		return nil, err
	}
	status := "stopped"
	if len(sessions) > 0 {
		status = "running"
	}
	return &Info{
		ID:         n.profilePath(cfg),
		Name:       cfg.Name,
		Status:     status,
		Agent:      cfg.Agent,
		Workspaces: []string{cfg.ProjectDir},
	}, nil
}

func (n *NonoBackend) Run(cfg *config.Config, extraArgs ...string) error {
	agent, ok := nonoAgents[cfg.Agent]
	if !ok {
		return fmt.Errorf("agent %q is not supported by the nono backend; supported: claude-code, codex, opencode, shell", cfg.Agent)
	}
	args := []string{"run", "--profile", n.profilePath(cfg), "--", agent.binary}
	args = append(args, extraArgs...)
	cmd := exec.Command("nono", args...)
	cmd.Dir = cfg.ProjectDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunCommand runs a shell command in a fresh sibling session. The workspace
// path is a container-ism with no host meaning, so the command always runs
// in the real project directory.
func (n *NonoBackend) RunCommand(cfg *config.Config, command string) error {
	script := fmt.Sprintf("cd %s && %s", shellQuote(cfg.ProjectDir), command)
	cmd := exec.Command("nono", "run", "--profile", n.profilePath(cfg), "--", "bash", "-c", script)
	cmd.Dir = cfg.ProjectDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (n *NonoBackend) ExecSilent(cfg *config.Config, command ...string) (string, error) {
	args := []string{"run", "--profile", n.profilePath(cfg), "--"}
	args = append(args, command...)
	cmd := exec.Command("nono", args...)
	cmd.Dir = cfg.ProjectDir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// Shell replaces the current process with an interactive sandboxed bash.
// dir is honored only when it is an absolute path that exists on the host —
// cfg.Workspace values like /workspace/x are container paths with no host
// meaning — otherwise the shell starts in the project directory.
func (n *NonoBackend) Shell(cfg *config.Config, dir string) error {
	target := cfg.ProjectDir
	if filepath.IsAbs(dir) {
		if _, err := os.Stat(dir); err == nil {
			target = dir
		}
	}

	nonoPath, err := exec.LookPath("nono")
	if err != nil {
		return fmt.Errorf("nono not found in PATH: %w", err)
	}

	script := fmt.Sprintf("cd %s && exec bash", shellQuote(target))
	execArgs := []string{"nono", "run", "--profile", n.profilePath(cfg), "--", "bash", "-c", script}
	return syscall.Exec(nonoPath, execArgs, os.Environ())
}

// Stop stops every live session launched from our profile.
func (n *NonoBackend) Stop(cfg *config.Config) error {
	sessions, err := n.liveSessions(cfg)
	if err != nil {
		return err
	}
	for _, s := range sessions {
		cmd := exec.Command("nono", "stop", s.ID)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("nono stop %s failed: %w", s.ID, err)
		}
	}
	return nil
}

// Remove stops live sessions and deletes the generated profile and hash.
func (n *NonoBackend) Remove(cfg *config.Config) error {
	if err := n.Stop(cfg); err != nil {
		return err
	}
	return os.RemoveAll(n.profileDir(cfg))
}

func (n *NonoBackend) AllowNetwork(cfg *config.Config, domains []string) error {
	return fmt.Errorf("nono applies network policy when a session launches and cannot change it at runtime. Add the domain to 'network:' in %s and run 'blvckhole restart'.", cfg.ConfigPath)
}

func (n *NonoBackend) DenyNetwork(cfg *config.Config, domains []string) error {
	return fmt.Errorf("nono applies network policy when a session launches and cannot change it at runtime. nono blocks every domain not allowed; remove it from 'network:' in %s and run 'blvckhole restart'.", cfg.ConfigPath)
}

func (n *NonoBackend) RemoveNetwork(cfg *config.Config, domains []string) error {
	return fmt.Errorf("nono applies network policy when a session launches and cannot change it at runtime. Remove the domain from 'network:' in %s and run 'blvckhole restart'.", cfg.ConfigPath)
}

func (n *NonoBackend) ReadConfigHash(cfg *config.Config) (string, error) {
	data, err := os.ReadFile(n.hashPath(cfg))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (n *NonoBackend) WriteConfigHash(cfg *config.Config, hash string) error {
	if err := os.MkdirAll(n.profileDir(cfg), 0755); err != nil {
		return err
	}
	return os.WriteFile(n.hashPath(cfg), []byte(hash+"\n"), 0644)
}

// PrepareAgent is a no-op: the host agent already runs with the user's real
// ~/.claude settings, and rewriting them is exactly what nono must not do.
func (n *NonoBackend) PrepareAgent(cfg *config.Config) error {
	return nil
}

var _ Backend = (*NonoBackend)(nil)

// Provision validates the config against nono's capabilities, generates and
// validates the profile, runs on_create commands in sandboxed sessions, and
// stores the config hash. No long-lived session is launched — nono sessions
// are per-run, so a matching-hash profile that already exists is a cheap
// no-op instead of a destroy-and-recreate (idle nono state is normal, not
// "stale"). Validation errors surface before anything is written or removed;
// warnings print only when validation passes. Any failure after the profile
// file is written removes it again, so Exists() only ever reports a
// successfully provisioned profile.
func (n *NonoBackend) Provision(cfg *config.Config) error {
	if n.Exists(cfg) {
		stored, hashErr := n.ReadConfigHash(cfg)
		current := cfg.FileHash()
		if hashErr == nil && stored != "" && current != "" && stored == current {
			fmt.Println(ui.Info.Render("Profile up to date."))
			return nil
		}
	}

	warnings, err := validateForNono(cfg)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		fmt.Println(ui.Warn.Render(w))
	}

	if n.Exists(cfg) {
		// Idle sandbox, config changed (or hash missing/unreadable): full
		// re-provision, so on_create runs again against the new config.
		// Deferred until after validation so an invalid new config never
		// destroys the last good profile.
		if err := os.RemoveAll(n.profileDir(cfg)); err != nil {
			return fmt.Errorf("failed to remove stale profile: %w", err)
		}
	}

	fmt.Println(ui.Accent.Render("Generating nono profile..."))
	profile, err := generateProfile(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(n.profileDir(cfg), 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}
	if err := os.WriteFile(n.profilePath(cfg), profile, 0644); err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}

	fmt.Println(ui.Accent.Render("Validating profile..."))
	cmd := exec.Command("nono", "profile", "validate", n.profilePath(cfg))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.RemoveAll(n.profileDir(cfg))
		return fmt.Errorf("nono profile validate failed: %w", err)
	}

	for _, command := range cfg.Scripts.OnCreate {
		fmt.Println(ui.Accent.Render("Running: " + command))
		if err := n.RunCommand(cfg, command); err != nil {
			os.RemoveAll(n.profileDir(cfg))
			return fmt.Errorf("on_create command failed (%s): %w", command, err)
		}
	}

	if hash := cfg.FileHash(); hash != "" {
		if err := n.WriteConfigHash(cfg, hash); err != nil {
			os.RemoveAll(n.profileDir(cfg))
			return fmt.Errorf("failed to store config hash: %w", err)
		}
	}

	return nil
}
