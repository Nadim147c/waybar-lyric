package seek

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
		BoolVarP(&lyricsLine, "lyric", "l", lyricsLine, "Set player seek to lyrics line number")
}

func seek(p *mpris.Player, offset time.Duration) error {
	slog.Info("Seeking player position",
		"player", p.GetName(),
		"offset", offset,
	)
	if err := p.Seek(offset); err != nil {
		return fmt.Errorf("failed to seek player position: %w", err)
	}
	return nil
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

// seekExactOffset seeks by an absolute offset duration.
// Supports negative offsets to seek backwards.
func seekExactOffset(p *mpris.Player, s string) error {
	offset, err := cast.ToDurationE(s)
	if err != nil {
		return fmt.Errorf("failed to convert duration: %w", err)
	}
	return seek(p, offset)
}

// seekPercentOffset seeks by a percentage of total track length.
// Input should be a number followed by '%' (e.g., '20%' to seek 20% of track
// length).
func seekPercentOffset(p *mpris.Player, s string, length time.Duration) error {
	percStr := strings.TrimSuffix(s, "%")
	perc, err := cast.ToFloat64E(percStr)
	if err != nil {
		return fmt.Errorf("failed to convert percent: %w", err)
	}
	offset := time.Duration(float64(length) * perc / 100)
	return seek(p, offset)
}

// seekLyricLine seeks to a lyric line relative to the current position.
// Positive offsets seek forward, negative offsets seek backward (e.g., 1 seeks
// to next line, -1 seeks to previous line).
func seekLyricLine(
	p *mpris.Player,
	info *player.Metadata,
	lines lyric.Lines,
	s string,
) error {
	offset, err := cast.ToIntE(s)
	if err != nil {
		return fmt.Errorf("failed to convert lyric offset: %w", err)
	}

	slog.Debug("Parsed player information",
		"title", info.Title,
		"artist", info.Artist,
	)

	info.UpdatePosition(p)
	slog.Debug("Current position", "position", info.Position)

	// Find current lyric line based on position
	var currentIndex int
	for i, line := range lines {
		if info.Position <= line.Timestamp {
			break
		}
		currentIndex = i
	}

	slog.Debug("Current line", "index", currentIndex)

	lineNumber := currentIndex + offset
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

		slog.Debug("Seeking to negative index",
			"line-number", lineNumber,
			"resolved-index", idx,
		)
		pos = lines[idx].Timestamp
		return setPosition(p, pos)
	}

	if lineNumber >= len(lines) {
		return fmt.Errorf(
			"line number out of range (line-count=%d, requested=%d)",
			len(lines), lineNumber,
		)
	}

	slog.Debug("Seeking to positive line number",
		"line-number", lineNumber,
	)
	pos = lines[lineNumber].Timestamp

	slog.Info("Setting player position from lyric seek",
		"player", p.GetName(),
		"position", pos,
		"line-number", lineNumber,
	)

	return setPosition(p, pos)
}

// Command is the position seeker command
var Command = &cobra.Command{
	Use: "seek [+/-]<offset>[m/s/ms/ns/%]",
	Example: `
  waybar-lyric seek 20s # Seeks 20 seconds ahead
  waybar-lyric seek --lyric 1 # Seeks to next lyric line
  waybar-lyric seek -- -10s # Go back 10 seconds
  waybar-lyric seek -- 20% # Seek 20% of total length forward
  `,
	Short: "Seek player position",
	Args:  cobra.ExactArgs(1),

	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		input := args[0]

		// Route to appropriate seek method based on input format and flags
		if strings.HasSuffix(input, "%") {
			length, err := mp.GetLength()
			if err != nil {
				return fmt.Errorf("failed to get song duration: %v", err)
			}
			return seekPercentOffset(mp, input, length)
		}

		if !lyricsLine {
			return seekExactOffset(mp, input)
		}

		// Fetch lyrics for line-based seeking
		info, err := parser(mp)
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

		slog.Debug("Fetched lyrics", "line-count", len(lyrics.Lines))

		return seekLyricLine(mp, info, lyrics.Lines, input)
	},
}
