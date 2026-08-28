package runtime

import (
	"fmt"
	"strings"
)

type GoRuntime struct{}

func (r *GoRuntime) Name() string { return "go" }

func (r *GoRuntime) Validate(version string) error {
	if version == "" {
		return fmt.Errorf("go version cannot be empty")
	}
	return nil
}

// RootBlock downloads and installs Go from go.dev.
//
// Go's release tarballs are only published under a full major.minor.patch
// filename — even the first release of a minor is "go1.23.0", never "go1.23".
// So a pinned major.minor like "1.23" (which has no such tarball and 404s) is
// resolved to that minor's latest patch at build time via the download JSON
// API. A version that already carries a patch (two dots, e.g. "1.23.4") is
// downloaded directly.
func (r *GoRuntime) RootBlock(version string) string {
	if strings.Count(version, ".") >= 2 {
		return fmt.Sprintf(`RUN ARCH=$(dpkg --print-architecture) \
 && curl -fsSL https://go.dev/dl/go%s.linux-$ARCH.tar.gz | tar -C /usr/local -xzf -`, version)
	}
	return fmt.Sprintf(`RUN ARCH=$(dpkg --print-architecture) \
 && GOVER=$(curl -fsSL "https://go.dev/dl/?mode=json&include=all" | grep -oE '"go%s(\.[0-9]+)?"' | tr -d '"' | sed 's/^go//' | sort -V | tail -1) \
 && test -n "$GOVER" \
 && curl -fsSL "https://go.dev/dl/go$GOVER.linux-$ARCH.tar.gz" | tar -C /usr/local -xzf -`, version)
}

func (r *GoRuntime) AgentBlock(version string) string { return "" }

func (r *GoRuntime) EnvBlock(version string) string {
	return `ENV GOPATH="/home/agent/go"
ENV PATH="$GOPATH/bin:/usr/local/go/bin:$PATH"`
}
