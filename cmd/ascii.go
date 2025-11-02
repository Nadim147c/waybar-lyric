package cmd

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/Nadim147c/waybar-lyric/internal/config"
	"github.com/charmbracelet/lipgloss"
)

//go:embed ascii.txt
var ascii string

// PrintASCII prints ASCII logo with rainbow colors
func PrintASCII() {
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
	fmt.Fprintln(os.Stderr, asciiStyle.Render(ascii))
	fmt.Fprintln(os.Stderr, versionStyle.Render(config.Version))
}
