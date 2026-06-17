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

// RootBlock is empty: Python is installed per-user via uv (see AgentBlock).
//
// The previous apt approach (`apt-get install python<ver>`) only worked when the
// requested minor version happened to match the base image's distro default —
// Ubuntu ships exactly one python3 minor per release, so a pinned version like
// 3.12 has no apt package and the build fails. uv installs any pinned version
// from python-build-standalone regardless of the distro.
func (r *PythonRuntime) RootBlock(version string) string { return "" }

// AgentBlock installs uv, then installs the requested Python version with
// --default so that `python` and `python3` (not just the versioned
// `python<ver>`) are placed on the PATH. Both uv and the Python executables land
// in ~/.local/bin (see EnvBlock).
func (r *PythonRuntime) AgentBlock(version string) string {
	return fmt.Sprintf(`RUN curl -LsSf https://astral.sh/uv/install.sh | sh \
 && /home/agent/.local/bin/uv python install %s --default`, version)
}

func (r *PythonRuntime) EnvBlock(version string) string {
	return `ENV PATH="/home/agent/.local/bin:$PATH"`
}
