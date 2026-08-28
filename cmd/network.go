package cmd

import (
	"fmt"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/neoighodaro/blvckhole/internal/backend"
	"github.com/neoighodaro/blvckhole/internal/config"
	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

// maxAllowMins caps a timed --mins allow. Keeping the window short bounds how
// long the command blocks the terminal holding the countdown open.
const maxAllowMins = 30

var (
	allowPersist  bool
	allowMins     int
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
		if err := validateTimedAllow(allowMins, allowPersist); err != nil {
			return err
		}
		return runNetwork("allow", args, allowPersist, allowMins)
	},
}

var networkDenyCmd = &cobra.Command{
	Use:   "deny <domain>...",
	Short: "Deny outbound access to one or more domains (runtime-only, not persisted)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNetwork("deny", args, false, 0)
	},
}

var networkRemoveCmd = &cobra.Command{
	Use:   "remove <domain>...",
	Short: "Remove existing firewall rules for one or more domains",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNetwork("remove", args, removePersist, 0)
	},
}

func runNetwork(action string, domains []string, persist bool, mins int) error {
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

	// A timed allow blocks here counting down, then reverts the runtime rule.
	// It only reaches this point with a live sandbox: mins>0 implies
	// persist==false (validateTimedAllow), so the !exists guard above would
	// already have errored when no sandbox exists.
	if mins > 0 && action == "allow" && exists {
		return runAllowCountdown(b, cfg, domains, mins)
	}

	return nil
}

// validateTimedAllow checks the --mins value and its interaction with
// --persist. mins == 0 means the flag is unset (timed mode off).
func validateTimedAllow(mins int, persist bool) error {
	if mins == 0 {
		return nil
	}
	if persist {
		return fmt.Errorf("--mins cannot be combined with --persist: a timed allow is runtime-only and auto-reverts")
	}
	if mins < 1 || mins > maxAllowMins {
		return fmt.Errorf("--mins must be between 1 and %d, got %d", maxAllowMins, mins)
	}
	return nil
}

// runAllowCountdown blocks for mins minutes, rendering a live countdown bar,
// then removes the runtime allow rule. It reverts early if the window elapses
// or the process is interrupted (Ctrl-C, terminal close, SIGTERM), so the
// allow never outlives the command.
func runAllowCountdown(b backend.Backend, cfg *config.Config, domains []string, mins int) error {
	const barWidth = 24
	total := time.Duration(mins) * time.Minute
	deadline := time.Now().Add(total)
	label := strings.Join(domains, ", ")

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigc)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	printCountdown(label, 0, total, barWidth)

	interrupted := false
loop:
	for {
		select {
		case <-sigc:
			interrupted = true
			break loop
		case now := <-ticker.C:
			remaining := deadline.Sub(now)
			if remaining <= 0 {
				printCountdown(label, total, total, barWidth)
				break loop
			}
			printCountdown(label, total-remaining, total, barWidth)
		}
	}

	// Clear the countdown line before printing the outcome.
	fmt.Print("\r\033[K")

	if err := b.RemoveNetwork(cfg, domains); err != nil {
		return fmt.Errorf("failed to auto-revert allow for %s: %w", label, err)
	}

	if interrupted {
		fmt.Println(ui.Warn.Render("Interrupted — reverted allow for: " + label))
	} else {
		fmt.Println(ui.Success.Render("Time's up — reverted allow for: " + label))
	}
	return nil
}

// printCountdown redraws the countdown line in place using a carriage return
// and a clear-to-end-of-line escape.
func printCountdown(label string, elapsed, total time.Duration, barWidth int) {
	fmt.Print("\r\033[K" + ui.Accent.Render(renderCountdownLine(label, elapsed, total, barWidth)))
}

// renderCountdownLine builds the plain (unstyled) countdown line:
// "<label>  [████░░░░] M:SS remaining".
func renderCountdownLine(label string, elapsed, total time.Duration, barWidth int) string {
	return fmt.Sprintf("%s  [%s] %s remaining", label, countdownBar(elapsed, total, barWidth), formatRemaining(total-elapsed))
}

// countdownBar renders a fixed-width bar that fills as elapsed approaches
// total. The result is always exactly barWidth runes wide.
func countdownBar(elapsed, total time.Duration, barWidth int) string {
	if barWidth < 1 {
		barWidth = 1
	}
	frac := 0.0
	if total > 0 {
		frac = float64(elapsed) / float64(total)
	}
	frac = math.Max(0, math.Min(1, frac))
	filled := int(frac * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
}

// formatRemaining renders a duration as M:SS, rounding up to the next whole
// second and never going negative.
func formatRemaining(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	secs := int(math.Ceil(d.Seconds()))
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}

func init() {
	networkAllowCmd.Flags().BoolVar(&allowPersist, "persist", false, "also save the change to blvckhole.yaml")
	networkAllowCmd.Flags().IntVar(&allowMins, "mins", 0, fmt.Sprintf("auto-revert the allow after N minutes (1-%d); blocks showing a countdown, incompatible with --persist", maxAllowMins))
	networkRemoveCmd.Flags().BoolVar(&removePersist, "persist", false, "also remove the domains from blvckhole.yaml")

	networkCmd.AddCommand(networkAllowCmd)
	networkCmd.AddCommand(networkDenyCmd)
	networkCmd.AddCommand(networkRemoveCmd)
	rootCmd.AddCommand(networkCmd)
}
