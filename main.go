package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Nadim147c/go-mpris"
	"github.com/godbus/dbus/v5"
)

const (
	SleepTime = 500 * time.Millisecond
	Version   = "waybar-lyric v0.10.0 (https://github.com/Nadim147c/waybar-lyric)"
)

func truncate(input string) string {
	r := []rune(input)

	if len(r) <= MaxTextLength {
		return input
	}

	if MaxTextLength > 3 {
		return string(r[:MaxTextLength-3]) + "..."
	}

	return string(r[:MaxTextLength])
}

func main() {
	if PrintVersion {
		fmt.Fprint(os.Stderr, Version)
		return
	}

	if PrintInit {
		PrintSnippet()
		return
	}

	conn, err := dbus.SessionBus()
	if err != nil {
		slog.Error("Failed to create dbus connection", "error", err)
		return
	}

	var player *mpris.Player
	for player == nil {
		p, _, err := SelectPlayer(conn)
		if err == nil {
			slog.Debug("Failed to select player", "error", err)
			player = p
		}
	}
	slog.Debug("Player selected", "player", player)

	if ToggleState {
		slog.Info("Toggling player state")
		if err := player.PlayPause(); err != nil {
			slog.Error("Failed to toggle player state", "error", err)
		}
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Clean In memery lyrics cache every 10 minute
	go LyricStore.Cleanup(ctx, 10*time.Minute)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	playerSignal := make(chan *dbus.Signal)
	player.OnSignal(playerSignal)

	lyricTicker := time.NewTicker(SleepTime)
	defer lyricTicker.Stop()

	// Main loop
	fixedTicker := time.NewTicker(SleepTime)
	defer fixedTicker.Stop()

	var lastWaybar *Waybar

	instant := make(chan bool)
	go func() { instant <- true }()

	for {
		select {
		case <-ctx.Done():
			return // Clean exit on cancel
		case <-playerSignal:
			slog.Debug("Received player update signal")
		case <-instant:
		case <-lyricTicker.C:
		case <-fixedTicker.C:
		}

		player, parser, err := SelectPlayer(conn)
		if err != nil {
			slog.Error("Player not found!", "error", err)

			waybar := &Waybar{}
			if !waybar.Is(lastWaybar) {
				waybar.Encode()
				lastWaybar = waybar
			}

			continue
		}

		info, err := parser(player)
		if err != nil {
			slog.Error("Failed to parse dbus mpris metadata", "error", err)
			waybar := &Waybar{}
			if !waybar.Is(lastWaybar) {
				waybar.Encode()
				lastWaybar = waybar
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
			waybar := &Waybar{}
			if !waybar.Is(lastWaybar) {
				waybar.Encode()
				lastWaybar = waybar
			}
			continue
		}

		lyrics, err := GetLyrics(info)
		if err != nil {
			slog.Error("Failed to get lyrics", "error", err)
			waybar := info.Waybar()
			waybar.Alt = NoLyric
			if !waybar.Is(lastWaybar) {
				waybar.Encode()
				lastWaybar = waybar
			}

			continue
		}

		err = info.UpdatePosition(player)
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

		lyric := lyrics[idx]

		waybar := NewWaybar(lyrics, idx)
		if Detailed {
			waybar.Info = info
		}
		waybar.Percentage = info.Percentage()

		if info.Status == mpris.PlaybackPaused {
			waybar.Paused(info)
			if !waybar.Is(lastWaybar) {
				slog.Info("Lyrics",
					"line", lyric.Text,
					"line-time", lyric.Timestamp.String(),
					"position", info.Position.String(),
				)
				waybar.Encode()
				lastWaybar = waybar
			}
			continue
		}

		if lyric.Text == "" {
			waybar.Text = fmt.Sprintf("%s - %s", info.Artist, info.Title)
			waybar.Alt = Music
		}

		if !waybar.Is(lastWaybar) {
			slog.Info("Lyrics",
				"line", lyric.Text,
				"line-time", lyric.Timestamp.String(),
				"position", info.Position.String(),
			)
			waybar.Encode()
			lastWaybar = waybar
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
