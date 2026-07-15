package embedded

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// stubBin writes an executable stub script into dir under name.
func stubBin(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0755); err != nil {
		t.Fatal(err)
	}
}

// runBridge writes the embedded bridge.sh to a temp file and runs it with a
// PATH of stubbed socat/setsid/sudo/pgrep, returning the log file contents.
func runBridge(t *testing.T, extraEnv map[string]string, args ...string) (logOut string, socatArgs string, err error) {
	t.Helper()
	dir := t.TempDir()

	script := filepath.Join(dir, "bridge.sh")
	if err := os.WriteFile(script, BridgeSh, 0755); err != nil {
		t.Fatal(err)
	}

	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0755)
	socatLog := filepath.Join(dir, "socat.args")
	stubBin(t, binDir, "socat", `echo "$@" >> `+socatLog+`; exit 0`)
	stubBin(t, binDir, "setsid", `[ "$1" = "-f" ] && shift; exec "$@"`)
	stubBin(t, binDir, "sudo", `exec "$@"`)
	stubBin(t, binDir, "pgrep", `exit 1`)

	logFile := filepath.Join(dir, "bridge.log")

	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"BLVCKHOLE_BRIDGE_LOG="+logFile,
	)
	for k, v := range extraEnv {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	runErr := cmd.Run()

	log, _ := os.ReadFile(logFile)
	sa, _ := os.ReadFile(socatLog)
	return string(log), string(sa), runErr
}

func TestBridgeScript_StartsSocatFirst(t *testing.T) {
	dir := t.TempDir()
	hosts := filepath.Join(dir, "hosts")
	os.WriteFile(hosts, []byte("127.0.0.1 localhost\n"), 0644)

	log, socatArgs, err := runBridge(t, map[string]string{"SANDBOX_HOSTS_FILE": hosts}, "pgsql", "5432", "53432")
	if err != nil {
		t.Fatalf("bridge.sh failed: %v\nlog:\n%s", err, log)
	}
	if !strings.Contains(socatArgs, "TCP-LISTEN:5432,bind=127.0.0.1") {
		t.Errorf("socat not started with the listen mapping; got: %q", socatArgs)
	}
	if !strings.Contains(socatArgs, "TCP:host.docker.internal:53432") {
		t.Errorf("socat not pointed at the host port; got: %q", socatArgs)
	}
	if !strings.Contains(log, "bridge started") {
		t.Errorf("expected bridge-started log, got:\n%s", log)
	}
	// Writable /etc/hosts: the alias is added, no env override needed.
	got, _ := os.ReadFile(hosts)
	if !strings.Contains(string(got), "127.0.0.1  pgsql") {
		t.Errorf("expected /etc/hosts alias for pgsql, got:\n%s", got)
	}
}

func TestBridgeScript_EnvFallbackWhenHostsReadOnly(t *testing.T) {
	dir := t.TempDir()
	hosts := filepath.Join(dir, "hosts")
	os.WriteFile(hosts, []byte("127.0.0.1 localhost\n"), 0444)
	envFile := filepath.Join(dir, "persistent.sh")
	os.WriteFile(envFile, []byte("# persistent\n"), 0644)

	log, _, err := runBridge(t, map[string]string{
		"SANDBOX_HOSTS_FILE": hosts,
		"CLAUDE_ENV_FILE":    envFile,
	}, "pgsql", "5432", "53432", "DB_HOST")
	if err != nil {
		t.Fatalf("bridge.sh failed: %v\nlog:\n%s", err, log)
	}

	got, _ := os.ReadFile(envFile)
	if !strings.Contains(string(got), "export DB_HOST=127.0.0.1") {
		t.Errorf("expected DB_HOST override in env file when /etc/hosts is read-only, got:\n%s", got)
	}
}

func TestBridgeScript_ReadOnlyHostsNoEnvIsNotFatal(t *testing.T) {
	dir := t.TempDir()
	hosts := filepath.Join(dir, "hosts")
	os.WriteFile(hosts, []byte("127.0.0.1 localhost\n"), 0444)

	log, _, err := runBridge(t, map[string]string{"SANDBOX_HOSTS_FILE": hosts}, "pgsql", "5432", "53432")
	if err != nil {
		t.Fatalf("bridge.sh should not fail when hosts is read-only and no env set: %v\nlog:\n%s", err, log)
	}
	if !strings.Contains(log, "bridge started") {
		t.Errorf("bridge should still come up, got:\n%s", log)
	}
}
