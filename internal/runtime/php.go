package runtime

import "fmt"

type PhpRuntime struct{}

func (r *PhpRuntime) Name() string { return "php" }

func (r *PhpRuntime) Validate(version string) error {
	if version == "" {
		return fmt.Errorf("php version cannot be empty")
	}
	return nil
}

func (r *PhpRuntime) RootBlock(version string) string {
	return fmt.Sprintf(`RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      php%s-cli php%s-mbstring php%s-xml php%s-curl \
      php%s-pgsql php%s-redis php%s-zip php%s-bcmath \
      php%s-intl php%s-gd php%s-readline \
 && rm -rf /var/lib/apt/lists/*
COPY --from=composer:latest /usr/bin/composer /usr/bin/composer`,
		version, version, version, version,
		version, version, version, version,
		version, version, version)
}

func (r *PhpRuntime) AgentBlock(version string) string { return "" }

func (r *PhpRuntime) EnvBlock(version string) string {
	return `ENV COMPOSER_HOME="/home/agent/.composer"
ENV COMPOSER_IGNORE_PLATFORM_REQS=1
ENV PATH="$COMPOSER_HOME/vendor/bin:$PATH"`
}
