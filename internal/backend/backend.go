// Package backend abstracts the sandbox runtime (docker sbx today, nono
// planned) behind a capability-shaped interface. Follow the pattern of
// internal/runtime: implementations live in this package and are looked up
// by name in a static registry.
package backend

import (
	"sort"

	"github.com/neoighodaro/blvckhole/internal/config"
)

// Info describes a sandbox as reported by the backend.
type Info struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	Agent      string   `json:"agent"`
	Workspaces []string `json:"workspaces"`
}

// Backend is a sandbox runtime. Methods take the full config because
// different backends key off different fields (sbx uses cfg.Name; nono will
// use a generated profile). Provisioning is a single method — not
// create/link/publish verbs — because backends differ radically in how an
// environment comes to exist (sbx: image build + kit + create; nono:
// profile generation, no image, no ports).
type Backend interface {
	// Name is the registry key ("sbx").
	Name() string
	// EnsureAvailable errors with install instructions when the backend's
	// CLI is not on PATH.
	EnsureAvailable() error

	Exists(cfg *config.Config) bool
	IsRunning(cfg *config.Config) bool
	Status(cfg *config.Config) (*Info, error)

	// Provision builds whatever the backend needs (image, kit, profile) and
	// brings the sandbox to a running, fully configured state. It prints its
	// own progress lines and stores the config hash on success.
	Provision(cfg *config.Config) error

	// Run launches the sandbox's configured agent, attached to the terminal.
	Run(cfg *config.Config, extraArgs ...string) error
	// RunCommand runs a shell command in the sandbox's workspace directory
	// (cfg.Workspace, falling back to cfg.ProjectDir), attached to the
	// terminal.
	RunCommand(cfg *config.Config, command string) error
	// ExecSilent runs a command in the sandbox and returns combined output.
	ExecSilent(cfg *config.Config, command ...string) (string, error)
	// Shell replaces the current process with an interactive shell in the
	// sandbox, starting in dir when non-empty. It only returns on error.
	Shell(cfg *config.Config, dir string) error

	Stop(cfg *config.Config) error
	Remove(cfg *config.Config) error

	AllowNetwork(cfg *config.Config, domains []string) error
	DenyNetwork(cfg *config.Config, domains []string) error
	RemoveNetwork(cfg *config.Config, domains []string) error

	// ReadConfigHash and WriteConfigHash persist the hash of blvckhole.yaml
	// in backend-specific state, used to detect config drift.
	ReadConfigHash(cfg *config.Config) (string, error)
	WriteConfigHash(cfg *config.Config, hash string) error
}

var registry = map[string]Backend{
	"sbx": &SbxBackend{},
}

// Get returns the backend registered under name, or nil.
func Get(name string) Backend {
	return registry[name]
}

// Names returns the registered backend names, sorted.
func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
