package runtime

import "fmt"

type BunRuntime struct{}

func (r *BunRuntime) Name() string { return "bun" }

func (r *BunRuntime) Validate(version string) error {
	if version == "" {
		return fmt.Errorf("bun version cannot be empty")
	}
	return nil
}

func (r *BunRuntime) RootBlock(version string) string { return "" }

func (r *BunRuntime) AgentBlock(version string) string {
	return fmt.Sprintf(`RUN curl -fsSL https://bun.sh/install | bash -s "bun-v%s"`, version)
}

func (r *BunRuntime) EnvBlock(version string) string {
	return `ENV BUN_INSTALL="/home/agent/.bun"
ENV PATH="$BUN_INSTALL/bin:$PATH"`
}
