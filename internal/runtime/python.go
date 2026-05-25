package runtime

import "fmt"

type PythonRuntime struct{}

func (r *PythonRuntime) Name() string { return "python" }

func (r *PythonRuntime) Validate(version string) error {
	if version == "" {
		return fmt.Errorf("python version cannot be empty")
	}
	return nil
}

func (r *PythonRuntime) RootBlock(version string) string {
	return fmt.Sprintf(`RUN apt-get update \
 && apt-get install -y --no-install-recommends python%s python%s-venv python%s-dev \
 && rm -rf /var/lib/apt/lists/*`, version, version, version)
}

func (r *PythonRuntime) AgentBlock(version string) string { return "" }
func (r *PythonRuntime) EnvBlock(version string) string   { return "" }
