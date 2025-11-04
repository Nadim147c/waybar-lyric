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
		Foreground(lipgloss.Color("#0CB37F")).
		Margin(1, 0, 0, 3)
	versionStyle := lipgloss.NewRenderer(os.Stderr).
		NewStyle().
		Background(lipgloss.Color("#6B50FF")).
		Blink(true).
		Padding(0, 1).
		Margin(0, 3, 1, 3)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		asciiStyle.Render(ascii),
		versionStyle.Render(version),
	)
}
