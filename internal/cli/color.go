package cli

import "github.com/charmbracelet/lipgloss"

var (
	primaryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C00"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00CFCF"))
	silentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#808080"))
	textStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#4A4A4A", Dark: "#C0C0C0"})
)

func Primary(text string) string { return primaryStyle.Render(text) }
func Error(text string) string   { return errorStyle.Render(text) }
func Warning(text string) string { return warningStyle.Render(text) }
func Info(text string) string    { return infoStyle.Render(text) }
func Silent(text string) string  { return silentStyle.Render(text) }
func Text(text string) string    { return textStyle.Render(text) }
