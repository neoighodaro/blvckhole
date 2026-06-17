package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/neoighodaro/blvckhole/internal/handoff"
	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var (
	handoffPort   int
	handoffBind   string
	handoffStore  string
	handoffDaemon bool
	handoffKill   bool
)

var handoffCmd = &cobra.Command{
	Use:   "handoff",
	Short: "Run the cross-sandbox handoff broker",
	Long: "Run the cross-sandbox handoff broker.\n\n" +
		"By default it runs in the foreground and stops when you close the terminal.\n" +
		"Use --daemon to run it in the background, and --kill to stop a running one.",
	RunE: func(cmd *cobra.Command, args []string) error {
		pidPath := handoff.DefaultPidPath()

		if handoffKill {
			return killBroker(pidPath)
		}

		// Refuse to start a second broker; they would fight over the same port and
		// clobber each other's pid file.
		if pid, err := readPidFile(pidPath); err == nil && processAlive(pid) {
			return fmt.Errorf("a handoff broker is already running (pid %d); stop it with `blvckhole handoff --kill`", pid)
		}

		if handoffDaemon {
			return startBrokerDaemon(pidPath)
		}

		return runBroker(cmd.Context(), pidPath)
	},
}

// runBroker serves the broker in the foreground. This is also the code path the
// backgrounded child runs (it is re-exec'd without --daemon).
func runBroker(ctx context.Context, pidPath string) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	path := handoffStore
	if path == "" {
		path = handoff.DefaultStorePath()
	}
	store := handoff.NewStore(path)

	addr := net.JoinHostPort(handoffBind, strconv.Itoa(handoffPort))
	srv := &http.Server{Addr: addr, Handler: handoff.NewServer(store)}

	boardURL := "http://" + addr + "/handoff"
	sandboxURL := fmt.Sprintf("http://host.docker.internal:%d", handoffPort)

	// Record our PID so `--kill` can find us, and clean it up on the way out.
	if err := writePidFile(pidPath); err != nil {
		fmt.Fprintln(os.Stderr, ui.Warn.Render("could not write pid file: "+err.Error()))
	} else {
		defer os.Remove(pidPath)
	}

	fmt.Println(ui.Info.Render("Handoff broker on " + boardURL + " — Ctrl-C or close terminal to stop"))
	fmt.Println(ui.Info.Render("Sandboxes reach it at " + sandboxURL))

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// startBrokerDaemon re-execs this binary in its own session so it survives the
// terminal closing, pipes its output to a log file, and returns. The child
// records its own PID via runBroker.
func startBrokerDaemon(pidPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	logPath := handoff.DefaultLogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return err
	}
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer logf.Close()

	childArgs := []string{"handoff", "--port", strconv.Itoa(handoffPort), "--bind", handoffBind}
	if handoffStore != "" {
		childArgs = append(childArgs, "--store", handoffStore)
	}

	child := exec.Command(exe, childArgs...)
	child.Stdout = logf
	child.Stderr = logf
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // detach into a new session
	if err := child.Start(); err != nil {
		return err
	}
	pid := child.Process.Pid
	_ = child.Process.Release() // let the parent exit without leaving a zombie

	// Give the child a moment to bind; if it died (e.g. port in use) say so.
	time.Sleep(300 * time.Millisecond)
	if !processAlive(pid) {
		return fmt.Errorf("handoff broker exited immediately — see %s", logPath)
	}

	addr := net.JoinHostPort(handoffBind, strconv.Itoa(handoffPort))
	fmt.Println(ui.Success.Render(fmt.Sprintf("Handoff broker started in background (pid %d) on http://%s/handoff", pid, addr)))
	fmt.Println(ui.Info.Render("Logs: " + logPath + "  ·  stop with `blvckhole handoff --kill`"))
	return nil
}

// killBroker stops a broker recorded in the pid file.
func killBroker(pidPath string) error {
	pid, err := readPidFile(pidPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println(ui.Info.Render("No handoff broker is running."))
			return nil
		}
		return err
	}
	if !processAlive(pid) {
		_ = os.Remove(pidPath)
		fmt.Println(ui.Info.Render("No handoff broker is running (cleaned up a stale pid file)."))
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("could not stop broker (pid %d): %w", pid, err)
	}

	// Wait briefly for it to exit and clear its own pid file.
	for i := 0; i < 50; i++ {
		if !processAlive(pid) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = os.Remove(pidPath)
	fmt.Println(ui.Success.Render(fmt.Sprintf("Stopped handoff broker (pid %d).", pid)))
	return nil
}

func writePidFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0644)
}

func readPidFile(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(b)))
}

// processAlive reports whether pid refers to a live process (signal 0 probe).
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func init() {
	handoffCmd.Flags().IntVarP(&handoffPort, "port", "P", 8787, "port for the handoff broker")
	handoffCmd.Flags().BoolVarP(&handoffDaemon, "daemon", "D", false, "run the broker in the background")
	handoffCmd.Flags().BoolVarP(&handoffKill, "kill", "K", false, "stop a running background broker")
	handoffCmd.Flags().StringVar(&handoffBind, "bind", "127.0.0.1", "bind address (loopback by default)")
	handoffCmd.Flags().StringVar(&handoffStore, "store", "", "override the store path (default: XDG/HOME config)")
	rootCmd.AddCommand(handoffCmd)
}
