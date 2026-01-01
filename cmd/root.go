package cmd

import (
	"errors"
	"log/slog"
	"math"
	"os"
	"path/filepath"

	initcmd "github.com/Nadim147c/waybar-lyric/cmd/init"
	"github.com/Nadim147c/waybar-lyric/cmd/next"
	"github.com/Nadim147c/waybar-lyric/cmd/playpause"
	"github.com/Nadim147c/waybar-lyric/cmd/position"
	"github.com/Nadim147c/waybar-lyric/cmd/previous"
	"github.com/Nadim147c/waybar-lyric/cmd/seek"
	"github.com/Nadim147c/waybar-lyric/cmd/volume"
	"github.com/Nadim147c/waybar-lyric/internal/config"
	"github.com/carapace-sh/carapace"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

func init() {
	Command.Flags().
		BoolVarP(&config.Compact, "compact", "c", config.Compact, "Output only text content on each line")
	Command.Flags().
		BoolVarP(&config.Detailed, "detailed", "d", config.Detailed, "Put detailed player information in output")
	Command.Flags().
		BoolVarP(&config.LyricOnly, "lyric-only", "l", config.LyricOnly, "Display only lyrics in text output")
	Command.Flags().
		BoolVarP(&config.NoTooltip, "no-tooltip", "T", config.NoTooltip, "Disable tooltip from output")
	Command.Flags().
		BoolVarP(&config.PrintInit, "init", "i", config.PrintInit, "Display JSON snippet for waybar/config.jsonc")
	Command.Flags().
		BoolVarP(&config.PrintVersion, "version", "V", config.PrintVersion, "Display waybar-lyric version information")
	Command.Flags().
		BoolVarP(&config.ToggleState, "toggle", "t", config.ToggleState, "Toggle player state between pause and resume")
	Command.Flags().
		BoolVarP(&config.ExperimentalChromiumSupport, "experimental-chromium-support", "x", config.ToggleState, "Enable experimental chromium support.")
	Command.Flags().
		IntVarP(&config.BreakTooltip, "break-tooltip", "b", config.BreakTooltip, "Break long lines in tooltip")
	Command.Flags().
		IntVarP(&config.MaxTextLength, "max-length", "m", config.MaxTextLength, "Set maximum character length for lyrics text")
	Command.Flags().
		IntVarP(&config.TooltipLines, "tooltip-lines", "L", config.TooltipLines, "Set maximum number of lines in waybar tooltip")
	Command.Flags().
		StringVarP(&config.FilterProfanityType, "filter-profanity", "f", config.FilterProfanityType, "Filter profanity from lyrics (values: full, partial)")
	Command.Flags().
		StringVarP(&config.TooltipColor, "tooltip-color", "C", config.TooltipColor, "Set color for inactive lyrics lines")

	Command.Flags().MarkDeprecated("init", "use 'waybar-lyric init'.")
	Command.Flags().MarkDeprecated("toggle", "use 'waybar-lyric play-pause'.")

	Command.MarkFlagsMutuallyExclusive("toggle", "init")

	Command.PersistentFlags().
		BoolP("help", "h", false, "Display help for waybar-lyric")
	Command.PersistentFlags().
		BoolVarP(&config.Quiet, "quiet", "q", config.Quiet, "Suppress all log output")
	Command.PersistentFlags().
		BoolVarP(&config.Verbose, "verbose", "v", config.Verbose, "Enable verbose logging")
	Command.PersistentFlags().
		StringVarP(&config.LogFilePath, "log-file", "o", config.LogFilePath, "Specify file path for saving logs")

	Command.MarkFlagsMutuallyExclusive("quiet", "verbose")
	Command.MarkFlagsMutuallyExclusive("quiet", "log-file")

	Command.AddCommand(initcmd.Command)
	Command.AddCommand(next.Command)
	Command.AddCommand(playpause.Command)
	Command.AddCommand(position.Command)
	Command.AddCommand(previous.Command)
	Command.AddCommand(seek.Command)
	Command.AddCommand(volume.Command)

	comp := carapace.Gen(Command)
	comp.Standalone()
	comp.FlagCompletion(carapace.ActionMap{
		"log-file": carapace.ActionFiles(),
	})
}

var logFile *os.File

// Command is root command for waybar
var Command = &cobra.Command{
	Use:          "waybar-lyric",
	Short:        "A waybar module for song lyrics",
	SilenceUsage: true,
	RunE:         Execute,
	PersistentPostRunE: func(_ *cobra.Command, _ []string) error {
		if logFile != nil {
			return logFile.Close()
		}
		return nil
	},
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if config.ToggleState {
			defer func() {
				cmd.RemoveCommand(playpause.Command)
				playpause.Command.Execute()
				os.Exit(0)
			}()
		}

		if config.PrintInit {
			defer func() {
				cmd.RemoveCommand(initcmd.Command)
				initcmd.Command.Execute()
				os.Exit(0)
			}()
		}

		switch config.FilterProfanityType {
		case "":
			config.FilterProfanity = false
		case "full", "partial":
			config.FilterProfanity = true
		default:
			return errors.New(
				"Profanity filter must one of 'full' or 'partial'",
			)
		}

		if config.TooltipLines < 4 {
			return errors.New("Tooltip lines limit must be at least 4")
		}

		var level log.Level

		if config.Quiet {
			level = math.MaxInt // ignore all logs
		}

		if config.Verbose {
			level = log.DebugLevel
		}

		handler := slog.New(log.NewWithOptions(os.Stderr, log.Options{
			Level: log.Level(level),
		}))

		if config.LogFilePath == "" {
			slog.SetDefault(handler)
			return nil
		}

		os.MkdirAll(filepath.Dir(config.LogFilePath), 0o755)

		file, err := os.OpenFile(
			config.LogFilePath,
			os.O_CREATE|os.O_APPEND|os.O_WRONLY,
			0o666,
		)
		if err != nil {
			slog.SetDefault(handler)
			slog.Error("Failed to open log-file", "error", err)
			return err
		}

		slog.SetDefault(slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{
			Level:     slog.LevelDebug, // file logging always verbose
			AddSource: true,
		})))
		logFile = file

		return nil
	},
}
