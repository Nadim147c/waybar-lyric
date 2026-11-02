package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Nadim147c/go-mpris"
	"github.com/Nadim147c/waybar-lyric/internal/config"
	"github.com/Nadim147c/waybar-lyric/internal/lyric"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/Nadim147c/waybar-lyric/internal/waybar"
	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"
)

// Execute is the main function for lyrics.
func Execute(cmd *cobra.Command, _ []string) error {
	if !config.Quiet {
		fmt.Fprintln(os.Stderr, cmd.Version)
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

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	ticker := time.NewTicker(config.UpdateInterval)
	defer ticker.Stop()

	var mprisPlayer *mpris.Player
	for mprisPlayer == nil {
		p, err := player.Select(conn)
		if err != nil {
			slog.Debug("Failed to select player", "error", err)
			time.Sleep(config.UpdateInterval)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}
		mprisPlayer = p
	}
	slog.Debug("Player selected", "player", mprisPlayer)

	signals := make(chan *dbus.Signal)

	go mpris.OnSignal(conn, signals)

	var lastWaybar *waybar.Waybar
	var info *player.Metadata
	var lastUpdate time.Time

	var doUpdate bool
	var counter uint64

	var lastServerPos time.Duration
	var lastServerFetch time.Time
	var estimatedPos time.Duration

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-signals:
			slog.Debug("Received player update signal")
			doUpdate = true
		case <-ticker.C:
			counter++
			doUpdate = time.Since(lastUpdate) > 2*time.Second
		}

		if doUpdate { //nolint
			lastUpdate = time.Now()
			mprisPlayer, err := player.Select(conn)
			if err != nil {
				slog.Error("Player not found!", "error", err)

				w := waybar.Zero
				if !w.Is(lastWaybar) {
					w.Encode()
					lastWaybar = w
				}

				continue
			}

			info, err = player.Parse(mprisPlayer)
			if err != nil {
				slog.Error("Failed to parse dbus mpris metadata", "error", err)
				w := waybar.Zero
				if !w.Is(lastWaybar) {
					w.Encode()
					lastWaybar = w
				}
				continue
			}
		}

		if info == nil {
			continue
		}

		slog.Debug(
			"PlayerInfo",
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

		lyrics, err := lyric.Store.Load(info.ID, false)
		if err != nil {
			w := waybar.ForPlayer(info)
			w.Alt = waybar.Getting
			w.Class = append(w.Class, waybar.Getting)
			w.Encode()
			mctx, cancel := NewMprisContext(ctx, info.ID, mprisPlayer)
			lyrics, err = lyric.GetLyrics(mctx, info)
			if err != nil {
				var scoreErr *models.LyricsMatchScoreError
				if errors.Is(err, models.ErrLyricsNotFound) ||
					errors.Is(err, models.ErrLyricsNotSynced) ||
					errors.As(err, &scoreErr) {
					slog.Info("Lyrics not available", "reason", err)
				} else {
					slog.Error(
						"Failed to get lyrics",
						"error", err,
						"lines", lyrics.Lines,
					)
				}
			}
			cancel()
		}

		lyrics.Metadata = info

		if err != nil || len(lyrics.Lines) == 0 {
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

		now := time.Now()
		serverPos := info.Position

		if info.Status == mpris.PlaybackPlaying {
			if serverPos != lastServerPos {
				lastServerPos = serverPos
				lastServerFetch = now
				estimatedPos = serverPos
			} else if !lastServerFetch.IsZero() {
				elapsed := now.Sub(lastServerFetch)
				estimatedPos = serverPos + elapsed
			} else {
				lastServerFetch = now
				estimatedPos = serverPos
			}
		} else {
			lastServerPos = serverPos
			lastServerFetch = now
			estimatedPos = serverPos
		}

		info.Position = estimatedPos

		var idx int
		for i, line := range lyrics.Lines {
			if info.Position <= line.Timestamp {
				break
			}
			idx = i
		}

		currentLyric := lyrics.Lines[idx]

		w := waybar.ForLyrics(lyrics, idx)
		w.Percentage = info.Percentage()

		if info.Status == mpris.PlaybackPaused {
			w.Paused(info)
			if !w.Is(lastWaybar) {
				slog.Info(
					"Lyrics",
					"line", currentLyric.Text,
					"line-time", currentLyric.Timestamp.String(),
					"position", info.Position.String(),
				)
				w.Encode()
				lastWaybar = w
			}
			continue
		}

		if currentLyric.Text == "" && len(currentLyric.Words) == 0 {
			w.SetText(fmt.Sprintf("%s - %s", info.Artist, info.Title))
			w.Alt = waybar.Music
		}

		if !w.Is(lastWaybar) {
			slog.Info(
				"Lyrics",
				"line", currentLyric.Text,
				"line-time", currentLyric.Timestamp.String(),
				"position", info.Position.String(),
			)
			w.Encode()
			lastWaybar = w
		}
	}
}
