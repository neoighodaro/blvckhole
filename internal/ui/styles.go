package ui

import "github.com/charmbracelet/lipgloss"

var (
	Purple = lipgloss.Color("#A855F7")
	Violet = lipgloss.Color("#C084FC")
	Green  = lipgloss.Color("#A8DB8F")
	Red    = lipgloss.Color("#FF6B6B")
	Gray   = lipgloss.Color("#6B7280")
	White  = lipgloss.Color("#F9FAFB")
	Dim    = lipgloss.Color("#4B5563")

	Bold    = lipgloss.NewStyle().Bold(true)
	Success = lipgloss.NewStyle().Foreground(Green)
	Error   = lipgloss.NewStyle().Foreground(Red)
	Info    = lipgloss.NewStyle().Foreground(Gray)
	Accent  = lipgloss.NewStyle().Foreground(Purple)
)
