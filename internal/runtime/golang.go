package runtime

import "fmt"

type GoRuntime struct{}

func (r *GoRuntime) Name() string { return "go" }

func (r *GoRuntime) Validate(version string) error {
	if version == "" {
		return fmt.Errorf("go version cannot be empty")
	}
	return nil
}

func (r *GoRuntime) RootBlock(version string) string {
	return fmt.Sprintf(`RUN ARCH=$(dpkg --print-architecture) \
 && curl -fsSL https://go.dev/dl/go%s.linux-$ARCH.tar.gz | tar -C /usr/local -xzf -`, version)
}

func (r *GoRuntime) AgentBlock(version string) string { return "" }

func (r *GoRuntime) EnvBlock(version string) string {
	return `ENV GOPATH="/home/agent/go"
ENV PATH="$GOPATH/bin:/usr/local/go/bin:$PATH"`
}
