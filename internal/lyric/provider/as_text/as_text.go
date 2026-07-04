package astext

import (
	"context"

	"github.com/Nadim147c/go-mpris"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/formats/lrc"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/spf13/cast"
)

// Provider is a lyrics provider that gets lyrics from touan's asText metadata.
var Provider = provider.NewProvider("asText metadata parser",
	func(_ context.Context, metadata *player.Metadata) (lyrics models.Lyrics, err error) {
		lyrics.Metadata = metadata
		lyrics.NoCache = true

		asText, err := metadata.Metadata.Get(mpris.KeyAsText)
		if err != nil {
			return lyrics, err
		}

		text, err := cast.ToStringE(asText)
		if err != nil {
			return
		}

		lyrics.Lines, err = lrc.ParseText(text)
		return
	})
