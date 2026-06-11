package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/neoighodaro/blvckhole/internal/config"
	"github.com/neoighodaro/blvckhole/internal/sandbox"
	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var rebuildFlag bool

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Launch the AI agent in the sandbox",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureSbxInstalled(); err != nil {
			return err
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		cfg, err := loadConfig(cwd)
		if err != nil {
			return err
		}

		if rebuildFlag {
			if sandbox.IsRunning(cfg.Name) {
				fmt.Println(ui.Info.Render("Removing existing sandbox..."))
				sandbox.Remove(cfg.Name)
			}
		}

		if !sandbox.Exists(cfg.Name) {
			if err := runStart(cfg); err != nil {
				return err
			}
		}

		if err := mergeAgentSettings(cfg); err != nil {
			fmt.Println(ui.Info.Render("Warning: could not merge agent settings: " + err.Error()))
		}

		renameZellijTab(cfg)

		fmt.Println(ui.Accent.Render("Starting agent..."))
		return sandbox.Run(cfg.Name, args...)
	},
}

func mergeAgentSettings(cfg *config.Config) error {
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

	_, err := sandbox.ExecSilent(cfg.Name, "bash", "-c", script)
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

	tabName := " \uf023 " + cfg.Zellij.DisplayName

	out, err := exec.Command("zellij", "action", "current-tab-info").Output()
	if err != nil {
		return
	}

	currentTab := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	currentTab = strings.TrimPrefix(currentTab, "name: ")

	if currentTab != tabName {
		exec.Command("zellij", "action", "rename-tab", tabName).Run()
	}
}

func init() {
	agentCmd.Flags().BoolVar(&rebuildFlag, "rebuild", false, "Force rebuild the sandbox image before starting")
	rootCmd.AddCommand(agentCmd)
}
