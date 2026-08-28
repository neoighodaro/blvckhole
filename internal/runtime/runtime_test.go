package runtime

import (
	"strings"
	"testing"
)

func TestNodeRuntime_RootBlock(t *testing.T) {
	r := Get("node")
	block := r.RootBlock("22")
	if !strings.Contains(block, "setup_22.x") {
		t.Errorf("expected nodesource 22.x setup, got:\n%s", block)
	}
	if !strings.Contains(block, "apt-get install") {
		t.Errorf("expected apt-get install, got:\n%s", block)
	}
}

func TestNodeRuntime_AgentBlock(t *testing.T) {
	r := Get("node")
	block := r.AgentBlock("22")
	if block != "" {
		t.Errorf("node should have no agent block, got:\n%s", block)
	}
}

func TestBunRuntime_AgentBlock(t *testing.T) {
	r := Get("bun")
	block := r.AgentBlock("1.2.0")
	if !strings.Contains(block, "bun-v1.2.0") {
		t.Errorf("expected bun version in install script, got:\n%s", block)
	}
}

func TestBunRuntime_RootBlock(t *testing.T) {
	r := Get("bun")
	block := r.RootBlock("latest")
	if block != "" {
		t.Errorf("bun should have no root block, got:\n%s", block)
	}
}

func TestBunRuntime_EnvBlock(t *testing.T) {
	r := Get("bun")
	block := r.EnvBlock("latest")
	if !strings.Contains(block, "BUN_INSTALL") {
		t.Errorf("expected BUN_INSTALL in env block, got:\n%s", block)
	}
}

func TestPythonRuntime_AgentBlock(t *testing.T) {
	r := Get("python")
	if block := r.RootBlock("3.12"); block != "" {
		t.Errorf("expected empty root block (python installs via uv in agent block), got:\n%s", block)
	}
	block := r.AgentBlock("3.12")
	if !strings.Contains(block, "astral.sh/uv/install.sh") {
		t.Errorf("expected uv installer in agent block, got:\n%s", block)
	}
	if !strings.Contains(block, "uv python install 3.12 --default") {
		t.Errorf("expected pinned python install in agent block, got:\n%s", block)
	}
}

func TestGoRuntime_RootBlock(t *testing.T) {
	r := Get("go")

	// A pinned major.minor has no tarball of its own; it must resolve the
	// latest patch at build time rather than hit go1.23.linux (which 404s).
	minor := r.RootBlock("1.23")
	if strings.Contains(minor, "go1.23.linux") {
		t.Errorf("major.minor must not download go1.23.linux directly, got:\n%s", minor)
	}
	if !strings.Contains(minor, `grep -oE '"go1.23(\.[0-9]+)?"'`) {
		t.Errorf("expected latest-patch resolution for 1.23, got:\n%s", minor)
	}

	// A full major.minor.patch downloads directly.
	full := r.RootBlock("1.23.4")
	if !strings.Contains(full, "go1.23.4.linux") {
		t.Errorf("expected go1.23.4 direct download, got:\n%s", full)
	}
}

func TestGoRuntime_EnvBlock(t *testing.T) {
	r := Get("go")
	block := r.EnvBlock("1.23")
	if !strings.Contains(block, "GOPATH") {
		t.Errorf("expected GOPATH in env block, got:\n%s", block)
	}
}

func TestPhpRuntime_RootBlock(t *testing.T) {
	r := Get("php")
	block := r.RootBlock("8.4")
	if !strings.Contains(block, "php8.4-cli") {
		t.Errorf("expected php8.4-cli in root block, got:\n%s", block)
	}
	if !strings.Contains(block, "composer") {
		t.Errorf("expected composer in root block, got:\n%s", block)
	}
}

func TestRustRuntime_AgentBlock(t *testing.T) {
	r := Get("rust")
	block := r.AgentBlock("stable")
	if !strings.Contains(block, "rustup.rs") {
		t.Errorf("expected rustup in agent block, got:\n%s", block)
	}
}

func TestGet_UnknownRuntime(t *testing.T) {
	r := Get("java")
	if r != nil {
		t.Error("expected nil for unknown runtime")
	}
}

func TestAllRuntimes_ValidateAcceptsVersionStrings(t *testing.T) {
	cases := map[string]string{
		"node": "22", "bun": "latest", "python": "3.12",
		"go": "1.23", "php": "8.4", "rust": "stable",
	}
	for name, ver := range cases {
		r := Get(name)
		if r == nil {
			t.Errorf("Get(%q) returned nil", name)
			continue
		}
		if err := r.Validate(ver); err != nil {
			t.Errorf("%s.Validate(%q) = %v", name, ver, err)
		}
	}
}

func TestPnpmRuntime_SharesCorepackCacheWithAgent(t *testing.T) {
	r := Get("pnpm")

	root := r.RootBlock("11.5.3")
	if !strings.Contains(root, "corepack prepare pnpm@11.5.3 --activate") {
		t.Errorf("expected corepack prepare for the pinned version, got:\n%s", root)
	}
	if !strings.Contains(root, "COREPACK_HOME="+corepackHome) {
		t.Errorf("corepack must populate the shared cache as root, got:\n%s", root)
	}
	if !strings.Contains(root, "chown -R agent "+corepackHome) {
		t.Errorf("shared cache must be owned by the agent user, got:\n%s", root)
	}

	// Without this the agent user gets its own empty cache and re-downloads pnpm.
	if env := r.EnvBlock("11.5.3"); env != "ENV COREPACK_HOME="+corepackHome {
		t.Errorf("agent must inherit the shared cache, got:\n%s", env)
	}
}
