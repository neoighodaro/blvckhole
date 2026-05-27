package template

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/neoighodaro/blvckhole/internal/config"
	"github.com/neoighodaro/blvckhole/internal/embedded"
	"github.com/neoighodaro/blvckhole/internal/runtime"
)

type templateData struct {
	Packages    []string
	RootBlocks  []string
	AgentBlocks []string
	EnvBlocks   []string
	Plugins     config.ClaudePlugins
}

func Render(cfg *config.Config) (string, error) {
	data := templateData{
		Packages: cfg.Packages,
		Plugins:  cfg.Claude.Plugins,
	}

	names := make([]string, 0, len(cfg.Runtimes))
	for name := range cfg.Runtimes {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		version := cfg.Runtimes[name]
		r := runtime.Get(name)
		if r == nil {
			return "", fmt.Errorf("unknown runtime: %s", name)
		}
		if name == "php" {
			if php, ok := r.(*runtime.PhpRuntime); ok {
				php.Extensions = cfg.Php.Extensions
			}
		}
		if block := r.RootBlock(version); block != "" {
			data.RootBlocks = append(data.RootBlocks, block)
		}
		if block := r.AgentBlock(version); block != "" {
			data.AgentBlocks = append(data.AgentBlocks, block)
		}
		if block := r.EnvBlock(version); block != "" {
			data.EnvBlocks = append(data.EnvBlocks, block)
		}
	}

	funcMap := template.FuncMap{
		"join": strings.Join,
	}

	tmpl, err := template.New("Dockerfile").Funcs(funcMap).Parse(embedded.DockerfileTmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse Dockerfile template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render Dockerfile: %w", err)
	}

	return buf.String(), nil
}

func Build(cfg *config.Config) error {
	dockerfile, err := Render(cfg)
	if err != nil {
		return err
	}

	kitDir := cfg.KitDir()

	if err := os.WriteFile(filepath.Join(kitDir, "Dockerfile"), []byte(dockerfile), 0644); err != nil {
		return err
	}

	cmd := exec.Command("docker", "build", "-t", cfg.SandboxImageName(), kitDir)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker build failed:\n%s", string(output))
	}

	return nil
}

func LoadTemplate(cfg *config.Config) error {
	tarPath := fmt.Sprintf("/tmp/%s.tar", cfg.SandboxImageName())

	save := exec.Command("docker", "image", "save", cfg.SandboxImageName(), "-o", tarPath)
	if output, err := save.CombinedOutput(); err != nil {
		return fmt.Errorf("docker image save failed:\n%s", string(output))
	}

	load := exec.Command("sbx", "template", "load", tarPath)
	if output, err := load.CombinedOutput(); err != nil {
		exec.Command("rm", "-f", tarPath).Run()
		return fmt.Errorf("sbx template load failed:\n%s", string(output))
	}

	exec.Command("rm", "-f", tarPath).Run()
	return nil
}
