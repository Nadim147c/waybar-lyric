package importcmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Nadim147c/waybar-lyric/internal/lyric"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"
)

// Command is the track skipper command.
var Command = &cobra.Command{
	Use: "import",
	Example: `
  # Import as lyrics for current track
  waybar-lyric import /path/to/lyrics.lrc

  # Download and import a lyrics
  waybar-lyric import <(curl url://to/my/song/lyrics)
  `,
	Short: "Manually import lyrics for current",
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

		info, err := player.Parse(mp)
		if err != nil {
			return fmt.Errorf("failed to parse player informations: %w", err)
		}

		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer f.Close()

		lines, err := provider.ParseReader(f)
		if err != nil {
			return err
		}

		m := models.Lyrics{Metadata: info, Lines: lines} //nolint:exhaustruct

		return lyric.Store.Save(m)
	},
}
