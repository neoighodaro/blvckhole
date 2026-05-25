package embedded

import (
	_ "embed"
)

//go:embed theme.json
var ThemeJSON []byte

//go:embed settings.json
var SettingsJSON []byte

//go:embed dashboard.json
var DashboardJSON []byte

//go:embed Dockerfile.tmpl
var DockerfileTmpl string

//go:embed bashrc.sh
var BashrcSh []byte

//go:embed bash_aliases.sh
var BashAliasesSh []byte
