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

// corepackHome is a shared cache. Corepack defaults to a per-user directory, so
// a root-time `corepack prepare` is invisible to the agent user and pnpm gets
// re-downloaded on first use.
const corepackHome = "/usr/local/share/corepack"

func (r *PnpmRuntime) RootBlock(version string) string {
	return fmt.Sprintf(`RUN mkdir -p %[1]s \
 && COREPACK_HOME=%[1]s corepack enable \
 && COREPACK_HOME=%[1]s corepack prepare pnpm@%[2]s --activate \
 && chown -R agent %[1]s`, corepackHome, version)
}

func (r *PnpmRuntime) AgentBlock(version string) string { return "" }

func (r *PnpmRuntime) EnvBlock(version string) string {
	return fmt.Sprintf("ENV COREPACK_HOME=%s", corepackHome)
}
