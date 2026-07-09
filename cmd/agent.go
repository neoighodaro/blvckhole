package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/neoighodaro/blvckhole/internal/backend"
	"github.com/neoighodaro/blvckhole/internal/config"
	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var rebuildFlag bool

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Launch the AI agent in the sandbox",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		cfg, err := loadConfig(cwd)
		if err != nil {
			return err
		}

		b, err := loadBackend(cfg)
		if err != nil {
			return err
		}

		// Rename the tab up front, while the launching pane is still focused.
		// Doing this after the (potentially slow) sandbox startup would risk
		// renaming whatever tab the user switched to in the meantime.
		renameZellijTab(cfg)

		if rebuildFlag {
			// Rebuild must tear down whatever exists — including a stopped
			// (existing but not running) sandbox — otherwise the !Exists check
			// below skips runStart and the agent boots in the stale sandbox.
			if b.Exists(cfg) {
				fmt.Println(ui.Info.Render("Removing existing sandbox..."))
				if err := b.Remove(cfg); err != nil {
					return fmt.Errorf("failed to remove sandbox: %w", err)
				}
			}
		}

		if !b.Exists(cfg) {
			if err := runStart(b, cfg); err != nil {
				return err
			}
		} else if b.IsRunning(cfg) && configChanged(b, cfg) {
			fmt.Println(ui.Warn.Render("Config has changed since this sandbox was created."))
			fmt.Println(ui.Info.Render("  Run 'blvckhole agent --rebuild' to apply the changes."))
		}

		if err := mergeAgentSettings(b, cfg); err != nil {
			fmt.Println(ui.Info.Render("Warning: could not merge agent settings: " + err.Error()))
		}

		fmt.Println(ui.Accent.Render("Starting agent..."))
		return b.Run(cfg, args...)
	},
}

func mergeAgentSettings(b backend.Backend, cfg *config.Config) error {
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

	_, err := b.ExecSilent(cfg, "bash", "-c", script)
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

func renameZellijTab(cfg *config.Config) {
	if cfg.Zellij.DisplayName == "" {
		return
	}

	if _, err := exec.LookPath("zellij"); err != nil {
		return
	}

	// zellij's `rename-tab` always targets the *focused* tab, not the tab that
	// launched this process. Only rename when our own pane is the focused one,
	// so switching to another tab while a sandbox starts can't rename it.
	if !paneIsFocused() {
		return
	}

	out, err := exec.Command("zellij", "action", "list-tabs", "--state", "--json").Output()
	if err != nil {
		return
	}

	active, ok := activeZellijTab(string(out))
	if !ok {
		return
	}

	// Don't hijack a tab's name when launched from an ad-hoc floating pane \u2014 the
	// floating pane is a transient overlay, not the tab's identity. Our pane is
	// the focused one (checked above), so visible floating panes in the active
	// tab mean we were launched from a floating pane.
	if active.FloatingVisible {
		return
	}

	tabName := "\uf023 " + cfg.Zellij.DisplayName
	if active.Name != tabName {
		exec.Command("zellij", "action", "rename-tab", tabName).Run()
	}
}

// zellijTab is the subset of `zellij action list-tabs --json` we care about.
type zellijTab struct {
	Name            string `json:"name"`
	Active          bool   `json:"active"`
	FloatingVisible bool   `json:"are_floating_panes_visible"`
}

// activeZellijTab returns the active (focused) tab from `list-tabs --json`
// output. The second return is false if the output can't be parsed or no tab is
// marked active.
func activeZellijTab(listTabsJSON string) (zellijTab, bool) {
	var tabs []zellijTab
	if err := json.Unmarshal([]byte(listTabsJSON), &tabs); err != nil {
		return zellijTab{}, false
	}
	for _, t := range tabs {
		if t.Active {
			return t, true
		}
	}
	return zellijTab{}, false
}

// paneIsFocused reports whether the pane this process runs in is the one
// currently focused in the zellij session. It compares our $ZELLIJ_PANE_ID
// against the focused pane reported by `zellij action list-clients`, whose
// ZELLIJ_PANE_ID column is rendered as e.g. "terminal_5".
func paneIsFocused() bool {
	paneID := os.Getenv("ZELLIJ_PANE_ID")
	if paneID == "" {
		return false
	}

	out, err := exec.Command("zellij", "action", "list-clients").Output()
	if err != nil {
		return false
	}

	return clientsFocusPane(string(out), paneID)
}

// clientsFocusPane reports whether any client in `zellij action list-clients`
// output is focused on the pane with the given $ZELLIJ_PANE_ID. The PANE_ID
// column is rendered as e.g. "terminal_5".
func clientsFocusPane(listClientsOutput, paneID string) bool {
	want := "terminal_" + paneID
	lines := strings.Split(listClientsOutput, "\n")
	for _, line := range lines[1:] { // skip header row
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == want {
			return true
		}
	}

	return false
}

func init() {
	agentCmd.Flags().BoolVar(&rebuildFlag, "rebuild", false, "Force rebuild the sandbox image before starting")
	rootCmd.AddCommand(agentCmd)
}
