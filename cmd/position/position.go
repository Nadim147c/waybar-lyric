package position

import (
	"fmt"
	"log/slog"

	"github.com/Nadim147c/waybar-lyric/internal/lyric"
	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/godbus/dbus/v5"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

var lyricsLine bool

func init() {
	Command.Flags().
		BoolVarP(&lyricsLine, "lyric", "l", lyricsLine, "Set player position to lyrics line number")
}

// Command is the position changer command
var Command = &cobra.Command{
	Use: "position [+]<position>[m/s/ms/ns]",
	Example: `  waybar-lyric position 20s # Set the player position to 20 seconds
  waybar-lyric position --lyric 1 # Set the player position to first lyrics line
  waybar-lyric position -- -10s # Set player position 10 seconds before the end`,
	Short: "Set position player position",
	Args:  cobra.ExactArgs(1),

	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		pos, err := cast.ToDurationE(args[0])
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

		if lyricsLine {
			info, err := parser(mp)
			if err != nil {
				return fmt.Errorf(
					"failed to parse player informations: %w",
					err,
				)
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

			lineNumber := int(pos)

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
		} else {
			if pos < 0 {
				slog.Debug(
					"Negative duration detected. Trying to substract from song length",
					"position",
					pos,
				)
				length, err := mp.GetLength()
				if err != nil {
					return fmt.Errorf("failed to song duration: %v", err)
				}
				pos = max(0, length+pos)
			}
		}

		slog.Info(
			"Setting player position",
			"player",
			mp.GetName(),
			"position",
			pos,
		)
		if err := mp.SetPosition(pos); err != nil {
			return fmt.Errorf("failed to set player position: %w", err)
		}

		return nil
	},
}
