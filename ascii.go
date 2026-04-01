package main

import (
	_ "embed"
	"os"

	"github.com/charmbracelet/lipgloss"
)

//go:embed ascii.txt
var ascii string

// GetASCII prints ASCII logo with rainbow colors
func GetASCII() string {
	asciiStyle := lipgloss.NewRenderer(os.Stderr).
		NewStyle().
		Foreground(lipgloss.Color("#46d1ac")).
		Margin(1, 0, 0, 3)
	versionStyle := lipgloss.NewRenderer(os.Stderr).
		NewStyle().
		Background(lipgloss.Color("#d43650")).
		Padding(0, 1).
		Margin(0, 3, 1, 3)
	donateStyle := lipgloss.NewRenderer(os.Stderr).
		NewStyle().
		Bold(true).
		Margin(0, 3, 1, 3)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		asciiStyle.Render(ascii),
		versionStyle.Render(version),
		donateStyle.Render("If you find this useful, consider donating?"),
	)
}
