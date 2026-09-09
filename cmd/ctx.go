package cmd

import (
	"context"
	"time"

	"github.com/Nadim147c/go-mpris"
	"github.com/Nadim147c/waybar-lyric/internal/config"
	"github.com/Nadim147c/waybar-lyric/internal/player"
)

func NewMprisContext(parent context.Context, id string, mp *mpris.Player) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)

	go func() {
		ticker := time.NewTicker(config.UpdateInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				info, err := player.Parse(mp)
				if err != nil {
					continue
				}
				if info.ID != id {
					cancel()
					return
				}
			}
		}
	}()

	return ctx, cancel
}
