package runtime

type Runtime interface {
	Name() string
	Validate(version string) error
	RootBlock(version string) string
	AgentBlock(version string) string
	EnvBlock(version string) string
}

var registry = map[string]Runtime{
	"node":   &NodeRuntime{},
	"pnpm":   &PnpmRuntime{},
	"bun":    &BunRuntime{},
	"python": &PythonRuntime{},
	"go":     &GoRuntime{},
	"php":    &PhpRuntime{},
	"rust":   &RustRuntime{},
}

func Get(name string) Runtime {
	return registry[name]
}
