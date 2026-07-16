package export

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Nadim147c/waybar-lyric/internal/lyric"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/formats/lrc"
	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"
)

var format = "json"

func init() {
	Command.Flags().StringVarP(&format, "format", "f", format, "Lyrics file format (lrc or json)")
}

// Command is the track skipper command.
var Command = &cobra.Command{
	Use: "export",
	Example: `
  # Export lyrics of current track
  waybar-lyric export

  # Export lyrics of given id
  waybar-lyric export <id>
  `,
	Short: "Manually import lyrics for current",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var id string
		if len(args) == 1 {
			id = args[0]
		} else {
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
				return fmt.Errorf("failed to parse player information: %w", err)
			}
			id = info.ID
		}

		lyrics, err := lyric.Store.Load(id)
		if err != nil {
			return err
		}

		switch format {
		case "json":
			return json.NewEncoder(cmd.OutOrStdout()).Encode(lyrics)
		case "lrc":
			_, err := cmd.OutOrStdout().Write(lrc.Render(lyrics))
			return err
		case "ttml":
			return errors.New("ttml export is not supported")
		default:
			return fmt.Errorf("unknown lyrics format: %v", format)
		}
	},
}
