package volume

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/godbus/dbus/v5"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

// Command is the volume changer command
var Command = &cobra.Command{
	Use: "volume",
	Example: `  waybar-lyric volume 20% # Set player volume to 20%
  waybar-lyric volume 0.5 # Set player volume to 50%
  waybar-lyric volume +10% # Increase player volume by 10%
  waybar-lyric volume -5% # Decrease player volume by 5%`,
	Short: "Set player volume",
	Args:  cobra.ExactArgs(1),

	DisableFlagsInUseLine: true,
	RunE: func(_ *cobra.Command, args []string) error {
		volStr := args[0]
		var volume float64
		relative := false
		operation := 1.0 // 1 for add, -1 for subtract

		// Check for relative operation prefix
		if strings.HasPrefix(volStr, "+") || strings.HasPrefix(volStr, "-") {
			relative = true
			if strings.HasPrefix(volStr, "-") {
				operation = -1.0
			}
			volStr = volStr[1:] // Remove the prefix
		}

		// Parse the volume value
		if strings.HasSuffix(volStr, "%") {
			vol, err := cast.ToFloat64E(strings.TrimSuffix(volStr, "%"))
			if err != nil {
				slog.Error("Failed to convert volume to float", "error", err)
				return fmt.Errorf("failed convert volume to float: %v", err)
			}
			volume = vol / 100
		} else {
			vol, err := cast.ToFloat64E(volStr)
			if err != nil {
				slog.Error("Failed to convert volume to float", "error", err)
				return fmt.Errorf("failed convert volume to float: %v", err)
			}
			volume = vol
		}

		// Validate absolute volume range
		if !relative && (volume < 0 || volume > 1) {
			slog.Error(
				"Volume is out of range. Volume value must be between 0-1 or 0-100%",
				"got",
				volStr,
			)
			return fmt.Errorf("volume is out of range. volume=%.2f", volume)
		}

		conn, err := dbus.SessionBus()
		if err != nil {
			return fmt.Errorf("failed to create dbus connection: %w", err)
		}
		slog.Debug("Created dbus session bus")

		mp, _, err := player.Select(conn)
		if err != nil {
			return fmt.Errorf("failed to select player: %w", err)
		}

		slog.Debug("Selected player", "player", mp.GetName())

		// Handle relative volume adjustment
		if relative {
			currentVol, err := mp.GetVolume()
			if err != nil {
				slog.Error("Failed to get current volume", "error", err)
				return fmt.Errorf("failed to get current volume: %w", err)
			}
			volume = currentVol + (operation * volume)
			// Clamp the volume between 0 and 1 after adjustment
			if volume < 0 {
				volume = 0
			} else if volume > 1 {
				volume = 1
			}
		}

		slog.Info("Setting player volume", "volume", volume)
		if err := mp.SetVolume(volume); err != nil {
			slog.Error("Failed to set volume", "error", err)
			return err
		}

		return nil
	},
}
