package runtime

import "fmt"

type NodeRuntime struct{}

func (r *NodeRuntime) Name() string { return "node" }

func (r *NodeRuntime) Validate(version string) error {
	if version == "" {
		return fmt.Errorf("node version cannot be empty")
	}
	return nil
}

func (r *NodeRuntime) RootBlock(version string) string {
	return fmt.Sprintf(`RUN curl -fsSL https://deb.nodesource.com/setup_%s.x | bash - \
 && apt-get install -y --no-install-recommends nodejs \
 && rm -rf /var/lib/apt/lists/* \
 && rm -rf /usr/bin/npm /usr/bin/npx /usr/lib/node_modules/npm`, version)
}

func (r *NodeRuntime) AgentBlock(version string) string { return "" }
func (r *NodeRuntime) EnvBlock(version string) string   { return "" }
