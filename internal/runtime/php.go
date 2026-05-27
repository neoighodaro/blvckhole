package runtime

import (
	"fmt"
	"strings"
)

var supportedPhpVersions = map[string]bool{
	"8.4": true,
	"8.5": true,
}

type PhpRuntime struct{}

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

	b.WriteString(fmt.Sprintf(`RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      php%s-cli php%s-mbstring php%s-xml php%s-curl \
      php%s-pgsql php%s-redis php%s-zip php%s-bcmath \
      php%s-intl php%s-gd php%s-readline \
 && rm -rf /var/lib/apt/lists/*
COPY --from=composer:latest /usr/bin/composer /usr/bin/composer`,
		version, version, version, version,
		version, version, version, version,
		version, version, version))

	return b.String()
}

func (r *PhpRuntime) AgentBlock(version string) string { return "" }

func (r *PhpRuntime) EnvBlock(version string) string {
	return `ENV COMPOSER_HOME="/home/agent/.composer"
ENV COMPOSER_IGNORE_PLATFORM_REQS=1
ENV PATH="$COMPOSER_HOME/vendor/bin:$PATH"`
}
