package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Nadim147c/go-mpris"
	"github.com/Nadim147c/waybar-lyric/internal/config"
	"github.com/Nadim147c/waybar-lyric/internal/lyric"
	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/Nadim147c/waybar-lyric/internal/waybar"
	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"
)

// SleepTime is the time for main fixed loop
const SleepTime = 500 * time.Millisecond

// Execute is the main function for lyrics
func Execute(cmd *cobra.Command, _ []string) error {
	if !config.Quiet {
		PrintASCII()
	}

	if config.PrintVersion {
		fmt.Fprint(os.Stderr, config.Version)
		return nil
	}

	conn, err := dbus.SessionBus()
	if err != nil {
		slog.Error("Failed to create dbus connection", "error", err)
		return nil
	}

	var mprisPlayer *mpris.Player
	for mprisPlayer == nil {
		p, _, err := player.Select(conn)
		if err != nil {
			slog.Debug("Failed to select player", "error", err)
			time.Sleep(SleepTime)
		}
		mprisPlayer = p
	}
	slog.Debug("Player selected", "player", mprisPlayer)

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Clean In memery lyrics cache every 10 minute
	go lyric.Store.Cleanup(ctx, 10*time.Minute)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	playerSignal := make(chan *dbus.Signal)
	mprisPlayer.OnSignal(playerSignal)

	lyricTicker := time.NewTicker(SleepTime)
	defer lyricTicker.Stop()

	// Main loop
	fixedTicker := time.NewTicker(SleepTime)
	defer fixedTicker.Stop()

	var lastWaybar *waybar.Waybar

	instant := make(chan bool)
	go func() { instant <- true }()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-playerSignal:
			slog.Debug("Received player update signal")
		case <-instant:
		case <-lyricTicker.C:
		case <-fixedTicker.C:
		}

		mprisPlayer, parser, err := player.Select(conn)
		if err != nil {
			slog.Error("Player not found!", "error", err)

			w := waybar.Zero
			if !w.Is(lastWaybar) {
				w.Encode()
				lastWaybar = w
			}

			continue
		}

		info, err := parser(mprisPlayer)
		if err != nil {
			slog.Error("Failed to parse dbus mpris metadata", "error", err)
			w := waybar.Zero
			if !w.Is(lastWaybar) {
				w.Encode()
				lastWaybar = w
			}
			continue
		}

		slog.Debug("PlayerInfo",
			"id", info.ID,
			"title", info.Title,
			"artist", info.Artist,
			"album", info.Album,
			"position", info.Position.String(),
			"length", info.Length.String(),
		)

		if info.Status == mpris.PlaybackStopped {
			slog.Info("Player is stopped")
			w := waybar.Zero
			if !w.Is(lastWaybar) {
				w.Encode()
				lastWaybar = w
			}
			continue
		}

		lyrics, err := lyric.GetLyrics(info)
		if err != nil {
			slog.Error("Failed to get lyrics", "error", err)
			w := waybar.ForPlayer(info)
			w.Alt = waybar.NoLyric
			if !w.Is(lastWaybar) {
				w.Encode()
				lastWaybar = w
			}

			continue
		}

		err = info.UpdatePosition(mprisPlayer)
		if err != nil {
			slog.Error("Failed to update position", "error", err)
			continue
		}

		var idx int
		for i, line := range lyrics {
			if info.Position <= line.Timestamp {
				break
			}
			idx = i
		}

		currentLyric := lyrics[idx]

		w := waybar.ForLyrics(lyrics, idx)
		if config.Detailed {
			w.Info = info
		}
		w.Percentage = info.Percentage()

		if info.Status == mpris.PlaybackPaused {
			w.Paused(info)
			if !w.Is(lastWaybar) {
				slog.Info("Lyrics",
					"line", currentLyric.Text,
					"line-time", currentLyric.Timestamp.String(),
					"position", info.Position.String(),
				)
				w.Encode()
				lastWaybar = w
			}
			continue
		}

		if currentLyric.Text == "" {
			w.SetText(fmt.Sprintf("%s - %s", info.Artist, info.Title))
			w.Alt = waybar.Music
		}

		if !w.Is(lastWaybar) {
			slog.Info("Lyrics",
				"line", currentLyric.Text,
				"line-time", currentLyric.Timestamp.String(),
				"position", info.Position.String(),
			)
			w.Encode()
			lastWaybar = w
		}

		if len(lyrics) > idx+1 {
			n := lyrics[idx+1]
			d := n.Timestamp - info.Position
			if d <= 0 {
				slog.Warn("Negative sleep time",
					"duration", d.String(),
					"position", info.Position.String(),
					"next", n.Timestamp.String(),
				)
				continue
			}
			slog.Debug("Sleep",
				"duration", d.String(),
				"position", info.Position.String(),
				"next", n.Timestamp.String(),
			)
			lyricTicker.Reset(d)
		}
	}
}
