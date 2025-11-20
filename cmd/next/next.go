package next

import (
	"fmt"
	"log/slog"

	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"
)

// Command is the track skipper command
var Command = &cobra.Command{
	Use:     "next",
	Example: `waybar-lyric next # Skips to the next track`,
	Short:   "Skip to the next track",
	Args:    cobra.ExactArgs(0),

	DisableFlagsInUseLine: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		conn, err := dbus.SessionBus()
		if err != nil {
			return fmt.Errorf("failed to create dbus connection: %w", err)
		}
		slog.Debug("Created dbus session bus")

		mp, _, err := player.Select(conn)
		if err != nil {
			return fmt.Errorf("failed to select player: %w", err)
		}

		slog.Info("Going to next track", "player", mp.GetName())
		if err := mp.Next(); err != nil {
			slog.Error("Failed to set volume", "error", err)
			return err
		}

		return nil
	},
}
