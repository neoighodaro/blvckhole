package runtime

import "fmt"

type PnpmRuntime struct{}

func (r *PnpmRuntime) Name() string { return "pnpm" }

func (r *PnpmRuntime) Validate(version string) error {
	if version == "" {
		return fmt.Errorf("pnpm version cannot be empty")
	}
	return nil
}

func (r *PnpmRuntime) RootBlock(version string) string {
	return fmt.Sprintf(`RUN corepack enable && corepack prepare pnpm@%s --activate`, version)
}

func (r *PnpmRuntime) AgentBlock(version string) string { return "" }
func (r *PnpmRuntime) EnvBlock(version string) string   { return "" }
