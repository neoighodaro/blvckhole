package runtime

import (
	"fmt"
	"strings"
)

type NodeRuntime struct{}

func (r *NodeRuntime) Name() string { return "node" }

func (r *NodeRuntime) Validate(version string) error {
	if version == "" {
		return fmt.Errorf("node version cannot be empty")
	}
	return nil
}

func (r *NodeRuntime) RootBlock(version string) string {
	if strings.Contains(version, ".") {
		return fmt.Sprintf(`RUN ARCH=$(dpkg --print-architecture) \
 && if [ "$ARCH" = "amd64" ]; then ARCH="x64"; fi \
 && curl -fsSL https://nodejs.org/dist/v%s/node-v%s-linux-$ARCH.tar.gz | tar -xz -C /usr/local --strip-components=1 \
 && rm -rf /usr/local/lib/node_modules/npm /usr/local/bin/npm /usr/local/bin/npx`, version, version)
	}
	return fmt.Sprintf(`RUN curl -fsSL https://deb.nodesource.com/setup_%s.x | bash - \
 && apt-get install -y --no-install-recommends nodejs \
 && rm -rf /var/lib/apt/lists/* \
 && rm -rf /usr/bin/npm /usr/bin/npx /usr/lib/node_modules/npm`, version)
}

func (r *NodeRuntime) AgentBlock(version string) string { return "" }
func (r *NodeRuntime) EnvBlock(version string) string   { return "" }
