package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var logoLines = []string{
	"  ██████╗ ██╗    ██╗   ██╗ ██████╗██╗  ██╗██╗  ██╗ ██████╗ ██╗     ███████╗",
	"  ██╔══██╗██║    ██║   ██║██╔════╝██║ ██╔╝██║  ██║██╔═══██╗██║     ██╔════╝",
	"  ██████╔╝██║    ██║   ██║██║     █████╔╝ ███████║██║   ██║██║     █████╗  ",
	"  ██╔══██╗██║    ╚██╗ ██╔╝██║     ██╔═██╗ ██╔══██║██║   ██║██║     ██╔══╝  ",
	"  ██████╔╝███████╗╚████╔╝ ╚██████╗██║  ██╗██║  ██║╚██████╔╝███████╗███████╗",
	"  ╚═════╝ ╚══════╝ ╚═══╝   ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚══════╝",
}

var logoColors = []lipgloss.Color{
	"#A855F7",
	"#B366F9",
	"#C084FC",
	"#D4A0FE",
	"#E0B8FF",
	"#ECD0FF",
}

func RenderLogo() string {
	var b strings.Builder
	b.WriteString("\n")
	for i, line := range logoLines {
		color := logoColors[0]
		if i < len(logoColors) {
			color = logoColors[i]
		}
		style := lipgloss.NewStyle().Foreground(color).Bold(true)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}
	return b.String()
}
