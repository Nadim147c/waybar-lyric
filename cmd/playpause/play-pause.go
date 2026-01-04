package playpause

import (
	"log/slog"

	"github.com/Nadim147c/go-mpris"
	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"
)

// Command is the play-pause command
var Command = &cobra.Command{
	Use:          "play-pause",
	Short:        "Toggle play-pause state",
	SilenceUsage: true,
	// Disable flag parsing
	DisableFlagParsing: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		conn, err := dbus.SessionBus()
		if err != nil {
			slog.Error("Failed to create dbus connection", "error", err)
			return err
		}

		var mp *mpris.Player
		for mp == nil {
			p, err := player.Select(conn)
			if err == nil {
				slog.Debug("Failed to select player", "error", err)
				mp = p
			}
		}
		slog.Debug("Player selected", "player", mp)

		slog.Info("Toggling player state")
		if err := mp.PlayPause(); err != nil {
			slog.Error("Failed to toggle player state", "error", err)
			return err
		}
		return nil
	},
}
