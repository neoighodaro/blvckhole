package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type SandboxInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Agent  string `json:"agent"`
	Ports  string `json:"ports"`
}

func IsRunning(name string) bool {
	cmd := exec.Command("sbx", "ls", "--quiet")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) == name {
			return true
		}
	}
	return false
}

func Exists(name string) bool {
	cmd := exec.Command("sbx", "ls", "--quiet")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) == name {
			return true
		}
	}
	return false
}

func Create(name, template, kitDir, agent, workDir string) error {
	args := []string{"create", "--template", template, "--name", name, "--kit", kitDir, agent, workDir}
	cmd := exec.Command("sbx", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Stop(name string) error {
	cmd := exec.Command("sbx", "stop", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Remove(name string) error {
	cmd := exec.Command("sbx", "rm", "-f", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Run(name string, extraArgs ...string) error {
	args := []string{"run", name}
	if len(extraArgs) > 0 {
		args = append(args, "--")
		args = append(args, extraArgs...)
	}
	cmd := exec.Command("sbx", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Exec(name string, interactive bool, command ...string) error {
	args := []string{"exec"}
	if interactive {
		args = append(args, "-it")
	}
	args = append(args, name, "--")
	args = append(args, command...)
	cmd := exec.Command("sbx", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ExecSilent(name string, command ...string) (string, error) {
	args := []string{"exec", name, "--"}
	args = append(args, command...)
	cmd := exec.Command("sbx", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func AllowNetwork(name string, domains []string) error {
	if len(domains) == 0 {
		return nil
	}
	joined := strings.Join(domains, ",")
	cmd := exec.Command("sbx", "policy", "allow", "network", name, joined)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func PublishPort(name string, mapping string) error {
	cmd := exec.Command("sbx", "ports", name, "--publish", mapping)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Status(name string) (*SandboxInfo, error) {
	cmd := exec.Command("sbx", "ls", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("sbx ls failed: %w", err)
	}

	var sandboxes []SandboxInfo
	if err := json.Unmarshal(output, &sandboxes); err != nil {
		var wrapped map[string]json.RawMessage
		if err2 := json.Unmarshal(output, &wrapped); err2 != nil {
			return nil, fmt.Errorf("failed to parse sbx ls output: %w", err)
		}
		for _, v := range wrapped {
			if err2 := json.Unmarshal(v, &sandboxes); err2 == nil {
				break
			}
		}
	}

	for _, s := range sandboxes {
		if s.Name == name {
			return &s, nil
		}
	}

	return nil, nil
}

func TemplateLoaded(name string) bool {
	cmd := exec.Command("sbx", "template", "ls")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), name)
}
