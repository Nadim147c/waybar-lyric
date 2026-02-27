package astext

import (
	"context"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/spf13/cast"
)

// Provider is a lyrics provider that gets lyrics from touan's asText metadata.
var Provider = provider.NewProvider("asText metadata parser",
	func(_ context.Context, metadata *player.Metadata) (lyrics models.Lyrics, err error) {
		lyrics.Metadata = metadata
		asText, ok := metadata.Metadata["xesam:asText"]
		if !ok && asText.Value() == "" {
			return lyrics, models.ErrLyricsNotFound
		}

		text, err := cast.ToStringE(asText.Value())
		if err != nil {
			return
		}

		lyrics.Lines, err = provider.ParseText(text)
		return
	})
