package init

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

//go:embed init.jsonc
var snippet string

// Command is the init command.
var Command = &cobra.Command{
	Use:          "init",
	Short:        "Print json snippet for waybar confg",
	SilenceUsage: true,
	// Disable flag parsing
	DisableFlagParsing: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		fmt.Fprint(os.Stderr, "Put the following object in your waybar config:\n\n")
		_, err := fmt.Print(snippet) //nolint:forbidigo
		return err
	},
}
