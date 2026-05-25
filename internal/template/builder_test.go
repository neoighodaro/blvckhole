package template

import (
	"strings"
	"testing"

	"github.com/neoighodaro/blvckhole/internal/config"
)

func TestRender_MinimalConfig(t *testing.T) {
	cfg := &config.Config{
		Name:  "test",
		Agent: "claude-code",
	}

	out, err := Render(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "FROM docker/sandbox-templates:claude-code-minimal") {
		t.Error("expected base image in output")
	}
	if strings.Contains(out, "zsh") {
		t.Error("should not have zsh packages in minimal config")
	}
	if strings.Contains(out, "ripgrep") {
		t.Error("should not have user packages with no packages configured")
	}
}

func TestRender_WithPackages(t *testing.T) {
	cfg := &config.Config{
		Name:     "test",
		Agent:    "claude-code",
		Packages: []string{"ripgrep", "jq"},
	}

	out, err := Render(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "ripgrep jq") {
		t.Errorf("expected packages in apt-get, got:\n%s", out)
	}
}

func TestRender_WithRuntimes(t *testing.T) {
	cfg := &config.Config{
		Name:     "test",
		Agent:    "claude-code",
		Runtimes: map[string]string{"node": "22", "bun": "latest"},
	}

	out, err := Render(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "setup_22.x") {
		t.Error("expected node 22 setup in output")
	}
	if !strings.Contains(out, "bun-vlatest") {
		t.Error("expected bun install in output")
	}
	if !strings.Contains(out, "BUN_INSTALL") {
		t.Error("expected BUN_INSTALL env in output")
	}
}

func TestRender_WithPlugins(t *testing.T) {
	cfg := &config.Config{
		Name:  "test",
		Agent: "claude-code",
		Claude: config.ClaudeConfig{
			Plugins: config.ClaudePlugins{
				Marketplaces: []string{"anthropics/claude-plugins-official"},
				Install:      []string{"superpowers@claude-plugins-official"},
			},
		},
	}

	out, err := Render(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "marketplace add anthropics/claude-plugins-official") {
		t.Error("expected marketplace add in output")
	}
	if !strings.Contains(out, "plugin install superpowers@claude-plugins-official") {
		t.Error("expected plugin install in output")
	}
}

func TestRender_RuntimeOrdering(t *testing.T) {
	cfg := &config.Config{
		Name:     "test",
		Agent:    "claude-code",
		Runtimes: map[string]string{"bun": "latest", "node": "22"},
	}

	out, err := Render(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rootIdx := strings.Index(out, "USER root")
	agentIdx := strings.Index(out, "USER agent")
	nodeIdx := strings.Index(out, "setup_22.x")
	bunIdx := strings.Index(out, "bun.sh/install")

	if nodeIdx < rootIdx || nodeIdx > agentIdx {
		t.Error("node (root block) should appear between USER root and USER agent")
	}
	if bunIdx < agentIdx {
		t.Error("bun (agent block) should appear after USER agent")
	}
}
