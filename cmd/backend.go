package cmd

import (
	"fmt"
	"strings"

	"github.com/neoighodaro/blvckhole/internal/backend"
	"github.com/neoighodaro/blvckhole/internal/config"
)

// loadBackend resolves the configured sandbox backend and verifies its CLI
// is installed. It replaces the old sbx pre-check, which ran before the
// config was even loaded; the backend choice now comes from the config, so
// commands load config first.
func loadBackend(cfg *config.Config) (backend.Backend, error) {
	b := backend.Get(cfg.Backend)
	if b == nil {
		return nil, fmt.Errorf("unknown backend %q in %s: must be one of: %s",
			cfg.Backend, cfg.ConfigPath, strings.Join(backend.Names(), ", "))
	}
	if err := b.EnsureAvailable(); err != nil {
		return nil, err
	}
	return b, nil
}
