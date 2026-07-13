package backend

import (
	"strings"
	"testing"

	"github.com/neoighodaro/blvckhole/internal/config"
)

func TestGetSbx(t *testing.T) {
	b := Get("sbx")
	if b == nil {
		t.Fatal("Get(\"sbx\") = nil, want SbxBackend")
	}
	if b.Name() != "sbx" {
		t.Fatalf("Name() = %q, want \"sbx\"", b.Name())
	}
}

func TestParseSandboxListPlainArray(t *testing.T) {
	out := []byte(`[{"id":"abc","name":"myapp","status":"running","agent":"claude","workspaces":["/w"]}]`)
	got, err := parseSandboxList(out)
	if err != nil {
		t.Fatalf("parseSandboxList: %v", err)
	}
	if len(got) != 1 || got[0].Name != "myapp" || got[0].Status != "running" {
		t.Fatalf("parseSandboxList = %+v", got)
	}
}

func TestParseSandboxListWrappedObject(t *testing.T) {
	out := []byte(`{"sandboxes":[{"id":"abc","name":"myapp","status":"stopped","agent":"claude","workspaces":[]}]}`)
	got, err := parseSandboxList(out)
	if err != nil {
		t.Fatalf("parseSandboxList: %v", err)
	}
	if len(got) != 1 || got[0].Status != "stopped" {
		t.Fatalf("parseSandboxList = %+v", got)
	}
}

func TestParseSandboxListInvalid(t *testing.T) {
	if _, err := parseSandboxList([]byte("not json")); err == nil {
		t.Fatal("parseSandboxList(invalid) = nil error, want error")
	}
}

func TestSbxEnsureAvailableMissing(t *testing.T) {
	t.Setenv("PATH", "")
	err := Get("sbx").EnsureAvailable()
	if err == nil {
		t.Fatal("EnsureAvailable with empty PATH = nil, want error")
	}
	if !strings.Contains(err.Error(), "sbx is not installed") {
		t.Fatalf("error = %q, want mention of sbx install", err.Error())
	}
}

func TestJqSettingsFilterIncludesPlugins(t *testing.T) {
	cfg := &config.Config{}
	cfg.Claude.Plugins.Install = []string{"foo@bar"}
	f := jqSettingsFilter(cfg)
	if !strings.Contains(f, `.enabledPlugins["foo@bar"] = true`) {
		t.Fatalf("jqSettingsFilter missing plugin enable: %s", f)
	}
	if !strings.Contains(f, ".enabledPlugins = (.enabledPlugins // {})") {
		t.Fatalf("jqSettingsFilter missing enabledPlugins init: %s", f)
	}
}

func TestJqMergeFilterMergesDocumentsFirst(t *testing.T) {
	cfg := &config.Config{}
	if !strings.HasPrefix(jqMergeFilter(cfg), ".[0] * .[1] | ") {
		t.Fatalf("jqMergeFilter = %q, want merge prefix", jqMergeFilter(cfg))
	}
	if jqNoMergeFilter(cfg) != jqSettingsFilter(cfg) {
		t.Fatal("jqNoMergeFilter must equal jqSettingsFilter")
	}
}
