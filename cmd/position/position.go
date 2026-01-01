package position

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Nadim147c/go-mpris"
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

func setPosition(p *mpris.Player, pos time.Duration) error {
	slog.Info("Setting player position",
		"player", p.GetName(),
		"position", pos,
	)
	if err := p.SetPosition(pos); err != nil {
		return fmt.Errorf("failed to set player position: %w", err)
	}
	return nil
}

// setExactPosition sets the player position to an absolute duration.
// Supports negative durations to specify time from the end (e.g., -10s means
// 10s before end).
func setExactPosition(p *mpris.Player, s string, length time.Duration) error {
	pos, err := cast.ToDurationE(s)
	if err != nil {
		return fmt.Errorf("failed to convert duration: %w", err)
	}
	if pos < 0 {
		pos = max(length+pos, 0)
	}
	return setPosition(p, pos)
}

// setPercentPosition sets the player position to a percentage of total track
// length. Input should be a number followed by '%' (e.g., '20%' for 20% through
// the track).
func setPercentPosition(p *mpris.Player, s string, length time.Duration) error {
	percStr := strings.TrimSuffix(s, "%")
	perc, err := cast.ToFloat64E(percStr)
	if err != nil {
		return fmt.Errorf("failed to convert percent: %w", err)
	}
	if perc < 0 || perc > 100 {
		return fmt.Errorf(
			"percentage must be between 0 and 100, got %.1f",
			perc,
		)
	}
	pos := time.Duration(float64(length) * perc / 100)
	return setPosition(p, pos)
}

// setLyricPosition sets the player position to a specific lyric line.
// Positive indices count from the start (0-indexed), negative indices count
// from the end (-1 is the last line).
func setLyricPosition(p *mpris.Player, lines lyric.Lines, s string) error {
	lineNumber, err := cast.ToIntE(s)
	if err != nil {
		return fmt.Errorf("failed to convert lyric index: %w", err)
	}

	var pos time.Duration

	if lineNumber >= 0 {
		if lineNumber >= len(lines) {
			return fmt.Errorf(
				"line number out of range line-count=%d, requested=%d",
				len(lines),
				lineNumber,
			)
		}

		slog.Debug("Setting position from positive line number",
			"line-number", lineNumber,
		)
		pos = lines[lineNumber].Timestamp

		return setPosition(p, pos)
	}

	idx := len(lines) + lineNumber
	if idx < 0 {
		return fmt.Errorf(
			"line number out of range line-count=%d, requested=%d",
			len(lines), lineNumber,
		)
	}

	slog.Debug(
		"Setting position from negative index",
		"line-number", lineNumber,
		"resolved-index", idx,
	)
	pos = lines[idx].Timestamp

	return setPosition(p, pos)
}

// Command is the position changer command
var Command = &cobra.Command{
	Use: "position",
	Example: `
  waybar-lyric position 20s # Set the player position to 20 seconds
  waybar-lyric position --lyric 1 # Set the player position to first lyrics line
  waybar-lyric position -- -10s # Set player position 10 seconds before the end
  waybar-lyric position -- 20% # Set player position to 20% of total length
  `,
	Short: "Set player position",
	Args:  cobra.ExactArgs(1),

	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := dbus.SessionBus()
		if err != nil {
			return fmt.Errorf("failed to create dbus connection: %w", err)
		}
		slog.Debug("Created dbus session bus")

		mp, err := player.Select(conn)
		if err != nil {
			return fmt.Errorf("failed to select player: %w", err)
		}
		slog.Debug("Selected player", "player", mp.GetName())

		length, err := mp.GetLength()
		if err != nil {
			return fmt.Errorf("failed to get song duration: %v", err)
		}

		input := args[0]

		// Route to appropriate position setter based on input format
		if strings.HasSuffix(input, "%") {
			return setPercentPosition(mp, input, length)
		}

		if !lyricsLine {
			return setExactPosition(mp, input, length)
		}

		// Fetch lyrics for line-based positioning
		info, err := player.Parse(mp)
		if err != nil {
			return fmt.Errorf("failed to parse player informations: %w", err)
		}

		slog.Debug("Parsed player information",
			"title", info.Title,
			"artist", info.Artist,
		)

		lyrics, err := lyric.GetLyrics(cmd.Context(), info)
		if err != nil {
			return fmt.Errorf("failed to fetch lyrics: %w", err)
		}

		return setLyricPosition(mp, lyrics.Lines, input)
	},
}
