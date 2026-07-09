package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sandbox status",
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

		info, err := b.Status(cfg)
		if err != nil {
			return err
		}

		if info == nil {
			fmt.Println(ui.Info.Render("Sandbox not created yet."))
			fmt.Println(ui.Info.Render("  Run 'blvckhole start' to create it."))
			return nil
		}

		label := lipgloss.NewStyle().Foreground(ui.Gray).Width(12)
		value := lipgloss.NewStyle().Foreground(ui.White)

		statusStyle := lipgloss.NewStyle().Bold(true)
		switch info.Status {
		case "running":
			statusStyle = statusStyle.Foreground(ui.Green)
		case "stopped":
			statusStyle = statusStyle.Foreground(ui.Red)
		default:
			statusStyle = statusStyle.Foreground(ui.Gray)
		}

		fmt.Println()
		fmt.Println(label.Render("  Sandbox") + value.Render(cfg.Name))
		fmt.Println(label.Render("  Status") + statusStyle.Render(info.Status))
		fmt.Println(label.Render("  Agent") + value.Render(info.Agent))
		fmt.Println(label.Render("  ID") + lipgloss.NewStyle().Foreground(ui.Dim).Render(info.ID))

		if len(info.Workspaces) > 0 {
			fmt.Println(label.Render("  Workspace") + value.Render(strings.Join(info.Workspaces, ", ")))
		}

		if len(cfg.Ports) > 0 {
			fmt.Println(label.Render("  Ports") + value.Render(strings.Join(cfg.Ports, ", ")))
		}

		if len(cfg.Network) > 0 {
			fmt.Println(label.Render("  Network") + value.Render(strings.Join(cfg.Network, ", ")))
		}

		if len(cfg.Runtimes) > 0 {
			runtimes := make([]string, 0, len(cfg.Runtimes))
			for name, version := range cfg.Runtimes {
				runtimes = append(runtimes, name+" "+version)
			}
			fmt.Println(label.Render("  Runtimes") + value.Render(strings.Join(runtimes, ", ")))
		}

		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
