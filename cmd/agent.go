package cmd

import (
	"fmt"
	"os"

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

		if !sandbox.IsRunning(cfg.Name) {
			if err := runStart(cfg); err != nil {
				return err
			}
		}

		if err := mergeAgentSettings(cfg); err != nil {
			fmt.Println(ui.Info.Render("Warning: could not merge agent settings: " + err.Error()))
		}

		fmt.Println(ui.Accent.Render("Starting agent..."))
		return sandbox.Run(cfg.Name, args...)
	},
}

func mergeAgentSettings(cfg *config.Config) error {
	script := `
export SLPATH="$(ls -d ~/.claude/plugins/cache/claude-dashboard/claude-dashboard/*/dist/index.js 2>/dev/null | sort -V | tail -1)"
if [ -f ~/.claude/settings.json ]; then
  node -e "
    const fs=require('fs'),h=process.env.HOME,p=h+'/.claude/settings.json';
    const s=JSON.parse(fs.readFileSync(p,'utf8'));
    const tmpl=h+'/.claude/settings.sandbox.json';
    if(fs.existsSync(tmpl)){
      const custom=JSON.parse(fs.readFileSync(tmpl,'utf8'));
      Object.assign(s,custom);
    }
    if(process.env.SLPATH){
      s.statusLine={type:'command',command:'node '+process.env.SLPATH};
    }
    if(!s.enabledPlugins) s.enabledPlugins={};
`

	for _, plugin := range cfg.Claude.Plugins.Install {
		script += fmt.Sprintf("    s.enabledPlugins['%s']=true;\n", plugin)
	}

	script += `
    const themeFile=h+'/.claude/themes/sandbox.json';
    if(fs.existsSync(themeFile)){s.theme='custom:sandbox';s.themeId='custom:sandbox';}
    fs.writeFileSync(p,JSON.stringify(s,null,2));
  " 2>/dev/null
fi
`

	_, err := sandbox.ExecSilent(cfg.Name, "bash", "-c", script)
	return err
}

func init() {
	agentCmd.Flags().BoolVar(&rebuildFlag, "rebuild", false, "Force rebuild the sandbox image before starting")
	rootCmd.AddCommand(agentCmd)
}
