package runtime

import (
	"fmt"
	"strings"
)

var supportedPhpVersions = map[string]bool{
	"8.4": true,
	"8.5": true,
}

var defaultPhpExtensions = []string{
	"mbstring", "xml", "curl", "pgsql", "redis",
	"zip", "bcmath", "intl", "gd", "readline",
}

type PhpRuntime struct {
	Extensions []string
}

func (r *PhpRuntime) Name() string { return "php" }

func (r *PhpRuntime) Validate(version string) error {
	if version == "" {
		return fmt.Errorf("php version cannot be empty")
	}
	if !supportedPhpVersions[version] {
		versions := make([]string, 0, len(supportedPhpVersions))
		for v := range supportedPhpVersions {
			versions = append(versions, v)
		}
		return fmt.Errorf("unsupported php version %q: must be one of: %s", version, strings.Join(versions, ", "))
	}
	return nil
}

func (r *PhpRuntime) extensions() []string {
	if len(r.Extensions) == 0 {
		return defaultPhpExtensions
	}

	seen := make(map[string]bool, len(defaultPhpExtensions)+len(r.Extensions))
	result := make([]string, 0, len(defaultPhpExtensions)+len(r.Extensions))
	for _, ext := range defaultPhpExtensions {
		seen[ext] = true
		result = append(result, ext)
	}
	for _, ext := range r.Extensions {
		if !seen[ext] {
			result = append(result, ext)
		}
	}
	return result
}

func (r *PhpRuntime) RootBlock(version string) string {
	var b strings.Builder

	// Ubuntu 25.04 (Plucky) only ships up to PHP 8.4. For 8.5, pull from
	// Ubuntu 26.04 (Resolute) repos with APT pinning so only PHP and its
	// dependencies are upgraded.
	if version == "8.5" {
		b.WriteString(`RUN printf '%s\n' \
      'Types: deb' \
      'URIs: http://ports.ubuntu.com/ubuntu-ports/' \
      'Suites: resolute resolute-updates resolute-security' \
      'Components: main universe' \
      'Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg' \
    > /etc/apt/sources.list.d/resolute-php.sources \
 && printf '%s\n' \
      'Package: *' \
      'Pin: release n=resolute*' \
      'Pin-Priority: 100' \
      '' \
      'Package: php8.5* libc6 libc-bin libc-gconv-modules-extra libicu78' \
      'Pin: release n=resolute*' \
      'Pin-Priority: 500' \
    > /etc/apt/preferences.d/resolute-php
`)
	}

	exts := r.extensions()
	pkgs := make([]string, 0, len(exts)+1)
	pkgs = append(pkgs, fmt.Sprintf("php%s-cli", version))
	for _, ext := range exts {
		pkgs = append(pkgs, fmt.Sprintf("php%s-%s", version, ext))
	}

	b.WriteString(fmt.Sprintf("RUN apt-get update \\\n && apt-get install -y --no-install-recommends \\\n      %s \\\n && rm -rf /var/lib/apt/lists/*\nCOPY --from=composer:latest /usr/bin/composer /usr/bin/composer",
		strings.Join(pkgs, " ")))

	return b.String()
}

func (r *PhpRuntime) AgentBlock(version string) string { return "" }

func (r *PhpRuntime) EnvBlock(version string) string {
	return `ENV COMPOSER_HOME="/home/agent/.composer"
ENV COMPOSER_IGNORE_PLATFORM_REQS=1
ENV PATH="$COMPOSER_HOME/vendor/bin:$PATH"`
}
