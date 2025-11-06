package seek

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/lyric"
	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/godbus/dbus/v5"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

var lyricsLine bool

func init() {
	Command.Flags().
		BoolVarP(&lyricsLine, "lyric", "l", lyricsLine, "Set player seek to lyrics line number")
}

// Command is the position seeker command
var Command = &cobra.Command{
	Use: "seek [+/-]<offset>[m/s/ms/ns]",
	Example: `  waybar-lyric seek 20s # Seeks 20 seconds ahead
  waybar-lyric seek --lyric 1 # Seeks to next lyric line
  waybar-lyric seek -- -10s # Go back 20 seconds`,
	Short: "Seek player position",
	Args:  cobra.ExactArgs(1),

	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		offset, err := cast.ToDurationE(args[0])
		if err != nil {
			return fmt.Errorf("failed to convert duration: %w", err)
		}

		conn, err := dbus.SessionBus()
		if err != nil {
			return fmt.Errorf("failed to create dbus connection: %w", err)
		}
		slog.Debug("Created dbus session bus")

		mp, parser, err := player.Select(conn)
		if err != nil {
			return fmt.Errorf("failed to select player: %w", err)
		}
		slog.Debug("Selected player", "player", mp.GetName())

		if !lyricsLine {
			slog.Info(
				"Seeking player position",
				"player",
				mp.GetName(),
				"offset",
				offset,
			)
			if err := mp.Seek(offset); err != nil {
				return fmt.Errorf("failed to set player position: %w", err)
			}
			return nil
		}

		info, err := parser(mp)
		if err != nil {
			return fmt.Errorf("failed to parse player informations: %w", err)
		}
		slog.Debug(
			"Parsed player information",
			"title",
			info.Title,
			"artist",
			info.Artist,
		)

		lyrics, err := lyric.GetLyrics(cmd.Context(), info)
		if err != nil {
			return fmt.Errorf("failed to fetch lyrics: %w", err)
		}
		lines := lyrics.Lines
		slog.Debug("Fetched lyrics", "line-count", len(lines))

		info.UpdatePosition(mp)

		var currentIndex int
		for i, line := range lines {
			if info.Position <= line.Timestamp {
				break
			}
			currentIndex = i
		}

		slog.Debug("Current Line", "number", currentIndex)

		lineNumber := currentIndex + int(offset)

		var pos time.Duration
		if lineNumber < 0 {
			idx := len(lines) + lineNumber
			if idx < 0 {
				return fmt.Errorf(
					"line number out of range (line-count=%d, requested=%d)",
					len(lines),
					lineNumber,
				)
			}
			slog.Debug(
				"Setting position from negative index",
				"line-number",
				lineNumber,
				"resolved-index",
				idx,
			)
			pos = lines[idx].Timestamp
		} else {
			if lineNumber > len(lines) {
				return fmt.Errorf(
					"line number out of range (line-count=%d, requested=%d)",
					len(lines),
					lineNumber,
				)
			}
			slog.Debug(
				"Setting position from positive line number",
				"line-number",
				lineNumber,
			)
			pos = lines[lineNumber].Timestamp
		}

		slog.Info(
			"Setting player position",
			"player",
			mp.GetName(),
			"position",
			pos,
			"line-number",
			lineNumber,
		)
		if err := mp.SetPosition(pos); err != nil {
			return fmt.Errorf("failed to set player position: %w", err)
		}

		return nil
	},
}
