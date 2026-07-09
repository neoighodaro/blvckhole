package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/neoighodaro/blvckhole/internal/config"
	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var (
	allowPersist  bool
	removePersist bool
)

var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Manage the sandbox firewall (domain allow/deny rules)",
}

var networkAllowCmd = &cobra.Command{
	Use:   "allow <domain>...",
	Short: "Allow outbound access to one or more domains",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNetwork("allow", args, allowPersist)
	},
}

var networkDenyCmd = &cobra.Command{
	Use:   "deny <domain>...",
	Short: "Deny outbound access to one or more domains (runtime-only, not persisted)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNetwork("deny", args, false)
	},
}

var networkRemoveCmd = &cobra.Command{
	Use:   "remove <domain>...",
	Short: "Remove existing firewall rules for one or more domains",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNetwork("remove", args, removePersist)
	},
}

func runNetwork(action string, domains []string, persist bool) error {
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

	exists := b.Exists(cfg)
	if !exists && !persist {
		if action == "deny" {
			return fmt.Errorf("sandbox %q is not created; run 'blvckhole start' first (deny rules are runtime-only and cannot be saved to config)", cfg.Name)
		}
		return fmt.Errorf("sandbox %q is not created; run 'blvckhole start' first, or pass --persist to save this to config", cfg.Name)
	}

	if exists {
		var applyErr error
		switch action {
		case "allow":
			applyErr = b.AllowNetwork(cfg, domains)
		case "deny":
			applyErr = b.DenyNetwork(cfg, domains)
		case "remove":
			applyErr = b.RemoveNetwork(cfg, domains)
		}
		if applyErr != nil {
			return fmt.Errorf("failed to apply network policy: %w", applyErr)
		}
		fmt.Println(ui.Success.Render(fmt.Sprintf("Applied %s rule for: %s", action, strings.Join(domains, ", "))))
	} else {
		fmt.Println(ui.Info.Render(fmt.Sprintf("Sandbox not created; %s will apply on next 'blvckhole start'.", action)))
	}

	if persist {
		var persistErr error
		switch action {
		case "allow":
			persistErr = config.AddNetworkDomains(cfg.ConfigPath, domains)
		case "remove":
			persistErr = config.RemoveNetworkDomains(cfg.ConfigPath, domains)
		}
		if persistErr != nil {
			return fmt.Errorf("failed to update config: %w", persistErr)
		}
		fmt.Println(ui.Success.Render("Updated config: " + cfg.ConfigPath))
	}

	return nil
}

func init() {
	networkAllowCmd.Flags().BoolVar(&allowPersist, "persist", false, "also save the change to blvckhole.yaml")
	networkRemoveCmd.Flags().BoolVar(&removePersist, "persist", false, "also remove the domains from blvckhole.yaml")

	networkCmd.AddCommand(networkAllowCmd)
	networkCmd.AddCommand(networkDenyCmd)
	networkCmd.AddCommand(networkRemoveCmd)
	rootCmd.AddCommand(networkCmd)
}
