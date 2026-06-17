package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/neoighodaro/blvckhole/internal/handoff"
	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var (
	handoffPort  int
	handoffBind  string
	handoffStore string
)

var handoffCmd = &cobra.Command{
	Use:   "handoff",
	Short: "Run the cross-sandbox handoff broker (foreground; stops when you close the terminal)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
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
	},
}

func init() {
	handoffCmd.Flags().IntVar(&handoffPort, "port", 8787, "port for the handoff broker")
	handoffCmd.Flags().StringVar(&handoffBind, "bind", "127.0.0.1", "bind address (loopback by default)")
	handoffCmd.Flags().StringVar(&handoffStore, "store", "", "override the store path (default: XDG/HOME config)")
	rootCmd.AddCommand(handoffCmd)
}
