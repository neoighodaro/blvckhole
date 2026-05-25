package runtime

import "fmt"

type RustRuntime struct{}

func (r *RustRuntime) Name() string { return "rust" }

func (r *RustRuntime) Validate(version string) error {
	if version == "" {
		return fmt.Errorf("rust version cannot be empty")
	}
	return nil
}

func (r *RustRuntime) RootBlock(version string) string { return "" }

func (r *RustRuntime) AgentBlock(version string) string {
	return `RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y`
}

func (r *RustRuntime) EnvBlock(version string) string {
	return `ENV PATH="/home/agent/.cargo/bin:$PATH"`
}
